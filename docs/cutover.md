# Cutover and Rollback Plan

## Runtime Controls

Environment variables:

1. `DATA_COS_API_PORT` (default `9002`)
2. `DATA_COS_API_TOKEN` (optional bearer token)

## Startup

```bash
# Terminal 1: Rust API
cargo run --manifest-path apps/data-cos-suite/rust/Cargo.toml -p data-cos-api

# Terminal 2: React UI
npm --prefix apps/data-cos-suite/web run dev
```

## Desktop Startup

```bash
# One command local desktop run (Electron)
./apps/data-cos-suite/scripts/dev_desktop.sh
```

Desktop runtime behavior:

1. Electron main process allocates a dynamic localhost port.
2. Electron generates runtime bearer token.
3. Rust API child process is started with `DATA_COS_API_PORT` and `DATA_COS_API_TOKEN`.
4. Preload bridge injects runtime config to React (`window.desktopRuntime.getConfig()`).

## Cutover Sequence

1. Keep existing Streamlit app running as baseline.
2. Run parity script after each high-risk backend change.
3. Route selected users to React UI for `data_fetch` and `cos_filter` tasks.
4. Monitor extraction errors and response latency.
5. Promote to default path only after parity stays green and pilot feedback is clean.

## Rollback

If a production-blocking mismatch appears:

1. Stop routing traffic to new React + Rust app.
2. Return users to existing Streamlit pages.
3. Preserve failing payload and parity diff for root-cause analysis.
4. Patch Rust logic and rerun parity before re-enable.

## Packaging

```bash
# Build desktop package
./apps/data-cos-suite/scripts/package_desktop.sh
```

Packaging config:

- Electron builder config is in `/apps/data-cos-suite/desktop/package.json`.
- `extraResources` bundles:
  1. Rust API binary (`rust/target/release/data-cos-api*`)
  2. Built React assets (`web/dist`)
