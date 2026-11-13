package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/smallnest/uix/components/button"
	"github.com/smallnest/uix/components/checkbox"
	"github.com/smallnest/uix/components/empty"
	"github.com/smallnest/uix/components/field"
	"github.com/smallnest/uix/components/input"
	"github.com/smallnest/uix/components/menu"
	"github.com/smallnest/uix/components/searchfield"
	"github.com/smallnest/uix/components/segmented"
	"github.com/smallnest/uix/components/split"
	"github.com/smallnest/uix/components/table"
	"github.com/smallnest/uix/components/textarea"
)

func (a *application) view(c *ui.Context) {
	a.finishTasks()
	if a.busy {
		c.After(80 * time.Millisecond)
	}
	t := workbenchTheme(c)
	c.SetTheme(t)
	if c.Shortcut(ui.Cmd, ui.KeyEnter) {
		a.run()
	}
	if c.Shortcut(ui.Cmd, ui.KeyE) {
		a.export("xlsx")
	}
	if c.Shortcut(ui.Cmd, ui.KeyL) {
		a.entriesField.Focus()
	}
	if c.Shortcut(ui.Cmd, ui.KeyF) && a.result != nil {
		a.resultTab = 0
		a.searchField.Focus()
	}
	width, _ := c.Size()
	maxSidebar := min(float32(520), width-560)
	a.sidebarWidth = max(320, min(a.sidebarWidth, maxSidebar))
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		a.header(c)
		if split.Split(c.Key("workbench-split"), split.Props{Size: &a.sidebarWidth, First: func(c *ui.Context) {
			a.configuration(c)
		}, Second: func(c *ui.Context) {
			ui.Column(c).Fill().Padding(0, 16, 0, 12).Children(func() { a.results(c, width-a.sidebarWidth-32) })
		}}).SplitReset(360).Grow(1).MinHeight(0).Changed() {
			a.config.SidebarWidth = a.sidebarWidth
		}
		a.footer(c)
	})
	a.settingsDialog(c)
	a.customTestsDialog(c)
	ui.Modal(c.Key("diagnostics-dialog"), &a.dialogOpen, func() {
		ui.Text(c, a.dialogTitle).FontSize(18).FontWeight(500)
		ui.Scroll(c).Width(600).Height(320).Children(func() { ui.Text(c, a.dialogText).Selectable().FontSize(13) })
		workbenchButton(c, button.Props{Label: "关闭", Variant: button.Secondary, Size: button.Md, OnClick: func() { a.dialogOpen = false }})
	})
}

func (a *application) header(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Height(56).Padding(0, 20).Gap(12).Children(func() {
		ui.Icon(c, tableIcon).FontSize(20).TextColor(t.Accent)
		ui.Text(c, "数据提取").FontSize(18).FontWeight(500)
		ui.Text(c, "Data COS Suite").FontSize(12).TextColor(t.TextMuted).Grow(1)
		iconButton(c, "diagnostics", "读取环境检测", shieldIcon, a.busy).OnClick(a.diagnostics)
		iconButton(c, "settings", "数据目录与测试文件设置", slidersIcon, a.busy).OnClick(func() { a.settingsOpen = true })
		menu.Menu(c.Key("export"), menu.Props{Label: "导出结果", Disabled: a.busy || a.result == nil || len(a.result.Records) == 0, Items: []menu.Item{
			{Label: "Excel 工作簿（全部记录）", Action: func() { a.export("xlsx") }},
			{Label: "CSV 文件（全部记录）", Action: func() { a.export("csv") }},
		}}).Height(32)
	})
}

func (a *application) configuration(c *ui.Context) {
	ui.Column(c).Fill().Padding(0, 16, 8, 20).Gap(12).Children(func() {
		ui.Row(c).Height(32).Gap(12).Children(func() {
			fieldLabel(c, "提取条件")
			ui.Spacer(c)
			segmented.Segmented(c.Key("mode"), segmented.Props{Selected: &a.config.Mode, Labels: []string{"模块", "芯片"}, Label: "数据模式"}).FillEqually().Width(140).Disabled(a.busy)
		})
		a.entriesInput(c)
		a.currentInput(c)
		a.stationInput(c)
		a.runAction(c)
	})
}

