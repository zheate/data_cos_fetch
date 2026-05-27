# Data Fetch + COS Filter Migration Plan

## 1. Scope and Architecture

This migration extracts two Streamlit-bound capabilities into a standalone service/UI stack:

1. `data_fetch` extraction pipeline (path resolve, measurement file lookup, LVI/Rth parsing, row merge).
2. `cos_filter` pipeline (batch load, step1 filter, step2 prioritization, multi-strategy grouping, step4 grouped-chip data extraction).

Target architecture is fixed to:

1. React UI: input validation, workflow orchestration, result visualization.
2. Rust API (Axum): route boundary, auth check, payload mapping, error mapping.
3. Rust Core: deterministic filtering/grouping and record merge logic.
4. Rust Data Adapter: filesystem + Excel read + baseline-oriented extraction.

## 2. Phase Breakdown

### Stage A - Contract Freeze

- Frozen endpoints in `/apps/data-cos-suite/docs/endpoint-mapping.md`.
- Request/response schemas defined by Rust handler payloads in:
  - `rust/crates/data-cos-api/src/main.rs`
  - `rust/crates/data-cos-data-adapter/src/lib.rs`

### Stage B - Rust Core

Implemented in `rust/crates/data-cos-core/src/lib.rs`:

1. `filter_cos_step1`
2. `filter_cos_step2`
3. `group_wavelengths_greedy`
4. `group_wavelengths_optimal`
5. `group_wavelengths_flat_top`
6. `group_wavelengths_huang_meng`
7. `merge_data_fetch_records`

Coverage status:

- Unit tests added for step1/step2/four grouping strategies/merge path.

### Stage C - Rust API + Data Adapter

Implemented in:

1. `rust/crates/data-cos-data-adapter/src/lib.rs`
2. `rust/crates/data-cos-api/src/main.rs`

Adapter covers:

- Module/chip entry parsing.
- Test folder resolution with `测试` subdir preference.
- Measurement file indexing and latest-file selection.
- LVI/Rth extraction and field mapping.
- COS Excel header detection and row conversion.

API covers:

- Auth gate (`DATA_COS_API_TOKEN` + `Authorization: Bearer ...`).
- Health endpoint.
- Data-fetch and cos-filter route set.
- COS step4 bridge endpoint to extract measurement data from grouped COS chip selections.

### Stage D - Parity Regression

Automated parity script:

- `apps/data-cos-suite/scripts/parity_harness.py`

Report:

- `apps/data-cos-suite/docs/parity-report.md`

Current result:

- PASS on synthetic fixture for data-fetch and cos-filter step chain.

### Stage E - UI and Cutover Prep

React workbench implemented in:

- `apps/data-cos-suite/web/src/App.tsx`
- `apps/data-cos-suite/web/src/App.css`

The UI is now API-driven and does not import Python implementation details.

Desktop shell implemented in:

- `apps/data-cos-suite/desktop/main.cjs`
- `apps/data-cos-suite/desktop/preload.cjs`
- `apps/data-cos-suite/desktop/package.json`

Desktop flow:

1. Electron main spawns Rust API child process.
2. Main allocates dynamic port and injects runtime bearer token.
3. React renderer reads runtime config via preload bridge.

## 3. High-Risk Items and Controls

1. API field semantics drift (unit, defaults)
   - Controlled by explicit schema + parity compare.
2. Excel parser behavior drift on malformed files
   - Adapter validates worksheet/header presence and returns structured error.
3. Numeric stability for shift/cold wavelength
   - Deterministic baseline/linear regression path in adapter; tolerances enforced in parity.
4. Warehouse/owner priority logic regressions
   - Step2 logic fixed to treat empty filters as "no filter" and covered by parity.

## 4. Validation Commands

```bash
# Rust compile check
cargo check --manifest-path apps/data-cos-suite/rust/Cargo.toml

# Rust core unit tests
cargo test --manifest-path apps/data-cos-suite/rust/Cargo.toml -p data-cos-core

# Frontend production build
npm --prefix apps/data-cos-suite/web run build

# End-to-end parity (spawns Rust API, generates fixture, compares baseline vs Rust)
python3 apps/data-cos-suite/scripts/parity_harness.py
```

## 5. Cutover and Rollback

Detailed operations are in `/apps/data-cos-suite/docs/cutover.md`.

Summary:

1. Start Rust API and React UI in parallel with existing Streamlit.
2. Route pilot users to new app for `data_fetch` + `cos_filter` workloads.
3. Keep Python pages as baseline fallback until parity and user acceptance stabilize.
