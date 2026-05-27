use std::collections::HashMap;
use std::fs;
use std::io::{Read, Seek};
use std::path::{Path, PathBuf};

use anyhow::{Context, Result};
use calamine::{Cell, Data, Range, Reader, open_workbook_auto, open_workbook_auto_from_rs};
use chrono::NaiveDateTime;
use data_cos_core::{CosRecord, DataFetchRecord, merge_data_fetch_records};
use serde::{Deserialize, Serialize};

const CURRENT_TOLERANCE: f64 = 1e-6;
const LVI_SKIP_ROWS: usize = 18;
const RTH_SKIP_ROWS: usize = 8;
const DEFAULT_TEST_CATEGORIES: [&str; 6] = [
    "耦合测试",
    "Pre测试",
    "低温储存后测试",
    "Post测试",
    "封盖测试",
    "高温测试",
];

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum ExtractionMode {
    Module,
    Chip,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataFetchExtractRequest {
    pub mode: ExtractionMode,
    pub entries: Vec<String>,
    pub test_categories: Option<Vec<String>>,
    pub measurements: Option<Vec<String>>,
    pub current_points: Option<Vec<f64>>,
    pub module_default_root: Option<String>,
    pub chip_default_root: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataFetchExtractResponse {
    pub records: Vec<DataFetchRecord>,
    pub errors: Vec<String>,
    pub infos: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CosBatchFileInfo {
    pub file_path: String,
    pub file_name: String,
    pub modified_epoch_s: i64,
    pub size_bytes: u64,
}

#[derive(Debug, Clone)]
struct FileCandidate {
    path: PathBuf,
    timestamp: Option<i64>,
    mtime: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct LviPoint {
    current_a: f64,
    power_w: Option<f64>,
    voltage_v: Option<f64>,
    efficiency_pct: Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct RthPoint {
    current_a: f64,
    lambda_nm: f64,
}

use rayon::prelude::*;
use std::hash::{Hash, Hasher};
use std::collections::hash_map::DefaultHasher;

fn get_local_cache_path(excel_path: &Path, mtime: i64, prefix: &str) -> PathBuf {
    let mut hasher = DefaultHasher::new();
    excel_path.to_string_lossy().hash(&mut hasher);
    mtime.hash(&mut hasher);
    let hash = hasher.finish();

    let temp_dir = std::env::temp_dir().join("data-cos-suite-cache");
    let _ = fs::create_dir_all(&temp_dir);
    temp_dir.join(format!("{}_{:016x}.bin", prefix, hash))
}

pub fn extract_data(request: &DataFetchExtractRequest) -> Result<DataFetchExtractResponse> {
    let measurements = normalize_measurements(request.measurements.as_ref());

    // Use Rayon to process entries in parallel across all CPU cores
    let results: Vec<_> = request
        .entries
        .par_iter()
        .map(|entry| {
            let mut local_records = Vec::new();
            let mut local_errors = Vec::new();
            let mut local_infos = Vec::new();

            match request.mode {
                ExtractionMode::Module => {
                    if let Err(err) = extract_module_entry(
                        entry,
                        request,
                        &measurements,
                        &mut local_records,
                        &mut local_errors,
                        &mut local_infos,
                    ) {
                        local_errors.push(format!("{entry}: {err}"));
                    }
                }
                ExtractionMode::Chip => {
                    if let Err(err) = extract_chip_entry(
                        entry,
                        request,
                        &measurements,
                        &mut local_records,
                        &mut local_errors,
                        &mut local_infos,
                    ) {
                        local_errors.push(format!("{entry}: {err}"));
                    }
                }
            }
            (local_records, local_errors, local_infos)
        })
        .collect();

    let mut records = Vec::new();
    let mut errors = Vec::new();
    let mut infos = Vec::new();

    for (r, e, i) in results {
        records.extend(r);
        errors.extend(e);
        infos.extend(i);
    }

    let mut merged = merge_data_fetch_records(&records);
    for row in &mut merged {
        row.current_a = round_to_three(row.current_a);
        row.power_w = round_to_three(row.power_w);
        row.voltage_v = round_to_three(row.voltage_v);
        row.efficiency_pct = round_to_three(row.efficiency_pct.map(|value| value * 100.0));
        row.lambda_nm = round_to_three(row.lambda_nm);
        row.shift_nm = round_to_three(row.shift_nm);
    }
    merged.sort_by(|left, right| {
        left.entry_id
            .cmp(&right.entry_id)
            .then_with(|| left.test_category.cmp(&right.test_category))
            .then_with(|| {
                left.current_a
                    .unwrap_or(f64::NEG_INFINITY)
                    .total_cmp(&right.current_a.unwrap_or(f64::NEG_INFINITY))
            })
    });

    Ok(DataFetchExtractResponse {
        records: merged,
        errors,
        infos,
    })
}

fn round_to_three(value: Option<f64>) -> Option<f64> {
    value.map(|raw| (raw * 1000.0).round() / 1000.0)
}

pub fn load_cos_records_from_excel(path: &Path) -> Result<Vec<CosRecord>> {
    let range = read_first_sheet(path)?;
    let (header_idx, header_map) = locate_cos_header(&range)
        .with_context(|| format!("failed to detect COS header in {}", path.display()))?;

    let mut rows = Vec::new();
    for row in range.rows().skip(header_idx + 1) {
        let device_id = header_map
            .get("device_id")
            .and_then(|idx| row.get(*idx))
            .and_then(cell_to_trimmed_string)
            .unwrap_or_default();
        if device_id.is_empty() {
            continue;
        }

        let record = CosRecord {
            device_id,
            warehouse: get_cell_string(row, header_map.get("warehouse")),
            isolation: get_cell_string(row, header_map.get("isolation")),
            item_num: get_cell_string(row, header_map.get("item_num")),
            box_num: get_cell_string(row, header_map.get("box_num")),
            owner: get_cell_string(row, header_map.get("owner")),
            cold_wavelength_nm: get_cell_f64(row, header_map.get("cold_wavelength")),
            center_wavelength_nm: get_cell_f64(row, header_map.get("center_wavelength")),
            two_a_wavelength_nm: get_cell_f64(row, header_map.get("two_a_wavelength")),
            peak_wavelength_nm: get_cell_f64(row, header_map.get("peak_wavelength")),
        };
        rows.push(record);
    }

    Ok(rows)
}

pub fn list_cos_batch_files(directory: &Path) -> Result<Vec<CosBatchFileInfo>> {
    if !directory.exists() || !directory.is_dir() {
        return Ok(Vec::new());
    }

    let mut files = Vec::new();
    let entries = fs::read_dir(directory)
        .with_context(|| format!("failed to read directory {}", directory.display()))?;

    for entry in entries {
        let entry = entry?;
        let path = entry.path();
        if !path.is_file() {
            continue;
        }

        let extension = path
            .extension()
            .and_then(|ext| ext.to_str())
            .map(|ext| ext.to_ascii_lowercase())
            .unwrap_or_default();
        if extension != "xls" && extension != "xlsx" {
            continue;
        }

        let file_name = match path.file_name().and_then(|name| name.to_str()) {
            Some(name) => name.to_string(),
            None => continue,
        };
        if !file_name.contains("批次实例") {
            continue;
        }

        let metadata = fs::metadata(&path)?;
        let modified_epoch_s = metadata
            .modified()
            .ok()
            .and_then(|time| time.duration_since(std::time::UNIX_EPOCH).ok())
            .map(|duration| duration.as_secs() as i64)
            .unwrap_or(0);

        files.push(CosBatchFileInfo {
            file_path: path.to_string_lossy().to_string(),
            file_name,
            modified_epoch_s,
            size_bytes: metadata.len(),
        });
    }

    files.sort_by(|left, right| {
        right
            .modified_epoch_s
            .cmp(&left.modified_epoch_s)
            .then_with(|| left.file_name.cmp(&right.file_name))
    });
    Ok(files)
}

fn extract_module_entry(
    entry: &str,
    request: &DataFetchExtractRequest,
    measurements: &[String],
    out: &mut Vec<DataFetchRecord>,
    errors: &mut Vec<String>,
    infos: &mut Vec<String>,
) -> Result<()> {
    let default_root = request
        .module_default_root
        .as_deref()
        .unwrap_or("Z:/Ldtd/fcp/");
    let base_path = interpret_module_entry(entry, default_root)?;

    let categories: Vec<String> = request.test_categories.clone().unwrap_or_else(|| {
        DEFAULT_TEST_CATEGORIES
            .iter()
            .map(|item| item.to_string())
            .collect()
    });

    for category in categories {
        let test_folder = match resolve_test_folder(&base_path, &category) {
            Ok(path) => path,
            Err(err) => {
                errors.push(format!(
                    "{entry} [{category}] directory access failed: {err}"
                ));
                continue;
            }
        };
        let index = match build_measurement_index(&test_folder, false) {
            Ok(index) => index,
            Err(err) => {
                errors.push(format!(
                    "{entry} [{category}] directory access failed: {err}"
                ));
                continue;
            }
        };

        for measurement in measurements {
            let token = measurement_token(measurement);
            let (file, has_multiple) = match latest_measurement_file(&index, token) {
                Some(file) => file,
                None => {
                    infos.push(format!("{entry}/{category}/{measurement}: file not found"));
                    continue;
                }
            };
            if has_multiple {
                let file_name = file
                    .path
                    .file_name()
                    .and_then(|name| name.to_str())
                    .unwrap_or("unknown");
                infos.push(format!(
                    "{entry}/{category}/{measurement}: found multiple files, using latest -> {file_name}"
                ));
            }

            let mut rows = match measurement.as_str() {
                "LVI" => extract_lvi_records(
                    entry,
                    &category,
                    &file.path,
                    request.current_points.as_ref(),
                    file.mtime,
                ),
                "Rth" => extract_rth_records(
                    entry,
                    &category,
                    &file.path,
                    request.current_points.as_ref(),
                    file.mtime,
                ),
                "lambd" => extract_generic_records(
                    entry,
                    &category,
                    &file.path,
                    request.current_points.as_ref(),
                    file.mtime,
                ),
                other => {
                    infos.push(format!(
                        "{entry}/{category}/{other}: unsupported measurement"
                    ));
                    Ok(Vec::new())
                }
            };
            match rows.as_mut() {
                Ok(rows) => out.append(rows),
                Err(err) => errors.push(format!("{entry}/{category}/{measurement}: {err}")),
            }
        }
    }

    Ok(())
}

fn extract_chip_entry(
    entry: &str,
    request: &DataFetchExtractRequest,
    measurements: &[String],
    out: &mut Vec<DataFetchRecord>,
    errors: &mut Vec<String>,
    infos: &mut Vec<String>,
) -> Result<()> {
    let default_root = request.chip_default_root.as_deref().unwrap_or("Z:/Ldtd/");
    let chip_root = interpret_chip_entry(entry, default_root)?;
    let index = match build_measurement_index(&chip_root, true) {
        Ok(index) => index,
        Err(err) => {
            errors.push(format!("{entry} path resolution failed: {err}"));
            return Ok(());
        }
    };

    for measurement in measurements {
        let token = measurement_token(measurement);
        let (file, has_multiple) = match latest_measurement_file(&index, token) {
            Some(file) => file,
            None => {
                infos.push(format!("{entry}/{measurement}: file not found"));
                continue;
            }
        };
        if has_multiple {
            let file_name = file
                .path
                .file_name()
                .and_then(|name| name.to_str())
                .unwrap_or("unknown");
            infos.push(format!(
                "{entry}/{measurement}: found multiple files, using latest -> {file_name}"
            ));
        }

        let mut rows = match measurement.as_str() {
            "LVI" => extract_lvi_records(
                entry,
                "芯片测试",
                &file.path,
                request.current_points.as_ref(),
                file.mtime,
            ),
            "Rth" => extract_rth_records(
                entry,
                "芯片测试",
                &file.path,
                request.current_points.as_ref(),
                file.mtime,
            ),
            "lambd" => extract_generic_records(
                entry,
                "芯片测试",
                &file.path,
                request.current_points.as_ref(),
                file.mtime,
            ),
            other => {
                infos.push(format!("{entry}/{other}: unsupported measurement"));
                Ok(Vec::new())
            }
        };
        match rows.as_mut() {
            Ok(rows) => out.append(rows),
            Err(err) => errors.push(format!("{entry}/{measurement}: {err}")),
        }
    }

    Ok(())
}

fn normalize_measurements(measurements: Option<&Vec<String>>) -> Vec<String> {
    let default = vec![String::from("LVI"), String::from("Rth")];
    let picked = measurements.cloned().unwrap_or(default);
    picked
        .into_iter()
        .map(|item| {
            if item.eq_ignore_ascii_case("rth") {
                String::from("Rth")
            } else if item.eq_ignore_ascii_case("lvi") {
                String::from("LVI")
            } else {
                item
            }
        })
        .collect()
}

fn measurement_token(measurement: &str) -> &str {
    match measurement {
        "LVI" => "LVI",
        "Rth" => "Rth",
        other => other,
    }
}

fn interpret_module_entry(entry: &str, default_root: &str) -> Result<PathBuf> {
    let trimmed = entry.trim();
    anyhow::ensure!(!trimmed.is_empty(), "module entry is empty");
    if has_path_separator(trimmed) {
        return Ok(PathBuf::from(trimmed));
    }

    let mut path = PathBuf::from(default_root);
    for ch in trimmed.chars() {
        path.push(ch.to_string());
    }
    Ok(path)
}

fn interpret_chip_entry(entry: &str, default_root: &str) -> Result<PathBuf> {
    let trimmed = entry.trim();
    anyhow::ensure!(!trimmed.is_empty(), "chip entry is empty");
    if has_path_separator(trimmed) {
        let path = PathBuf::from(trimmed);
        anyhow::ensure!(
            path.exists() && path.is_dir(),
            "chip folder not found: {}",
            trimmed
        );
        return Ok(path);
    }

    let mut direct = PathBuf::from(default_root);
    direct.push(trimmed);
    if direct.exists() {
        return Ok(direct);
    }

    let mut by_chars = PathBuf::from(default_root);
    for ch in trimmed.chars() {
        by_chars.push(ch.to_string());
    }
    if by_chars.exists() {
        return Ok(by_chars);
    }

    anyhow::bail!("chip folder not found: {trimmed}")
}

fn has_path_separator(value: &str) -> bool {
    value.contains('\\') || value.contains('/') || value.contains(':')
}

fn resolve_test_folder(base_path: &Path, category: &str) -> Result<PathBuf> {
    let candidate = base_path.join(category);
    anyhow::ensure!(
        candidate.exists(),
        "test category folder not found: {}",
        candidate.display()
    );

    let nested = candidate.join("测试");
    if nested.exists() {
        Ok(nested)
    } else {
        Ok(candidate)
    }
}

fn build_measurement_index(
    root: &Path,
    recursive: bool,
) -> Result<HashMap<String, Vec<FileCandidate>>> {
    let mut index: HashMap<String, Vec<FileCandidate>> = HashMap::new();

    if recursive {
        collect_measurement_files_recursive(root, &mut index)?;
    } else {
        collect_measurement_files_shallow(root, &mut index)?;
    }

    Ok(index)
}

fn collect_measurement_files_shallow(
    root: &Path,
    index: &mut HashMap<String, Vec<FileCandidate>>,
) -> Result<()> {
    let entries = fs::read_dir(root)
        .with_context(|| format!("failed to read directory {}", root.display()))?;
    for entry in entries {
        let entry = entry?;
        let path = entry.path();
        if path.is_file() {
            maybe_push_measurement_file(path, index)?;
        }
    }
    Ok(())
}

fn collect_measurement_files_recursive(
    root: &Path,
    index: &mut HashMap<String, Vec<FileCandidate>>,
) -> Result<()> {
    if !root.exists() {
        return Ok(());
    }
    let entries = fs::read_dir(root)
        .with_context(|| format!("failed to read directory {}", root.display()))?;
    for entry in entries {
        let entry = entry?;
        let path = entry.path();
        if path.is_dir() {
            collect_measurement_files_recursive(&path, index)?;
        } else {
            maybe_push_measurement_file(path, index)?;
        }
    }
    Ok(())
}

fn maybe_push_measurement_file(
    path: PathBuf,
    index: &mut HashMap<String, Vec<FileCandidate>>,
) -> Result<()> {
    let extension = path
        .extension()
        .and_then(|ext| ext.to_str())
        .map(|ext| ext.to_ascii_lowercase())
        .unwrap_or_default();

    if extension != "xls" && extension != "xlsx" {
        return Ok(());
    }

    let stem = match path.file_stem().and_then(|stem| stem.to_str()) {
        Some(stem) => stem,
        None => return Ok(()),
    };

    let token = match stem.rsplit_once('=') {
        Some((_, token)) => token.trim(),
        None => return Ok(()),
    };

    let metadata = fs::metadata(&path)?;
    let mtime = metadata
        .modified()
        .ok()
        .and_then(|time| time.duration_since(std::time::UNIX_EPOCH).ok())
        .map(|duration| duration.as_secs() as i64)
        .unwrap_or(0);

    let timestamp = extract_timestamp_from_name(stem);

    index
        .entry(token.to_string())
        .or_default()
        .push(FileCandidate {
            path,
            timestamp,
            mtime,
        });

    Ok(())
}

fn latest_measurement_file<'a>(
    index: &'a HashMap<String, Vec<FileCandidate>>,
    token: &str,
) -> Option<(&'a FileCandidate, bool)> {
    index.get(token).and_then(|files| {
        let selected = files.iter().max_by(|left, right| {
            let left_has_timestamp = if left.timestamp.is_some() { 1i8 } else { 0i8 };
            let right_has_timestamp = if right.timestamp.is_some() { 1i8 } else { 0i8 };
            let left_primary = left.timestamp.unwrap_or(left.mtime);
            let right_primary = right.timestamp.unwrap_or(right.mtime);
            left_has_timestamp
                .cmp(&right_has_timestamp)
                .then_with(|| left_primary.cmp(&right_primary))
                .then_with(|| left.mtime.cmp(&right.mtime))
                .then_with(|| left.path.file_name().cmp(&right.path.file_name()))
        })?;
        Some((selected, files.len() > 1))
    })
}

fn extract_timestamp_from_name(stem: &str) -> Option<i64> {
    let prefix = stem.split('=').next().unwrap_or_default();
    let digits: String = prefix.chars().filter(|ch| ch.is_ascii_digit()).collect();

    for (format, length) in [
        ("%Y%m%d%H%M%S", 14usize),
        ("%Y%m%d%H%M", 12usize),
        ("%Y%m%d", 8usize),
    ] {
        if digits.len() >= length {
            let snippet = &digits[..length];
            let parsed = NaiveDateTime::parse_from_str(snippet, format)
                .ok()
                .map(|value| value.and_utc().timestamp());
            if parsed.is_some() {
                return parsed;
            }

            if format == "%Y%m%d" {
                let with_midnight = format!("{snippet}000000");
                if let Ok(value) = NaiveDateTime::parse_from_str(&with_midnight, "%Y%m%d%H%M%S") {
                    return Some(value.and_utc().timestamp());
                }
            }
        }
    }

    None
}

fn extract_lvi_records(
    entry_id: &str,
    test_category: &str,
    path: &Path,
    current_points: Option<&Vec<f64>>,
    mtime: i64,
) -> Result<Vec<DataFetchRecord>> {
    let cache_path = get_local_cache_path(path, mtime, "LVI");
    let mut points: Vec<LviPoint> = Vec::new();

    if let Ok(data) = fs::read(&cache_path) {
        if let Ok(cached) = bincode::deserialize(&data) {
            points = cached;
        }
    }

    if points.is_empty() {
        let range = read_first_sheet(path)?;

        // 动态定位起始行：寻找包含“电流”或类似关键词的第一行
        let skip_rows = range
            .rows()
            .enumerate()
            .find(|(_, row)| find_column(row, &["电流(A)", "电流", "current"]).is_some())
            .map(|(idx, _)| idx + 1)
            .unwrap_or(LVI_SKIP_ROWS);

        for row in range.rows().skip(skip_rows) {
            let current = row.first().and_then(cell_to_f64);
            let Some(current_a) = current else {
                continue;
            };
            if current_a.abs() <= CURRENT_TOLERANCE {
                continue;
            }

            points.push(LviPoint {
                current_a,
                power_w: row.get(1).and_then(cell_to_f64),
                voltage_v: row.get(2).and_then(cell_to_f64),
                efficiency_pct: row.get(3).and_then(cell_to_f64),
            });
        }

        anyhow::ensure!(
            !points.is_empty(),
            "LVI has no usable rows in {}",
            path.display()
        );

        if let Ok(data) = bincode::serialize(&points) {
            let _ = fs::write(&cache_path, data);
        }
    }

    let selected = pick_current_points(
        points.iter().map(|point| point.current_a).collect(),
        current_points,
    );

    let mut result = Vec::new();
    for point in points {
        if selected.contains(&normalize_current(point.current_a)) {
            result.push(DataFetchRecord {
                entry_id: entry_id.to_string(),
                test_category: test_category.to_string(),
                current_a: Some(point.current_a),
                power_w: point.power_w,
                voltage_v: point.voltage_v,
                efficiency_pct: point.efficiency_pct,
                lambda_nm: None,
                shift_nm: None,
                wavelength_2a_nm: None,
                wavelength_cold_nm: None,
            });
        }
    }

    Ok(result)
}

fn extract_rth_records(
    entry_id: &str,
    test_category: &str,
    path: &Path,
    current_points: Option<&Vec<f64>>,
    mtime: i64,
) -> Result<Vec<DataFetchRecord>> {
    let cache_path = get_local_cache_path(path, mtime, "Rth");
    let mut points: Vec<RthPoint> = Vec::new();

    if let Ok(data) = fs::read(&cache_path) {
        if let Ok(cached) = bincode::deserialize(&data) {
            points = cached;
        }
    }

    if points.is_empty() {
        let range = read_first_sheet(path)?;

        // 动态定位起始行
        let skip_rows = range
            .rows()
            .enumerate()
            .find(|(_, row)| find_column(row, &["Wavelength", "波长lambda"]).is_some())
            .map(|(idx, _)| idx + 1)
            .unwrap_or(RTH_SKIP_ROWS);

        for row in range.rows().skip(skip_rows) {
            let lambda = row.first().and_then(cell_to_f64);
            let current = row.get(2).and_then(cell_to_f64);
            let (Some(lambda_nm), Some(current_a)) = (lambda, current) else {
                continue;
            };
            if current_a.abs() <= CURRENT_TOLERANCE {
                continue;
            }

            points.push(RthPoint {
                current_a,
                lambda_nm,
            });
        }

        anyhow::ensure!(
            !points.is_empty(),
            "Rth has no usable rows in {}",
            path.display()
        );

        if let Ok(data) = bincode::serialize(&points) {
            let _ = fs::write(&cache_path, data);
        }
    }

    let baseline = points
        .iter()
        .find(|point| (point.current_a - 2.0).abs() <= CURRENT_TOLERANCE)
        .map(|point| point.lambda_nm)
        .unwrap_or_else(|| {
            points
                .iter()
                .min_by(|left, right| left.current_a.total_cmp(&right.current_a))
                .map(|point| point.lambda_nm)
                .unwrap_or(0.0)
        });

    let wavelength_2a = points
        .iter()
        .find(|point| (point.current_a - 2.0).abs() <= CURRENT_TOLERANCE)
        .map(|point| point.lambda_nm);

    let wavelength_cold = estimate_cold_wavelength(&points);

    let selected = pick_current_points(
        points.iter().map(|point| point.current_a).collect(),
        current_points,
    );

    let mut result = Vec::new();
    for point in points {
        if selected.contains(&normalize_current(point.current_a)) {
            result.push(DataFetchRecord {
                entry_id: entry_id.to_string(),
                test_category: test_category.to_string(),
                current_a: Some(point.current_a),
                power_w: None,
                voltage_v: None,
                efficiency_pct: None,
                lambda_nm: Some(point.lambda_nm),
                shift_nm: Some(point.lambda_nm - baseline),
                wavelength_2a_nm: wavelength_2a,
                wavelength_cold_nm: wavelength_cold,
            });
        }
    }

    Ok(result)
}

fn extract_generic_records(
    entry_id: &str,
    test_category: &str,
    path: &Path,
    current_points: Option<&Vec<f64>>,
    mtime: i64,
) -> Result<Vec<DataFetchRecord>> {
    let cache_path = get_local_cache_path(path, mtime, "Generic");
    let mut rows: Vec<DataFetchRecord> = Vec::new();

    if let Ok(data) = fs::read(&cache_path) {
        if let Ok(cached) = bincode::deserialize(&data) {
            rows = cached;
        }
    }

    if rows.is_empty() {
        let range = read_first_sheet(path)?;
        let Some((header_idx, header_map)) = locate_data_fetch_header(&range) else {
            anyhow::bail!(
                "Generic Excel file {} does not contain recognized header (expecting 'current' etc.)",
                path.display()
            );
        };
        let Some(current_idx) = header_map.get("current").copied() else {
            anyhow::bail!(
                "Generic Excel file {} missing 'current' column",
                path.display()
            );
        };

        for row in range.rows().skip(header_idx + 1) {
            let current = row.get(current_idx).and_then(cell_to_f64);
            let Some(current_a) = current else {
                continue;
            };
            if current_a.abs() <= CURRENT_TOLERANCE {
                continue;
            }

            rows.push(DataFetchRecord {
                entry_id: entry_id.to_string(),
                test_category: test_category.to_string(),
                current_a: Some(current_a),
                power_w: header_map
                    .get("power")
                    .and_then(|idx| row.get(*idx))
                    .and_then(cell_to_f64),
                voltage_v: header_map
                    .get("voltage")
                    .and_then(|idx| row.get(*idx))
                    .and_then(cell_to_f64),
                efficiency_pct: header_map
                    .get("efficiency")
                    .and_then(|idx| row.get(*idx))
                    .and_then(cell_to_f64),
                lambda_nm: header_map
                    .get("lambda")
                    .and_then(|idx| row.get(*idx))
                    .and_then(cell_to_f64),
                shift_nm: header_map
                    .get("shift")
                    .and_then(|idx| row.get(*idx))
                    .and_then(cell_to_f64),
                wavelength_2a_nm: header_map
                    .get("two_a")
                    .and_then(|idx| row.get(*idx))
                    .and_then(cell_to_f64),
                wavelength_cold_nm: header_map
                    .get("cold")
                    .and_then(|idx| row.get(*idx))
                    .and_then(cell_to_f64),
            });
        }

        if !rows.is_empty() {
            if let Ok(data) = bincode::serialize(&rows) {
                let _ = fs::write(&cache_path, data);
            }
        }
    }

    if rows.is_empty() {
        return Ok(rows);
    }

    let selected = pick_current_points(
        rows.iter().filter_map(|row| row.current_a).collect(),
        current_points,
    );
    let filtered = rows
        .into_iter()
        .filter(|row| {
            row.current_a
                .map(|current| selected.contains(&normalize_current(current)))
                .unwrap_or(false)
        })
        .collect();
    Ok(filtered)
}

fn estimate_cold_wavelength(points: &[RthPoint]) -> Option<f64> {
    if points.len() < 2 {
        return None;
    }

    let n = points.len() as f64;
    let sum_x: f64 = points.iter().map(|point| point.current_a).sum();
    let sum_y: f64 = points.iter().map(|point| point.lambda_nm).sum();
    let sum_xy: f64 = points
        .iter()
        .map(|point| point.current_a * point.lambda_nm)
        .sum();
    let sum_x2: f64 = points
        .iter()
        .map(|point| point.current_a * point.current_a)
        .sum();

    let denominator = n * sum_x2 - sum_x * sum_x;
    if denominator.abs() <= f64::EPSILON {
        return None;
    }

    let intercept = (sum_y * sum_x2 - sum_x * sum_xy) / denominator;
    Some(intercept)
}

fn pick_current_points(all_currents: Vec<f64>, requested: Option<&Vec<f64>>) -> Vec<String> {
    match requested {
        None => all_currents
            .into_iter()
            .map(normalize_current)
            .collect::<Vec<_>>(),
        Some(points) if points.is_empty() => {
            let max_current = all_currents
                .into_iter()
                .max_by(|left, right| left.total_cmp(right))
                .unwrap_or(0.0);
            vec![normalize_current(max_current)]
        }
        Some(points) => {
            let mut picked = Vec::new();
            for current in all_currents {
                if points
                    .iter()
                    .any(|expected| (current - expected).abs() <= CURRENT_TOLERANCE)
                {
                    picked.push(normalize_current(current));
                }
            }
            picked
        }
    }
}

fn normalize_current(current: f64) -> String {
    format!("{current:.6}")
}

fn read_first_sheet(path: &Path) -> Result<Range<Data>> {
    let primary_err = match read_first_sheet_from_extension(path) {
        Ok(range) => return Ok(range),
        Err(err) => err,
    };

    let bytes = fs::read(path)
        .with_context(|| format!("failed to read workbook bytes {}", path.display()))?;

    let probe_err = match read_first_sheet_from_bytes(path, &bytes) {
        Ok(range) => return Ok(range),
        Err(err) => err,
    };

    match read_first_sheet_from_delimited_bytes(&bytes) {
        Ok(range) => Ok(range),
        Err(delimited_err) => anyhow::bail!(
            "failed to open workbook {} (excel parser: {}; content probe: {}; delimited fallback: {})",
            path.display(),
            primary_err,
            probe_err,
            delimited_err
        ),
    }
}

fn read_first_sheet_from_extension(path: &Path) -> Result<Range<Data>> {
    let mut workbook = open_workbook_auto(path)?;
    read_first_sheet_from_workbook(path, &mut workbook)
}

fn read_first_sheet_from_bytes(path: &Path, bytes: &[u8]) -> Result<Range<Data>> {
    let cursor = std::io::Cursor::new(bytes);
    let mut workbook = open_workbook_auto_from_rs(cursor)?;
    read_first_sheet_from_workbook(path, &mut workbook)
}

fn read_first_sheet_from_workbook<RS>(
    path: &Path,
    workbook: &mut calamine::Sheets<RS>,
) -> Result<Range<Data>>
where
    RS: Read + Seek,
{
    let sheet_name = workbook
        .sheet_names()
        .first()
        .cloned()
        .context("workbook has no sheets")?;

    let range = workbook
        .worksheet_range(&sheet_name)
        .with_context(|| format!("failed to read sheet {sheet_name} in {}", path.display()))?;

    Ok(range)
}

fn read_first_sheet_from_delimited_bytes(bytes: &[u8]) -> Result<Range<Data>> {
    let delimiter = detect_delimiter(bytes);
    let mut reader = csv::ReaderBuilder::new()
        .has_headers(false)
        .delimiter(delimiter)
        .flexible(true)
        .from_reader(bytes);

    let mut cells: Vec<Cell<Data>> = Vec::new();
    for (row_idx, row) in reader.byte_records().enumerate() {
        let row = row.with_context(|| {
            format!(
                "failed to parse delimited text with delimiter byte {}",
                delimiter
            )
        })?;

        for (col_idx, raw) in row.iter().enumerate() {
            let text = String::from_utf8_lossy(raw);
            let trimmed = text.trim().trim_start_matches('\u{feff}');
            if trimmed.is_empty() {
                continue;
            }

            cells.push(Cell::new(
                (row_idx as u32, col_idx as u32),
                parse_delimited_cell(trimmed),
            ));
        }
    }

    anyhow::ensure!(!cells.is_empty(), "delimited parser found no usable cells");
    Ok(Range::from_sparse(cells))
}

fn detect_delimiter(bytes: &[u8]) -> u8 {
    const CANDIDATES: [u8; 4] = [b'\t', b',', b';', b'|'];
    let mut scores = [0usize; CANDIDATES.len()];

    for line in bytes.split(|byte| *byte == b'\n').take(200) {
        let trimmed = line
            .iter()
            .copied()
            .filter(|byte| *byte != b'\r' && *byte != b' ')
            .collect::<Vec<_>>();
        if trimmed.is_empty() {
            continue;
        }

        for (idx, delimiter) in CANDIDATES.iter().enumerate() {
            scores[idx] += trimmed.iter().filter(|byte| **byte == *delimiter).count();
        }
    }

    let mut best_idx = 0usize;
    let mut best_score = 0usize;
    for (idx, score) in scores.iter().copied().enumerate() {
        if score > best_score {
            best_score = score;
            best_idx = idx;
        }
    }

    if best_score == 0 {
        b','
    } else {
        CANDIDATES[best_idx]
    }
}

fn parse_delimited_cell(text: &str) -> Data {
    if let Ok(value) = text.parse::<f64>() {
        Data::Float(value)
    } else {
        Data::String(text.to_string())
    }
}

fn locate_cos_header(
    range: &calamine::Range<Data>,
) -> Option<(usize, HashMap<&'static str, usize>)> {
    for (row_idx, row) in range.rows().enumerate().take(50) {
        let mut map = HashMap::new();
        if let Some(idx) = find_column(row, &["LOT | SN", "LOT|SN"]) {
            map.insert("device_id", idx);
        }
        if let Some(idx) = find_column(row, &["仓库"]) {
            map.insert("warehouse", idx);
        }
        if let Some(idx) = find_column(row, &["是否隔离"]) {
            map.insert("isolation", idx);
        }
        if let Some(idx) = find_column(row, &["ItemNum", "ITEMNUM"]) {
            map.insert("item_num", idx);
        }
        if let Some(idx) = find_column(row, &["盒号"]) {
            map.insert("box_num", idx);
        }
        if let Some(idx) = find_column(row, &["货主"]) {
            map.insert("owner", idx);
        }
        if let Some(idx) = find_column(row, &["冷波长"]) {
            map.insert("cold_wavelength", idx);
        }
        if let Some(idx) = find_column(row, &["中心波长"]) {
            map.insert("center_wavelength", idx);
        }
        if let Some(idx) = find_column(row, &["2A波长"]) {
            map.insert("two_a_wavelength", idx);
        }
        if let Some(idx) = find_column(row, &["峰值波长"]) {
            map.insert("peak_wavelength", idx);
        }

        if map.contains_key("device_id")
            && map.contains_key("warehouse")
            && map.contains_key("isolation")
            && map.contains_key("item_num")
        {
            return Some((row_idx, map));
        }
    }

    None
}

fn locate_data_fetch_header(
    range: &calamine::Range<Data>,
) -> Option<(usize, HashMap<&'static str, usize>)> {
    for (row_idx, row) in range.rows().enumerate().take(50) {
        let mut map = HashMap::new();
        if let Some(idx) = find_column(row, &["电流(A)", "电流", "current_a", "current"]) {
            map.insert("current", idx);
        }
        if let Some(idx) = find_column(row, &["功率(W)", "功率", "power_w", "power"]) {
            map.insert("power", idx);
        }
        if let Some(idx) = find_column(row, &["电压(V)", "电压", "voltage_v", "voltage"]) {
            map.insert("voltage", idx);
        }
        if let Some(idx) = find_column(
            row,
            &["电光效率(%)", "电光效率", "efficiency_pct", "efficiency"],
        ) {
            map.insert("efficiency", idx);
        }
        if let Some(idx) = find_column(row, &["波长lambda", "lambda_nm", "lambda", "波长"]) {
            map.insert("lambda", idx);
        }
        if let Some(idx) = find_column(row, &["波长shift", "shift_nm", "wavelength_shift"]) {
            map.insert("shift", idx);
        }
        if let Some(idx) = find_column(row, &["2A波长", "wavelength_2a_nm"]) {
            map.insert("two_a", idx);
        }
        if let Some(idx) = find_column(row, &["冷波长", "wavelength_cold_nm"]) {
            map.insert("cold", idx);
        }

        if map.contains_key("current") {
            return Some((row_idx, map));
        }
    }
    None
}

fn find_column(row: &[Data], aliases: &[&str]) -> Option<usize> {
    row.iter().enumerate().find_map(|(idx, cell)| {
        let value = cell_to_trimmed_string(cell)?;
        if aliases
            .iter()
            .any(|alias| value.eq_ignore_ascii_case(alias) || value == *alias)
        {
            Some(idx)
        } else {
            None
        }
    })
}

fn get_cell_string(row: &[Data], idx: Option<&usize>) -> Option<String> {
    idx.and_then(|value| row.get(*value))
        .and_then(cell_to_trimmed_string)
}

fn get_cell_f64(row: &[Data], idx: Option<&usize>) -> Option<f64> {
    idx.and_then(|value| row.get(*value)).and_then(cell_to_f64)
}

fn cell_to_trimmed_string(cell: &Data) -> Option<String> {
    match cell {
        Data::String(value) => {
            let trimmed = value.trim();
            if trimmed.is_empty() {
                None
            } else {
                Some(trimmed.to_string())
            }
        }
        Data::Int(value) => Some(value.to_string()),
        Data::Float(value) => Some(value.to_string()),
        Data::Bool(value) => Some(value.to_string()),
        _ => None,
    }
}

fn cell_to_f64(cell: &Data) -> Option<f64> {
    match cell {
        Data::Float(value) => Some(*value),
        Data::Int(value) => Some(*value as f64),
        Data::String(value) => value.trim().parse::<f64>().ok(),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn detect_delimiter_prefers_tab() {
        let data = b"A\tB\tC\n1\t2\t3\n";
        assert_eq!(detect_delimiter(data), b'\t');
    }

    #[test]
    fn read_delimited_sheet_parses_numeric_cells() {
        let data = b"current,power,voltage\n2,10.5,5.2\n";
        let range = read_first_sheet_from_delimited_bytes(data).expect("must parse csv fallback");

        let current = range
            .get((1, 0))
            .and_then(cell_to_f64)
            .expect("current must be parsed");
        let power = range
            .get((1, 1))
            .and_then(cell_to_f64)
            .expect("power must be parsed");
        let voltage = range
            .get((1, 2))
            .and_then(cell_to_f64)
            .expect("voltage must be parsed");

        assert!((current - 2.0).abs() <= f64::EPSILON);
        assert!((power - 10.5).abs() <= f64::EPSILON);
        assert!((voltage - 5.2).abs() <= f64::EPSILON);
    }
}