func (a *application) entriesInput(c *ui.Context) {
	t := c.Theme()
	entries := lines(a.config.Entries)
	ui.Column(c).Height(168).Shrink(0).Gap(8).Children(func() {
		ui.Row(c).Height(32).Gap(4).Children(func() {
			fieldLabel(c, "编号 / 完整路径")
			ui.Textf(c, "%d 个", len(entries)).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted).Grow(1)
			iconButton(c, "import-entries", "导入文本清单", importIcon, a.busy).OnClick(a.importEntries)
			iconButton(c, "tidy-entries", "整理编号清单", tidyIcon, a.busy || len(entries) == 0).OnClick(a.tidyEntries)
			iconButton(c, "clear-entries", "清空编号", trashIcon, a.busy || a.config.Entries == "").OnClick(func() { a.config.Entries = ""; a.entriesField.Focus() })
		})
		textarea.Textarea(c.Key("entries"), textarea.Props{Value: &a.config.Entries, Placeholder: "每行一个编号或完整路径", Rows: 5, Disabled: a.busy}).Label("编号或完整路径清单").Bind(&a.entriesField).Grow(1).MinHeight(120).FontSize(14)
	})
}

func (a *application) currentInput(c *ui.Context) {
	t := c.Theme()
	_, err := parseCurrents(a.config.Currents)
	field.Field(c, field.Props{}, func() {
		ui.Row(c).Height(28).Gap(4).Children(func() {
			fieldLabel(c, "电流点")
			ui.Text(c, "A").FontSize(12).TextColor(t.TextMuted).Grow(1)
			a.currentPresets(c)
		})
		e := input.Input(c.Key("currents"), input.Props{Value: &a.config.Currents, Placeholder: "例如 2, 4, 8~12", Disabled: a.busy}).Label("电流点 / A").Height(36).Invalid(err != nil)
		if err != nil {
			errorText := err.Error()
			ui.Row(c).MinHeight(20).Children(func() {
				errorMessage := ui.Text(c.Key("current-error"), errorText).FontSize(12).TextColor(t.Danger).SingleLine().Grow(1).MinWidth(0).Tooltip(errorText)
				e.ErrorMessage(errorMessage)
			})
		}
	}).Gap(6)
}

func (a *application) stationInput(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).MinHeight(0).Gap(8).Disabled(a.busy).Children(func() {
		if a.config.Mode == 1 {
			fieldLabel(c, "芯片测试")
			ui.Text(c, "自动查找 postQCW 目录").FontSize(13).TextColor(t.TextMuted)
			return
		}
		stations := a.config.stations()
		selected := len(a.config.selectedTests())
		allSelected := len(stations) > 0 && selected == len(stations)
		ui.Row(c).Height(28).Gap(4).Children(func() {
			fieldLabel(c, "测试站别")
			ui.Textf(c, "%d 个已选", selected).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted).Grow(1)
			text := "全选"
			if allSelected {
				text = "清空"
			}
			quietButton(c, "all-tests", text, len(stations) == 0, func() {
				a.config.selectAllTests(!allSelected)
			}).Label(text + "测试站别")
			quietButton(c, "custom-button", "新增站别", false, a.openCustomTests).Label("自定义测试站别")
			quietButton(c, "delete-tests", "删除", a.busy || selected == 0, a.deleteSelectedStations).Label("删除选中站别")
		})
		count := len(stations)
		if count == 0 {
			ui.Text(c, "暂无站别，点击「新增站别」添加").FontSize(12).TextColor(t.TextMuted).MinHeight(32)
			return
		}
		rows := (count + 1) / 2
		// Give stations the remaining height while keeping entry and run controls visible.
		ui.Scroll(c.Key("station-list")).Label("测试站别列表").Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Gap(8).Children(func() {
				for row := 0; row < rows; row++ {
					ui.Row(c).Gap(8).Children(func() {
						for col := 0; col < 2; col++ {
							i := row*2 + col
							if i < count {
								a.stationChoice(c, stations[i])
							} else {
								ui.Spacer(c).Grow(1).Basis(0).Padding(6)
							}
						}
					})
				}
			})
		})
	})
}

func (a *application) stationChoice(c *ui.Context, name string) {
	checked := a.config.stationChecked(name)
	e := stationCheckbox(c, "test-"+name, name, &checked, a.busy)
	if e.Changed() {
		a.config.setStationChecked(name, checked)
	}
	if checked {
		e.Background(c.Theme().SurfaceHover)
	}
}

func (a *application) runAction(c *ui.Context) {
	_, err := a.config.request()
	hint := "Ctrl / ⌘ + Enter 提取"
	if err != nil {
		hint = err.Error()
	}
	if a.busy {
		hint = "正在提取；取消会保留上次结果"
	}
	ui.Row(c).Gap(8).Children(func() {
		if a.busy {
			workbenchButton(c.Key("cancel"), button.Props{Label: "取消", Variant: button.Secondary, Size: button.Lg, OnClick: func() {
				if a.cancel != nil {
					a.cancel()
				}
				a.status = "正在取消"
			}}).Height(44)
		}
		workbenchButton(c.Key("run"), button.Props{Label: "开始提取", Variant: button.Primary, Size: button.Lg, Disabled: a.busy || err != nil, OnClick: a.run}).Height(44).Grow(1).Justify(ui.Center).FontSize(14).Tooltip(hint)
	})
}

