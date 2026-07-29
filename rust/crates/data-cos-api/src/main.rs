use std::collections::HashSet;
use std::env;
use std::fs;
use std::io::Write;
use std::net::SocketAddr;
use std::path::Path;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};

use axum::extract::{DefaultBodyLimit, State};
use axum::http::{Request, StatusCode, header};
use axum::middleware::{Next, from_fn_with_state};
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post};
use axum::{Json, Router};
use data_cos_core::{
    CosGroupingRequest, CosRecord, CosStep1Request, CosStep2Request, filter_cos_step1,
    filter_cos_step2, group_wavelengths_flat_top, group_wavelengths_greedy,
    group_wavelengths_huang_meng, group_wavelengths_optimal,
};
use data_cos_data_adapter::{
    CosBatchFileInfo, DataFetchExtractRequest, DataFetchExtractResponse, ExtractionMode,
    extract_data, list_cos_batch_files, load_cos_records_from_excel,
};
use serde::{Deserialize, Serialize};
use tower_http::cors::{Any, CorsLayer};
use tower_http::trace::TraceLayer;
use tracing::{error, info, warn};

const REQUEST_BODY_LIMIT_BYTES: usize = 128 * 1024 * 1024;

#[derive(Clone)]
struct AppState {
    auth_token: Option<String>,
    cos_cache: Arc<RwLock<Vec<CosRecord>>>,
    /// Cached step1 result so step2/grouping can reference it server-side
    /// instead of receiving the full record array over HTTP.
    step1_cache: Arc<RwLock<Vec<CosRecord>>>,
}

#[derive(Debug)]
struct ApiError {
    status: StatusCode,
    message: String,
}

impl ApiError {
    fn bad_request(message: impl Into<String>) -> Self {
        Self {
            status: StatusCode::BAD_REQUEST,
            message: message.into(),
        }
    }

    fn internal(message: impl Into<String>) -> Self {
        Self {
            status: StatusCode::INTERNAL_SERVER_ERROR,
            message: message.into(),
        }
    }

    fn unauthorized() -> Self {
        Self {
            status: StatusCode::UNAUTHORIZED,
            message: String::from("unauthorized"),
        }
    }
}

impl IntoResponse for ApiError {
    fn into_response(self) -> Response {
        let payload = ErrorBody {
            error: self.message,
        };
        (self.status, Json(payload)).into_response()
    }
}

#[derive(Debug, Serialize)]
struct ErrorBody {
    error: String,
}

#[derive(Debug, Serialize)]
struct HealthResponse {
    status: &'static str,
    service: &'static str,
}

#[derive(Debug, Deserialize)]
struct LoadCosRequest {
    file_path: String,
}

#[derive(Debug, Deserialize)]
struct ListCosFilesRequest {
    directory: Option<String>,
}

#[derive(Debug, Serialize)]
struct LoadCosResponse {
    total: usize,
}

#[derive(Debug, Serialize)]
struct ListCosFilesResponse {
    files: Vec<CosBatchFileInfo>,
    total: usize,
}

#[derive(Debug, Deserialize)]
struct CosStep1Payload {
    params: CosStep1Request,
}

/// Step2 no longer requires the client to upload the full record array.
/// It references the server-side step1 cache instead.
#[derive(Debug, Deserialize)]
struct CosStep2LightPayload {
    params: CosStep2Request,
    /// Optional client-side records override (backward compat). If empty,
    /// the server uses the cached step1 result.
    records: Option<Vec<CosRecord>>,
}

/// Grouping also references server-side step1 cache by default.
#[derive(Debug, Deserialize)]
struct CosGroupLightPayload {
    params: CosGroupingRequest,
    records: Option<Vec<CosRecord>>,
}

#[derive(Debug, Serialize)]
struct CosStep1Response {
    records: Vec<CosRecord>,
    total: usize,
}

#[derive(Debug, Serialize)]
struct CosStep2Response {
    records: Vec<CosRecord>,
    total: usize,
}

