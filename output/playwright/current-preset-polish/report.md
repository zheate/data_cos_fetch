# 电流快捷值与新增站别细节优化

范围：保留 MyGo 原生界面、uix 组件、现有颜色和工作区布局，修复电流快捷值样式及站别显示与交互的一致性。

产品：批量提取模块、芯片测量数据。
性格：紧凑、准确、安静。
场景：实验室数据工作台，保留中性灰与既有蓝色操作重点。
重点：编号输入、提取动作和结果表；局部控件不争夺视觉注意力。
标记：站别名称直接可见，已选数量与实际提取范围一致。

## 问题与修复

1. 「最大值／全部」的手工选中背景没有圆角。改为 uix ToggleGroup，统一轨道、选中项与焦点样式。自定义电流范围时，两项均不选中。未使用会强制选择第一项的 Segmented。
2. 自定义站别原来只显示数量，入口只显示设置图标。改为直接显示名称，并显示「新增站别」按钮。列表高度最多 152 DIP，长列表独立滚动。
3. 首次显示名称时使用了勾号行，与常用站别的复选框不一致。最终统一调用 `stationCheckbox`，共用 uix Checkbox 的尺寸、圆角、背景、焦点和禁用状态。空列使用相同内边距，单个新增站别与其他项等宽。
4. 新增站别现在可以勾选与取消。全选、清空和已选数量覆盖两类站别。请求只包含勾选的站别。新增配置字段 `uncheckedCustomTests` 保存未勾选的自定义名称；旧配置缺少此字段时，原有自定义站别继续默认勾选。编辑清单保留现有名称的选择，新增名称默认勾选，删除名称同时清理其选择记录。

规则：20ui R01（统一控件）、R09（分组间距）、R10（窗口适配）、R15（明确选择反馈）。设计方法沿用 unslop by Damir Sokolovsky（@heydamir），CC BY 4.0。

## 验证

- `npm run test:mygo`：通过。包含 Go 竞争检测、现有提取检查、原生界面测试、`go vet`。
- 新增原生界面检查：浅色和深色；1000×720 与 1440×900；保存、去重、取消、Escape、点击外部、删除；勾选、空格键、全选、清空、忙碌禁用；40 个自定义站别滚动至最后一项，编号输入与提取按钮保持可见。
- `npm run test:mygo:e2e` 的构建与截图检查通过；最终自动化脚本直接复用该测试构建运行，18 项通过。见 `native-e2e.json`。真实 Rust 引擎与临时 Excel 文件：三站别得到 36 条记录，取消一个自定义站别后得到 24 条记录。
- `go tool mygo vet .`：通过。
- `npm run build:mygo`：通过，生成 macOS arm64 原生应用。
- `codesign --verify --deep --strict`：通过。
- `git diff --check`：通过。

测试窗口使用临时设置和临时数据，结束后关闭本任务的窗口与进程。未关闭或重启用户当前窗口。

未验证：Windows 原生运行。Rust 的文件查找、合并、单位换算与导出规则未改动。

## 截图

- `before-1000x720.png`、`before-1440x900.png`：电流快捷值修改前。
- `presets-maximum-dark-false.png`、`presets-all-dark-true.png`：统一后的电流快捷值。
- `before-custom-stations-1000x720.png`、`before-custom-stations-1440x900.png`：只有自定义站别数量的原界面。
- `after-custom-stations-1000x720-dark-false.png`、`after-custom-stations-1000x720-dark-true.png`：统一复选框后的窄窗口浅色与深色。
- `after-native-custom-stations.png`：真实原生窗口，新增站别与常用站别共用复选框。

应用：`native/build/darwin-arm64/Data COS Suite.app`。重新打开此构建后生效。
