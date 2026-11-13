# MyGo 数据提取重构验证

日期：2026-11-13（Asia/Shanghai）

## 交付

- MyGo 原生 Go 界面，固定依赖版本 `v0.9.1-0.20261113060250-614ea7a6af6e`。
- Rust 提取 worker 直接复用原 `data-cos-data-adapter`，无本地 HTTP 服务。
- 默认 `npm start` 启动新工具；旧 Electron 入口保留为 `npm run start:electron`。
- macOS arm64 应用：`native/build/darwin-arm64/Data COS Suite.app`。
- 不包含 COS 筛选界面。未修改本次开始前的 Web 工作区改动。

## 按操作频率重排

用户指定编号／完整路径、电流点、站别、提取按钮、结果为主要区域。

- 左侧按输入顺序排列编号、电流点、站别和提取按钮。
- 编号输入框随窗口高度扩展，提取按钮固定在左侧底部，按钮高度 44 DIP。
- 右侧结果表占满可用高度。拖动中间分隔条可分配输入与结果的宽度。
- 数据根目录和测试文件类型移入右上角设置，不占用主工作区。
- 1440×900 与 1000×720 检查：重点控件均可见；编号输入框高度至少 140 DIP；结果区保留窗口高度减 120 DIP 以上。
- 窄窗口将搜索框放到结果工具栏的第二行，表格支持横向滚动。
- 以上为布局和交互检查，不代表用户实际长时间操作验收。

## 验证

- `cargo test --manifest-path rust/Cargo.toml -p data-cos-data-adapter -p data-cos-core`：19 项 Rust 测试通过。
- `npm run test:mygo`：Go 竞争检测及 `go vet` 通过。
- 真实测试 Excel：模块／芯片模式 × 最大电流／全部电流／指定电流，共 6 个场景通过；LVI 与 Rth 记录合并、功率、电压、波长、效率换算符合预期。
- `lambd` 通用波长文件在模块与芯片模式下各 1 个场景通过。
- 原生 UI 测试：中文文本、站别全选／清空、自定义站别、设置开关、Escape、模式切换、重点控件边界、取消后保留上次结果、请求快照和数值排序通过。
- Excel 导出使用数值单元格，不额外舍入；字符串不会转成公式；提取提示工作表和 CSV 精度检查通过。
- `npm run test:mygo:e2e`：9 项真实窗口检查通过，见 `native-e2e.json`。窗口类型为原生 content；模块 4 个编号提取 12 条合并记录，芯片提取 3 条记录。
- `npm run build:mygo`：macOS arm64 打包通过，约 18.6 MB。
- `codesign --verify --deep --strict --verbose=2 'native/build/darwin-arm64/Data COS Suite.app'`：通过。本地 ad hoc 签名，未公证或发布。
- 从应用包 Resources 中独立运行提取 worker：指定 2 A，返回 1 条合并记录，功率 5 W、效率 48%。
- `git diff --check`：通过。

## 截图

- `before.png`：旧界面，1440×900。
- `after-native-1440x900.png`：新布局，同尺寸，无数据状态。
- `after-native-1000x720.png`：新布局，窄窗口，无数据状态。
- `after-native-module.png`：真实原生窗口，测试数据提取 12 条记录。截图 2880×1684；内容视口为 1440×842，设备缩放为 2。
- `native-first.png`：本次早期上／下布局，用户反馈后取消该布局。仅用于记录此次调整。

## 未验证

- 用户真实网络盘、生产目录和真实批次数据。
- Windows 与 Linux 的实际运行及安装包。
- 系统输入法的候选窗和组合输入。自动化只验证中文文本输入。
- 系统文件选择、文件保存对话框的人工操作。导出文件内容单独通过测试。
- 长时间使用、大规模批次、屏幕阅读器与系统缩放的全面验收。