#[derive(Debug, Deserialize)]
struct CosStep4ExtractPayload {
    records: Vec<CosRecord>,
    measurements: Option<Vec<String>>,
    current_points: Option<Vec<f64>>,
    chip_default_root: Option<String>,
    chip_default_roots: Option<Vec<String>>,
}

#[derive(Debug, Serialize)]
struct CosGroupResponse {
    groups: Vec<Vec<CosRecord>>,
    remaining: Vec<CosRecord>,
    group_count: usize,
    remaining_count: usize,
}

#[derive(Debug, Serialize)]
struct DataFetchResponse {
    records: Vec<data_cos_core::DataFetchRecord>,
    errors: Vec<String>,
    infos: Vec<String>,
    total: usize,
}

#[derive(Debug, Deserialize)]
struct ExtractionDiagnosticsRequest {
    roots: Vec<String>,
}

#[derive(Debug, Serialize)]
struct ExtractionDiagnosticCheck {
    code: &'static str,
    label: &'static str,
    status: &'static str,
    path: Option<String>,
    detail: String,
    os_error_code: Option<i32>,
}

#[derive(Debug, Serialize)]
struct ExtractionDiagnosticsResponse {
    process_id: u32,
    executable_path: String,
    cache_directory: String,
    checks: Vec<ExtractionDiagnosticCheck>,
    failed: usize,
    warnings: usize,
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "data_cos_api=info,tower_http=info".into()),
        )
        .init();

    let auth_token = env::var("DATA_COS_API_TOKEN").ok();
    let state = AppState {
        auth_token,
        cos_cache: Arc::new(RwLock::new(Vec::new())),
        step1_cache: Arc::new(RwLock::new(Vec::new())),
    };

    let app = Router::new()
        .route("/health", get(health))
        .route("/api/v1/data-fetch/extract", post(data_fetch_extract))
        .route(
            "/api/v1/data-fetch/diagnostics",
            post(data_fetch_diagnostics),
        )
        .route("/api/v1/cos-filter/files", post(cos_list_files))
        .route("/api/v1/cos-filter/load", post(cos_load))
        .route("/api/v1/cos-filter/step1", post(cos_step1))
        .route("/api/v1/cos-filter/step2", post(cos_step2))
        .route("/api/v1/cos-filter/group/greedy", post(cos_group_greedy))
        .route("/api/v1/cos-filter/group/optimal", post(cos_group_optimal))
        .route(
            "/api/v1/cos-filter/group/flat-top",
            post(cos_group_flat_top),
        )
        .route(
            "/api/v1/cos-filter/group/huang-meng",
            post(cos_group_huang_meng),
        )
        .route("/api/v1/cos-filter/step4/extract", post(cos_step4_extract))
        .layer(from_fn_with_state(state.clone(), auth_middleware))
        .layer(
            CorsLayer::new()
                .allow_origin(Any)
                .allow_headers(Any)
                .allow_methods(Any),
        )
        .layer(DefaultBodyLimit::max(REQUEST_BODY_LIMIT_BYTES))
        .layer(TraceLayer::new_for_http())
        .with_state(state);

    let port = env::var("DATA_COS_API_PORT")
        .ok()
        .and_then(|value| value.parse::<u16>().ok())
        .unwrap_or(9002);
    let addr = SocketAddr::from(([127, 0, 0, 1], port));

    info!("data-cos-api listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr)
        .await
        .expect("failed to bind tcp listener");

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await
        .expect("server error");
}

async fn shutdown_signal() {
    let _ = tokio::signal::ctrl_c().await;
    info!("shutdown signal received");
}

async fn auth_middleware(
    State(state): State<AppState>,
    req: Request<axum::body::Body>,
    next: Next,
) -> Result<Response, ApiError> {
    if req.uri().path() == "/health" {
        return Ok(next.run(req).await);
    }

    if let Some(expected_token) = state.auth_token {
        let provided = req
            .headers()
            .get(header::AUTHORIZATION)
            .and_then(|value| value.to_str().ok())
            .and_then(|value| value.strip_prefix("Bearer "));

        if provided != Some(expected_token.as_str()) {
            return Err(ApiError::unauthorized());
        }
    }

    Ok(next.run(req).await)
}

