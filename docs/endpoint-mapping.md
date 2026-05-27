# Endpoint Mapping (Python Logic -> Rust API)

| Legacy Python Capability | Rust Endpoint | Request Shape | Response Shape | Status | Notes |
| --- | --- | --- | --- | --- | --- |
| Health Check | `GET /health` | none | `{ status, service }` | done | No auth required |
| Data Fetch (module/chip extraction) | `POST /api/v1/data-fetch/extract` | `mode, entries, test_categories?, measurements?, current_points?` | `records, errors, infos, total` | done | Supports LVI/Rth |
| COS Excel Load | `POST /api/v1/cos-filter/load` | `file_path` | `total` | done | Records stored in process memory cache; header auto-detection |
| COS Step1 (base filter) | `POST /api/v1/cos-filter/step1` | `params` | `records, total` | done | Reads from process cache (call `/load` first); isolation/warehouse/wavelength gates |
| COS Step2 (priority select) | `POST /api/v1/cos-filter/step2` | `records, params` | `records, total` | done | Supports optional item/box filter |
| COS Grouping (greedy) | `POST /api/v1/cos-filter/group/greedy` | `records, params` | `groups, remaining, group_count, remaining_count` | done | Deterministic greedy grouping |
| COS Grouping (optimal) | `POST /api/v1/cos-filter/group/optimal` | `records, params` | `groups, remaining, group_count, remaining_count` | done | Weighted interval scheduling |
| COS Grouping (flat-top) | `POST /api/v1/cos-filter/group/flat-top` | `records, params` | `groups, remaining, group_count, remaining_count` | done | Binned greedy + neighbor borrowing |
| COS Grouping (huang-meng) | `POST /api/v1/cos-filter/group/huang-meng` | `records, params` | `groups, remaining, group_count, remaining_count` | done | Dual-band pairing strategy |
| COS Step4 (extract grouped chips) | `POST /api/v1/cos-filter/step4/extract` | `records, measurements?, current_points?, chip_default_root?` | `records, errors, infos, total` | done | Reuses chip-mode extraction with COS-selected chips |

## Field Compatibility Notes

1. COS payload uses normalized English keys while preserving legacy semantic values.
2. Data-fetch response preserves legacy measurement semantics and merges LVI/Rth rows by `(entry_id, test_category, current_a)`.
3. Empty `item_num_filter` and `box_num_filter` are treated as "no filter".
4. `/cos-filter/load` stores records in `AppState::cos_cache` (process memory). Subsequent `/cos-filter/step1` reads from this cache instead of accepting records in the request body. This avoids transmitting large Excel datasets over HTTP repeatedly.