func (a *application) settingsDialog(c *ui.Context) {
	t := c.Theme()
	ui.Modal(c.Key("settings-dialog"), &a.settingsOpen, func() {
		ui.Column(c).Width(560).Gap(16).Children(func() {
			ui.Text(c, "数据目录与测试文件").FontSize(18).FontWeight(500)
			fieldLabel(c, "数据根目录")
			ui.Row(c).Gap(8).Children(func() {
				if a.config.Mode == 0 {
					input.Input(c.Key("module-root"), input.Props{Value: &a.config.ModuleRoot}).Label("模块根目录").Grow(1)
				} else {
					textarea.Textarea(c.Key("chip-roots"), textarea.Props{Value: &a.config.ChipRoots}).Label("芯片根目录，每行一个").Grow(1).Height(112)
				}
				iconButton(c, "choose-root", "选择数据根目录", folderIcon, a.busy).OnClick(a.chooseRoot)
			})
			ui.Text(c, "完整路径可以直接提取；编号使用此根目录查找").FontSize(12).TextColor(t.TextMuted)
			fieldLabel(c, "测试文件")
			ui.Row(c).Gap(24).Children(func() {
				for i, name := range measurements {
					checkbox.Checkbox(c.Key("measurement-"+name), checkbox.Props{Checked: &a.config.Measurements[i], Label: name})
				}
			})
			ui.Row(c).Justify(ui.End).Children(func() {
				workbenchButton(c, button.Props{Label: "完成", Variant: button.Secondary, Size: button.Md, OnClick: func() { a.settingsOpen = false }})
			})
		})
	})
}

func (a *application) customTestsDialog(c *ui.Context) {
	ui.Modal(c.Key("custom-tests-dialog"), &a.customOpen, func() {
		ui.Text(c, "自定义测试站别").FontSize(18).FontWeight(500)
		ui.Text(c, "每行一个站别，目录名与测试文件夹一致").TextColor(c.Theme().TextMuted).MaxWidth(460)
		textarea.Textarea(c.Key("custom-tests"), textarea.Props{Value: &a.customDraft, Rows: 6}).Label("自定义测试站别清单").Width(460).Height(160)
		ui.Row(c).Justify(ui.End).Gap(8).Children(func() {
			workbenchButton(c, button.Props{Label: "取消", Variant: button.Secondary, Size: button.Md, OnClick: func() { a.customOpen = false }})
			workbenchButton(c, button.Props{Label: "保存站别", Variant: button.Primary, Size: button.Md, OnClick: a.saveCustomTests})
		})
	})
}

func (a *application) resultSearch(c *ui.Context, width float32) {
	if searchfield.SearchField(c.Key("search"), searchfield.Props{Value: &a.query, Disabled: a.result == nil || a.resultTab == 1}).Label("搜索编号或站别").Placeholder("搜索编号或站别").Bind(&a.searchField).Width(width).Changed() {
		a.filter()
	}
}

func (a *application) results(c *ui.Context, width float32) {
	t := c.Theme()
	n, issues := 0, 0
	if a.result != nil {
		n = len(a.result.Records)
		issues = len(a.result.Errors) + len(a.result.Infos)
	}
	ui.Column(c).Grow(1).MinHeight(0).Border(1, t.Border).Radius(10).Clip().Role(ui.RoleRegion).Label("提取结果工作区").Children(func() {
		ui.Column(c).Padding(12).Gap(8).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				segmented.Segmented(c.Key("result-tabs"), segmented.Props{Selected: &a.resultTab, Labels: []string{fmt.Sprintf("提取结果  %d", n), fmt.Sprintf("查找提示  %d", issues)}, Label: "结果视图"})
				ui.Spacer(c)
				if width >= 760 {
					a.resultSearch(c, 224)
				}
				label := "复制筛选后的记录"
				if a.selection.Len() > 0 {
					label = "复制选中记录"
				}
				iconButton(c, "copy-results", label, copyIcon, len(a.rows) == 0 || a.resultTab == 1).OnClick(func() {
					rows := a.selectedRows()
					if len(rows) == 0 {
						rows = a.rows
					}
					a.copyRows(c, rows)
				})
				iconButton(c, "clear-results", "清空提取结果", trashIcon, a.busy || a.result == nil).OnClick(a.clearResult)
			})
			if width < 760 {
				a.resultSearch(c, max(160, width-26))
			}
		})
		ui.Divider(c)
		if a.resultTab == 1 {
			a.issues(c)
			return
		}
		ui.Box(c).Grow(1).MinHeight(0).Children(func() {
			a.resultTable(c)
			if len(a.rows) == 0 {
				title, detail := "从一组编号开始", "填写编号与提取条件后，结果会显示在这里"
				action := "填写编号"
				onAction := func() { a.entriesField.Focus() }
				if a.result != nil {
					title = "没有匹配的记录"
					detail = "检查查找提示或调整搜索条件"
					action = ""
					onAction = nil
				}
				if a.query != "" {
					action = "清除搜索"
					onAction = func() { a.query = ""; a.filter() }
				}
				empty.Empty(c.Key("empty-results"), empty.Props{Icon: tableIcon, Title: title, Description: detail, ActionLabel: action, OnAction: onAction}).Absolute().Top(40).Left(0).Right(0).Bottom(0).Background(t.Background)
			}
		})
	})
}