async fn health() -> Json<HealthResponse> {
    Json(HealthResponse {
        status: "ok",
        service: "data-cos-api",
    })
}

async fn data_fetch_extract(
    Json(payload): Json<DataFetchExtractRequest>,
) -> Result<Json<DataFetchResponse>, ApiError> {
    let result = tokio::task::spawn_blocking(move || extract_data(&payload))
        .await
        .map_err(|err| ApiError::internal(format!("task join error: {err}")))?
        .map_err(|err| {
            error!("data-fetch extract failed: {err:#}");
            ApiError::internal(format!("data-fetch extraction failed: {err}"))
        })?;

    if result.errors.is_empty() {
        info!("data-fetch completed with {} records", result.records.len());
    } else {
        warn!(
            "data-fetch completed with {} records and {} errors",
            result.records.len(),
            result.errors.len()
        );
        for message in &result.errors {
            warn!("data-fetch item failed: {message}");
        }
    }

    Ok(Json(DataFetchResponse {
        total: result.records.len(),
        records: result.records,
        errors: result.errors,
        infos: result.infos,
    }))
}

async fn data_fetch_diagnostics(
    Json(payload): Json<ExtractionDiagnosticsRequest>,
) -> Json<ExtractionDiagnosticsResponse> {
    let response = tokio::task::spawn_blocking(move || run_extraction_diagnostics(payload.roots))
        .await
        .unwrap_or_else(|err| ExtractionDiagnosticsResponse {
            process_id: std::process::id(),
            executable_path: current_executable_path(),
            cache_directory: extraction_cache_directory().display().to_string(),
            checks: vec![ExtractionDiagnosticCheck {
                code: "diagnostic_task",
                label: "诊断任务",
                status: "fail",
                path: None,
                detail: format!("诊断任务异常结束：{err}"),
                os_error_code: None,
            }],
            failed: 1,
            warnings: 0,
        });

    Json(response)
}

fn run_extraction_diagnostics(roots: Vec<String>) -> ExtractionDiagnosticsResponse {
    let executable_path = current_executable_path();
    let cache_directory = extraction_cache_directory();
    let mut checks = vec![ExtractionDiagnosticCheck {
        code: "backend_process",
        label: "后端进程",
        status: "pass",
        path: Some(executable_path.clone()),
        detail: format!(
            "后端进程正在运行（PID {}），界面与本机 API 通信正常。",
            std::process::id()
        ),
        os_error_code: None,
    }];

    let mut unique_roots = Vec::new();
    let mut seen = HashSet::new();
    for root in roots {
        let trimmed = root.trim();
        if !trimmed.is_empty() && seen.insert(trimmed.to_string()) {
            unique_roots.push(trimmed.to_string());
        }
        if unique_roots.len() >= 10 {
            break;
        }
    }

    if unique_roots.is_empty() {
        checks.push(ExtractionDiagnosticCheck {
            code: "data_root",
            label: "数据目录",
            status: "fail",
            path: None,
            detail: String::from("没有配置数据根目录。"),
            os_error_code: None,
        });
    } else {
        checks.extend(unique_roots.iter().map(|root| probe_data_root(root)));
    }

    checks.push(probe_cache_directory(&cache_directory));

    let failed = checks.iter().filter(|check| check.status == "fail").count();
    let warnings = checks
        .iter()
        .filter(|check| check.status == "warning")
        .count();

    info!(
        "extraction diagnostics completed: {} checks, {} failed, {} warnings",
        checks.len(),
        failed,
        warnings
    );
    for check in &checks {
        if check.status != "pass" {
            warn!(
                "diagnostic {} [{}]: {}",
                check.code, check.status, check.detail
            );
        }
    }

    ExtractionDiagnosticsResponse {
        process_id: std::process::id(),
        executable_path,
        cache_directory: cache_directory.display().to_string(),
        checks,
        failed,
        warnings,
    }
}

