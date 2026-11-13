# data-cos-suite

默认桌面入口现在是 **MyGo 原生数据提取工具**，只提供模块与芯片数据提取。界面与运行方式见 [native/README.md](native/README.md)。

```sh
npm run bootstrap
npm start
npm run build:mygo
npm run test:mygo
```

旧版 Electron 使用 `npm run start:electron`，旧版依赖安装使用 `npm run bootstrap:electron`。以下保留原迁移结构和旧版说明。

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

## Legacy Electron Desktop

```bash
# First run (or after dependency changes): install deps + warm the cargo cache
npm run bootstrap:electron

# Every launch after that: Electron + Rust API + Vite
npm run start:electron

# Build desktop installers (electron-builder)
./scripts/package_desktop.sh
```

`npm run start:electron` assumes dependencies are already installed (use `npm run bootstrap:electron`
once). In development the Electron shell reuses a prebuilt backend binary from
`rust/target/release` or `rust/target/debug` when it is newer than the rust
sources, and only falls back to `cargo run` when that binary is missing or stale
— so routing around a slow first compile or a cold crate cache. Run
`npm --prefix desktop run build:api` for a release binary, or point at a custom
one with `DATA_COS_API_BIN=/path/to/data-cos-api npm run start:electron`.