func (a *application) resultTable(c *ui.Context) {
	t := c.Theme()
	cols := make([]ui.TableColumn, len(columnTitles))
	for i, title := range columnTitles {
		cols[i] = ui.TableColumn{Title: title, Width: 104, MinWidth: 88, Sortable: true}
		if i == 0 {
			cols[i].Width = 184
		}
		if i == 1 {
			cols[i].Width = 128
		}
		if i > 1 {
			cols[i].Align = ui.End
		}
	}
	a.table.Sort = &a.sort
	a.table.Selection = &a.selection
	a.table.Key = func(i int) any { return a.rows[i].identity() }
	a.table.CellMenu = func(m *ui.Menu, row, col int) {
		if row < 0 || row >= len(a.rows) || col < 0 || col >= len(columnTitles) {
			return
		}
		if m.Item("复制单元格").Chosen() {
			text := plainValue(a.rows[row].values()[col])
			if col < 2 {
				text = safeText(text)
			}
			a.copyText(c, text, 0)
		}
		if m.Item("复制此行").Chosen() {
			a.copyRows(c, []record{a.rows[row]})
		}
		if m.Item("复制筛选后的记录").Chosen() {
			a.copyRows(c, a.rows)
		}
	}
	e := table.Table(c.Key("results-table"), table.Props{State: &a.table, Columns: cols, Rows: len(a.rows), Cell: func(row, col int) {
		text := formatValue(a.rows[row].values()[col])
		color := t.Text
		if a.selection.Has(a.rows[row].identity()) {
			color = t.AccentText
		} else if text == "—" {
			color = t.TextMuted
		}
		ui.Text(c, text).FontSize(13).FontFeatures("tnum").TextColor(color).SingleLine().Tooltip(text)
	}}).Label("测试数据提取结果").Fill().MinHeight(0).Border(0, ui.Transparent).Radius(0)
	if e.Changed() {
		a.sortRows()
	}
	if e.Shortcut(ui.Cmd, ui.KeyC) {
		rows := a.selectedRows()
		if len(rows) > 0 {
			a.copyRows(c, rows)
		}
	}
	if e.Shortcut(0, ui.KeyEscape) {
		a.selection.Clear()
	}
}

func (a *application) issues(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Grow(1).Padding(16).Gap(12).Children(func() {
		if a.result == nil || len(a.result.Errors)+len(a.result.Infos) == 0 {
			ui.Text(c, "暂无查找提示").TextColor(t.TextMuted)
			return
		}
		for i, msg := range a.result.Errors {
			ui.Column(c.Key(fmt.Sprintf("error-%d", i))).Gap(4).Children(func() { ui.Text(c, "读取错误").FontSize(12).TextColor(t.Danger); ui.Text(c, msg).Selectable() })
		}
		for i, msg := range a.result.Infos {
			ui.Column(c.Key(fmt.Sprintf("info-%d", i))).Gap(4).Children(func() { ui.Text(c, "查找提示").FontSize(12).TextColor(t.TextMuted); ui.Text(c, msg).Selectable() })
		}
	})
}

func (a *application) footer(c *ui.Context) {
	t := c.Theme()
	color := t.TextMuted
	if a.statusError {
		color = t.Danger
	}
	ui.Row(c).Height(32).Padding(0, 20).Gap(16).Children(func() {
		ui.Text(c, a.status).Role(ui.RoleStatus).FontSize(12).TextColor(color).SingleLine().Tooltip(a.status).Grow(1)
		if a.result != nil {
			if a.selection.Len() > 0 {
				ui.Textf(c, "选中 %d 条", a.selection.Len()).FontSize(12).TextColor(t.TextMuted)
			}
			ui.Text(c, fmt.Sprintf("显示 %d / %d 条", len(a.rows), len(a.result.Records))).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted)
		}
		if strings.TrimSpace(a.receipt) != "" {
			ui.Text(c, a.receipt).FontSize(12).TextColor(t.TextMuted)
		}
	})
}