fn current_executable_path() -> String {
    env::current_exe()
        .map(|path| path.display().to_string())
        .unwrap_or_else(|err| format!("无法确定后端路径：{err}"))
}

fn extraction_cache_directory() -> std::path::PathBuf {
    env::temp_dir().join("data-cos-suite-cache")
}

fn probe_data_root(root: &str) -> ExtractionDiagnosticCheck {
    let path = Path::new(root);
    let metadata = match fs::metadata(path) {
        Ok(metadata) => metadata,
        Err(err) => {
            return ExtractionDiagnosticCheck {
                code: "data_root",
                label: "数据目录",
                status: "fail",
                path: Some(root.to_string()),
                detail: format!("无法访问数据目录：{err}"),
                os_error_code: err.raw_os_error(),
            };
        }
    };

    if !metadata.is_dir() {
        return ExtractionDiagnosticCheck {
            code: "data_root",
            label: "数据目录",
            status: "fail",
            path: Some(root.to_string()),
            detail: String::from("配置路径存在，但不是目录。"),
            os_error_code: None,
        };
    }

    let entries = match fs::read_dir(path) {
        Ok(entries) => entries,
        Err(err) => {
            return ExtractionDiagnosticCheck {
                code: "data_root",
                label: "数据目录",
                status: "fail",
                path: Some(root.to_string()),
                detail: format!("目录存在，但后端无法枚举内容：{err}"),
                os_error_code: err.raw_os_error(),
            };
        }
    };

    let mut sampled = 0usize;
    let mut unreadable = 0usize;
    for entry in entries.take(64) {
        sampled += 1;
        if entry.is_err() {
            unreadable += 1;
        }
    }

    let (status, detail) = if unreadable > 0 {
        (
            "warning",
            format!("目录可访问；抽样 {sampled} 个条目，其中 {unreadable} 个无法读取。"),
        )
    } else {
        (
            "pass",
            format!("目录可访问并可枚举；已抽样检查 {sampled} 个条目。"),
        )
    };

    ExtractionDiagnosticCheck {
        code: "data_root",
        label: "数据目录",
        status,
        path: Some(root.to_string()),
        detail,
        os_error_code: None,
    }
}

fn probe_cache_directory(cache_directory: &Path) -> ExtractionDiagnosticCheck {
    if let Err(err) = fs::create_dir_all(cache_directory) {
        return ExtractionDiagnosticCheck {
            code: "cache_write",
            label: "缓存目录",
            status: "fail",
            path: Some(cache_directory.display().to_string()),
            detail: format!("无法创建缓存目录：{err}"),
            os_error_code: err.raw_os_error(),
        };
    }

    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_nanos())
        .unwrap_or(0);
    let probe_path = cache_directory.join(format!(
        ".diagnostic-{}-{timestamp}.tmp",
        std::process::id()
    ));

    let write_result = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&probe_path)
        .and_then(|mut file| file.write_all(b"data-cos-suite diagnostic probe"));

    match write_result {
        Ok(()) => {
            let cleanup_result = fs::remove_file(&probe_path);
            let (status, detail) = if let Err(err) = cleanup_result {
                (
                    "warning",
                    format!("缓存可写，但诊断临时文件无法删除：{err}"),
                )
            } else {
                ("pass", String::from("缓存目录可创建、写入并清理临时文件。"))
            };
            ExtractionDiagnosticCheck {
                code: "cache_write",
                label: "缓存目录",
                status,
                path: Some(cache_directory.display().to_string()),
                detail,
                os_error_code: None,
            }
        }
        Err(err) => ExtractionDiagnosticCheck {
            code: "cache_write",
            label: "缓存目录",
            status: "fail",
            path: Some(cache_directory.display().to_string()),
            detail: format!("缓存目录不可写：{err}"),
            os_error_code: err.raw_os_error(),
        },
    }
}

