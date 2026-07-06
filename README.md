# data-cos-suite

Standalone migration target for `data_fetch` and `cos_filter`.

## Layout

- `rust/crates/data-cos-core`: deterministic business logic
- `rust/crates/data-cos-data-adapter`: file + excel adapters
- `rust/crates/data-cos-api`: Axum routes and auth boundary
- `web`: React workbench
- `desktop`: Electron shell (spawns Rust API with dynamic port + token)
- `scripts/parity_harness.py`: parity regression harness
- `docs/*`: migration artifacts

## Quick Start

```bash
cargo check --manifest-path rust/Cargo.toml
cargo test --manifest-path rust/Cargo.toml -p data-cos-core
npm --prefix web install --include=dev
npm --prefix web run build
python3 scripts/parity_harness.py
```

## One-Click Desktop

```bash
# Start Electron + Rust API + Vite (development)
npm start

# Build desktop installers (electron-builder)
./scripts/package_desktop.sh
```
