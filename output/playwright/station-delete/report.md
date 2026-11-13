# 站别删除

本次方向沿用既有实验室数据工作台：紧凑、准确、安静；保留 MyGo 原生界面、uix 复选框、灰色表面与蓝色提取按钮。仅补齐站别删除与相关状态。

## 最终行为

- 常用站别与新增站别共用「删除站别」右键菜单。Tab 聚焦站别后，Shift+F10 打开同一菜单。
- 不常显删除按钮，也不在每个站别后放置删除按钮。名称继续使用完整行宽。
- 关闭菜单不会删除或切换勾选。提取期间禁用站别操作，删除处理函数也检查忙碌状态。
- 列表、已选数量和提取请求都从同一站别清单生成。删除后同步更新，后续全选不会恢复删除项。
- 新增站别删除时清理名称和未勾选记录；常用站别通过 `deletedTests` 保存移除状态，兼容旧配置。删除后的名称可以重新添加。
- 删除最后一个站别时显示添加提示，并禁用模块提取。芯片模式继续使用 postQCW 规则。
- 仅删除站别配置，测试文件和已有提取结果不变。删除后把键盘焦点移回编号输入，防止残留焦点继续操作移位后的控件。

## 检查与证据

- `npm run test:mygo`：通过，Go 竞争检测、原生界面测试、既有真实引擎检查、`go vet`。
- `TestStationDeletion`：浅色、深色；选中和未选中项；常用与新增站别；菜单取消；Shift+F10；重复删除；全选不恢复；忙碌时拒绝删除；删除最后一项；重新添加。
- `TestSettingsRoundTrip`：包含删除状态的保存与读取，恢复值一致。
- 原生界面截图：1000×720、1440×900，浅色与深色。长名称、40 个自定义站别滚动与提取按钮检查通过。
- `npm run test:mygo:e2e`：18 项通过，真实原生窗口与 Rust/Excel 测试数据；报告见 `native-e2e.json`。此套端到端检查覆盖提取流程，删除菜单另由原生 UI Tester 验证。
- `go tool mygo vet .`、`git diff --check`：通过。
- `npm run build:mygo`：通过，生成 macOS arm64 应用。签名校验通过。

未验证：Windows 原生运行。未关闭或重启用户窗口。

截图：`before-1000x720.png`、`before-1440x900.png` 为改前；`after-custom-stations-1000x720-dark-false.png`、`after-custom-stations-1000x720-dark-true.png`、对应 1440×900 文件及 `after-native-custom-stations.png` 为最终界面。

设计规则：20ui R01、R06、R08、R20；unslop by Damir Sokolovsky（@heydamir），CC BY 4.0。沿用现有视觉样式，不扩展整体改版。