async fn cos_list_files(
    Json(payload): Json<ListCosFilesRequest>,
) -> Result<Json<ListCosFilesResponse>, ApiError> {
    let default_directory = env::var("DATA_COS_BATCH_DIR").unwrap_or_else(|_| String::from("."));
    let directory = payload.directory.unwrap_or(default_directory);
    let files =
        tokio::task::spawn_blocking(move || list_cos_batch_files(std::path::Path::new(&directory)))
            .await
            .map_err(|err| ApiError::internal(format!("task join error: {err}")))?
            .map_err(|err| {
                error!("cos file list failed: {err:#}");
                ApiError::bad_request(format!("failed to list cos files: {err}"))
            })?;

    Ok(Json(ListCosFilesResponse {
        total: files.len(),
        files,
    }))
}

async fn cos_load(
    State(state): State<AppState>,
    Json(payload): Json<LoadCosRequest>,
) -> Result<Json<LoadCosResponse>, ApiError> {
    let file_path = payload.file_path.clone();
    let records = tokio::task::spawn_blocking(move || {
        let excel_path = std::path::Path::new(&file_path);
        let cache_path = excel_path.with_extension("cos_cache.bin");

        // 尝试从 bincode 缓存加载（缓存必须比 Excel 文件新）
        if cache_path.exists() {
            let excel_mtime = std::fs::metadata(excel_path)
                .and_then(|m| m.modified())
                .ok();
            let cache_mtime = std::fs::metadata(&cache_path)
                .and_then(|m| m.modified())
                .ok();

            let cache_valid = match (excel_mtime, cache_mtime) {
                (Some(e), Some(c)) => c >= e,
                _ => false,
            };

            if cache_valid {
                match std::fs::read(&cache_path)
                    .ok()
                    .and_then(|data| bincode::deserialize::<Vec<CosRecord>>(&data).ok())
                {
                    Some(cached) => {
                        info!("cos loaded {} records from bincode cache", cached.len());
                        return Ok(cached);
                    }
                    None => {
                        let _ = std::fs::remove_file(&cache_path);
                    }
                }
            }
        }
        load_and_cache_excel(excel_path, &excel_path.with_extension("cos_cache.bin"))
    })
    .await
    .map_err(|err| ApiError::internal(format!("task join error: {err}")))??;

    let total = records.len();
    info!("cos total {} records, stored in memory cache", total);
    *state.cos_cache.write().unwrap() = records;
    // Clear stale step1 cache when new data is loaded
    state.step1_cache.write().unwrap().clear();

    Ok(Json(LoadCosResponse { total }))
}

fn load_and_cache_excel(
    excel_path: &std::path::Path,
    cache_path: &std::path::Path,
) -> Result<Vec<CosRecord>, ApiError> {
    let records = load_cos_records_from_excel(excel_path).map_err(|err| {
        error!("cos load failed: {err:#}");
        ApiError::bad_request(format!("failed to load cos file: {err}"))
    })?;
    info!(
        "cos loaded {} records from Excel, writing bincode cache",
        records.len()
    );

    // 写入 bincode 缓存（失败不影响主流程）
    if let Ok(data) = bincode::serialize(&records) {
        let _ = std::fs::write(cache_path, data);
    }

    Ok(records)
}

async fn cos_step1(
    State(state): State<AppState>,
    Json(payload): Json<CosStep1Payload>,
) -> Result<Json<CosStep1Response>, ApiError> {
    let cached = state.cos_cache.read().unwrap();
    if cached.is_empty() {
        return Err(ApiError::bad_request(
            "no data loaded, call /cos-filter/load first",
        ));
    }
    let records = filter_cos_step1(&cached, &payload.params);
    drop(cached);
    info!("step1 filtered to {} records", records.len());

    // Cache step1 result so step2/grouping can reference it server-side
    *state.step1_cache.write().unwrap() = records.clone();

    Ok(Json(CosStep1Response {
        total: records.len(),
        records,
    }))
}

async fn cos_step2(
    State(state): State<AppState>,
    Json(payload): Json<CosStep2LightPayload>,
) -> Result<Json<CosStep2Response>, ApiError> {
    // Use client-provided records if present, otherwise fall back to cached step1
    let source = match payload.records {
        Some(ref r) if !r.is_empty() => r.clone(),
        _ => {
            let cached = state.step1_cache.read().unwrap();
            if cached.is_empty() {
                return Err(ApiError::bad_request(
                    "no step1 result cached; run step1 first or provide records",
                ));
            }
            cached.clone()
        }
    };
    let records = filter_cos_step2(&source, &payload.params);
    Ok(Json(CosStep2Response {
        total: records.len(),
        records,
    }))
}

/// Resolve records from payload or server-side step1 cache.
fn resolve_group_records(
    state: &AppState,
    payload_records: &Option<Vec<CosRecord>>,
) -> Result<Vec<CosRecord>, ApiError> {
    match payload_records {
        Some(r) if !r.is_empty() => Ok(r.clone()),
        _ => {
            let cached = state.step1_cache.read().unwrap();
            if cached.is_empty() {
                return Err(ApiError::bad_request(
                    "no step1 result cached; run step1 first or provide records",
                ));
            }
            Ok(cached.clone())
        }
    }
}

async fn cos_group_greedy(
    State(state): State<AppState>,
    Json(payload): Json<CosGroupLightPayload>,
) -> Result<Json<CosGroupResponse>, ApiError> {
    let records = resolve_group_records(&state, &payload.records)?;
    let grouped = group_wavelengths_greedy(&records, &payload.params)
        .map_err(|err| ApiError::bad_request(format!("grouping failed: {err}")))?;
    Ok(Json(build_group_response(grouped)))
}

async fn cos_group_optimal(
    State(state): State<AppState>,
    Json(payload): Json<CosGroupLightPayload>,
) -> Result<Json<CosGroupResponse>, ApiError> {
    let records = resolve_group_records(&state, &payload.records)?;
    let grouped = group_wavelengths_optimal(&records, &payload.params)
        .map_err(|err| ApiError::bad_request(format!("grouping failed: {err}")))?;
    Ok(Json(build_group_response(grouped)))
}

async fn cos_group_flat_top(
    State(state): State<AppState>,
    Json(payload): Json<CosGroupLightPayload>,
) -> Result<Json<CosGroupResponse>, ApiError> {
    let records = resolve_group_records(&state, &payload.records)?;
    let grouped = group_wavelengths_flat_top(&records, &payload.params)
        .map_err(|err| ApiError::bad_request(format!("grouping failed: {err}")))?;
    Ok(Json(build_group_response(grouped)))
}

async fn cos_group_huang_meng(
    State(state): State<AppState>,
    Json(payload): Json<CosGroupLightPayload>,
) -> Result<Json<CosGroupResponse>, ApiError> {
    let records = resolve_group_records(&state, &payload.records)?;
    let grouped = group_wavelengths_huang_meng(&records, &payload.params)
        .map_err(|err| ApiError::bad_request(format!("grouping failed: {err}")))?;
    Ok(Json(build_group_response(grouped)))
}

async fn cos_step4_extract(
    Json(payload): Json<CosStep4ExtractPayload>,
) -> Result<Json<DataFetchResponse>, ApiError> {
    let entries = collect_chip_entries_from_cos_records(&payload.records);
    if entries.is_empty() {
        return Err(ApiError::bad_request(
            "no valid chip entries found in records",
        ));
    }

    let request = DataFetchExtractRequest {
        mode: ExtractionMode::Chip,
        entries,
        test_categories: None,
        measurements: payload.measurements,
        current_points: payload.current_points,
        module_default_root: None,
        chip_default_root: payload.chip_default_root,
        chip_default_roots: payload.chip_default_roots,
    };

    let DataFetchExtractResponse {
        records,
        errors,
        infos,
    } = extract_data(&request).map_err(|err| {
        error!("cos step4 extract failed: {err:#}");
        ApiError::internal(format!("cos step4 extraction failed: {err}"))
    })?;

    Ok(Json(DataFetchResponse {
        total: records.len(),
        records,
        errors,
        infos,
    }))
}

fn collect_chip_entries_from_cos_records(records: &[CosRecord]) -> Vec<String> {
    let mut seen: HashSet<String> = HashSet::new();
    let mut entries = Vec::new();

    for record in records {
        let device_id = record.device_id.trim();
        if device_id.is_empty() {
            continue;
        }
        if seen.insert(device_id.to_string()) {
            entries.push(device_id.to_string());
        }
    }

    entries
}

fn build_group_response(grouped: data_cos_core::CosGroupingResult) -> CosGroupResponse {
    CosGroupResponse {
        group_count: grouped.groups.len(),
        remaining_count: grouped.remaining.len(),
        groups: grouped.groups,
        remaining: grouped.remaining,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn temporary_test_path(label: &str) -> std::path::PathBuf {
        let suffix = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("system clock must be after epoch")
            .as_nanos();
        env::temp_dir().join(format!("data-cos-api-{label}-{suffix}"))
    }

    fn record(device_id: &str) -> CosRecord {
        CosRecord {
            device_id: device_id.to_string(),
            warehouse: None,
            isolation: None,
            item_num: None,
            box_num: None,
            owner: None,
            cold_wavelength_nm: None,
            center_wavelength_nm: None,
            two_a_wavelength_nm: None,
            peak_wavelength_nm: None,
        }
    }

    #[test]
    fn collect_chip_entries_dedupes_and_trims_in_order() {
        let rows = vec![
            record(" CHIP-001 "),
            record("CHIP-002"),
            record("CHIP-001"),
            record("CHIP-003"),
            record(" CHIP-002 "),
        ];

        let entries = collect_chip_entries_from_cos_records(&rows);
        assert_eq!(entries, vec!["CHIP-001", "CHIP-002", "CHIP-003"]);
    }

    #[test]
    fn collect_chip_entries_ignores_blank_ids() {
        let rows = vec![record(""), record("   "), record("\t"), record("CHIP-100")];

        let entries = collect_chip_entries_from_cos_records(&rows);
        assert_eq!(entries, vec!["CHIP-100"]);
    }

    #[test]
    fn collect_chip_entries_returns_empty_for_all_blank() {
        let rows = vec![record(""), record(" "), record("\n")];
        let entries = collect_chip_entries_from_cos_records(&rows);
        assert!(entries.is_empty());
    }

    #[test]
    fn diagnostics_reports_readable_directory() {
        let root = temporary_test_path("readable-root");
        fs::create_dir_all(&root).expect("must create diagnostic root");
        fs::write(root.join("sample.txt"), b"sample").expect("must create sample file");

        let check = probe_data_root(root.to_str().expect("test path must be unicode"));

        assert_eq!(check.status, "pass");
        assert_eq!(check.code, "data_root");
        fs::remove_dir_all(root).expect("must clean diagnostic root");
    }

    #[test]
    fn diagnostics_reports_missing_directory() {
        let root = temporary_test_path("missing-root");
        let check = probe_data_root(root.to_str().expect("test path must be unicode"));

        assert_eq!(check.status, "fail");
        assert!(check.detail.contains("无法访问数据目录"));
    }

    #[test]
    fn diagnostics_probes_cache_write_and_cleanup() {
        let cache = temporary_test_path("cache");
        let check = probe_cache_directory(&cache);

        assert_eq!(check.status, "pass");
        assert_eq!(fs::read_dir(&cache).expect("cache must exist").count(), 0);
        fs::remove_dir_all(cache).expect("must clean cache directory");
    }
}
