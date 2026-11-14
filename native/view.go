package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
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
		if a.resultTab == 2 {
			a.resultTab = 0
		}
		a.searchField.Focus()
	}
	width, _ := c.Size()
	maxSidebar := min(float32(520), width-560)
	a.sidebarWidth = max(320, min(a.sidebarWidth, maxSidebar))
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		a.header(c)
		split := components.Split(c.Key("workbench-split"), &a.sidebarWidth, func() {
			a.configuration(c)
		}, func() {
			ui.Column(c).Fill().Padding(12, 16, 12, 12).Children(func() {
				a.results(c, width-a.sidebarWidth-32)
			})
		}, components.SplitOptions{FirstMin: 320, SecondMin: 560, Default: 360})
		split.Grow(1).MinHeight(0).FillWidth()
		if split.Changed() {
			a.config.SidebarWidth = a.sidebarWidth
		}
		a.footer(c)
	})
	ui.Box(c.Key("settings-dialog-layer")).Children(func() {
		a.settingsDialog(c)
	})
	ui.Box(c.Key("custom-tests-dialog-layer")).Children(func() {
		a.customTestsDialog(c)
	})
	ui.Box(c.Key("diagnostics-dialog-layer")).Children(func() {
		a.diagnosticsDialog(c)
	})
}

func (a *application) header(c *ui.Context) {
	t := c.Theme()
	rootText, rootTip := rootSummary(a.config)
	ui.Row(c).Height(48).Padding(0, 20).Gap(10).AlignItems(ui.Center).Background(t.Surface).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		ui.Box(c).Size(24, 24).Radius(6).Background(t.Accent).Center().Children(func() {
			components.Icon(c, lucide.Table2()).Size(14, 14).TextColor(t.AccentText)
		})
		components.Text(c, "数据提取").FontSize(15).FontWeight(600)
		ui.Spacer(c)
		rootBtn := components.Button(c.Key("root-shortcut"), "", components.ButtonGhost).
			Disabled(a.busy).
			Label("当前数据目录：" + rootText).
			Height(28).Padding(0, 8).MaxWidth(280).
			OnClick(func() { a.settingsOpen = true }).
			Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).MinWidth(0).Children(func() {
					components.Icon(c, lucide.Folder()).Size(14, 14).TextColor(t.TextMuted).Shrink(0)
					components.Text(c, rootText).FontSize(12).TextColor(t.TextMuted).SingleLine().MinWidth(0)
				})
			})
		components.Tooltip(c, rootBtn, rootTip)
		iconButton(c, "diagnostics", "读取环境检测", lucide.ShieldCheck(), a.busy).OnClick(a.diagnostics)
		iconButton(c, "settings", "数据目录与测试文件设置", lucide.SlidersHorizontal(), a.busy).OnClick(func() { a.settingsOpen = true })
		ui.Box(c).Width(1).Height(16).Background(t.Border)
		exportDisabled := a.busy || a.result == nil || len(a.result.Records) == 0
		exportBtn := components.Button(c.Key("export"), "导出结果", components.ButtonOutline).
			Disabled(exportDisabled).Height(30)
		if !exportDisabled {
			shortcut := "Ctrl+E"
			if runtime.GOOS == "darwin" {
				shortcut = "⌘E"
			}
			components.MenuButton(c, exportBtn, func(m *components.Menu) {
				if m.Item("Excel 工作簿（全部记录）").Shortcut(shortcut).Chosen() {
					a.export("xlsx")
				}
				if m.Item("CSV 文件（全部记录）").Chosen() {
					a.export("csv")
				}
			})
		}
	})
}

func (a *application) configuration(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(12, 16, 12, 20).Gap(16).Dividers(1, t.Border.Alpha(0.65)).Children(func() {
		ui.Row(c).Height(28).Gap(12).AlignItems(ui.Center).Children(func() {
			fieldLabel(c, "提取条件")
			ui.Spacer(c)
			components.Tabs(c.Key("mode"), &a.config.Mode, components.TabsSegmented, "模块", "芯片").
				List.Label("数据模式").Disabled(a.busy)
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
	ui.Column(c).Height(162).Shrink(0).Gap(6).Children(func() {
		ui.Row(c).Height(28).Gap(4).AlignItems(ui.Center).Children(func() {
			fieldLabel(c, "编号 / 完整路径")
			ui.Textf(c, "%d 个", len(entries)).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted).Grow(1)
			smallIconButton(c, "import-entries", "导入文本清单", lucide.Import(), a.busy).OnClick(a.importEntries)
			smallIconButton(c, "tidy-entries", "整理编号清单", lucide.Broom(), a.busy || len(entries) == 0).OnClick(a.tidyEntries)
			smallIconButton(c, "clear-entries", "清空编号", lucide.Trash(), a.busy || a.config.Entries == "").OnClick(func() {
				a.config.Entries = ""
				a.entriesField.Focus()
			})
		})
		components.TextBox(c.Key("entries"), &a.config.Entries, components.TextBoxOptions{Multiline: true}).
			Label("编号或完整路径清单").
			Placeholder("每行一个编号或完整路径").
			Disabled(a.busy).
			Bind(&a.entriesField).
			Lines(5, 5).
			Grow(1).MinHeight(120).FontSize(14)
	})
}

func (a *application) currentInput(c *ui.Context) {
	t := c.Theme()
	_, err := parseCurrents(a.config.Currents)
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Row(c).Height(28).Gap(4).AlignItems(ui.Center).Children(func() {
			fieldLabel(c, "电流点")
			components.Text(c, "A").FontSize(12).TextColor(t.TextMuted).Grow(1)
			a.currentPresets(c)
		})
		e := components.TextBox(c.Key("currents"), &a.config.Currents, components.TextBoxOptions{}).
			Label("电流点 / A").
			Placeholder("例如 2, 4, 8~12").
			Disabled(a.busy).
			Height(34).
			Invalid(err != nil)
		if err != nil {
			e.Border(1, t.Danger)
			errorText := err.Error()
			ui.Row(c).MinHeight(20).Children(func() {
				errorMessage := components.Text(c.Key("current-error"), errorText).
					FontSize(12).TextColor(t.Danger).SingleLine().Grow(1).MinWidth(0)
				components.Tooltip(c, errorMessage, errorText)
				e.ErrorMessage(errorMessage)
			})
		}
	})
}

func (a *application) stationInput(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).MinHeight(0).Gap(8).Disabled(a.busy).Children(func() {
		if a.config.Mode == 1 {
			fieldLabel(c, "芯片测试")
			components.Text(c, "自动查找 postQCW 目录").FontSize(13).TextColor(t.TextMuted)
			return
		}
		stations := a.config.stations()
		selected := len(a.config.selectedTests())
		allSelected := len(stations) > 0 && selected == len(stations)
		ui.Row(c).Height(28).Gap(4).AlignItems(ui.Center).Children(func() {
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
			delBtn := quietButton(c, "delete-tests", "删除", a.busy || selected == 0, a.deleteSelectedStations).Label("删除选中站别")
			if !a.busy && selected > 0 {
				delBtn.TextColor(t.Danger)
			}
		})
		count := len(stations)
		if count == 0 {
			components.Text(c, "暂无站别，点击「新增站别」添加").FontSize(12).TextColor(t.TextMuted).MinHeight(32)
			return
		}
		rows := (count + 1) / 2
		// Give stations the remaining height while keeping entry and run controls visible.
		components.ScrollView(c.Key("station-list"), components.ScrollViewOptions{}).Label("测试站别列表").Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				for row := 0; row < rows; row++ {
					ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Stretch).Children(func() {
						for col := 0; col < 2; col++ {
							i := row*2 + col
							if i < count {
								a.stationChoice(c, stations[i])
							} else {
								ui.Spacer(c).Grow(1).Basis(0).Padding(6, 8).Border(1, ui.Transparent)
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
}

func (a *application) runAction(c *ui.Context) {
	t := c.Theme()
	_, err := a.config.request()
	hint := "Ctrl / ⌘ + Enter 提取"
	if err != nil {
		hint = err.Error()
	}
	if a.busy {
		hint = "正在提取；取消会保留上次结果"
	}
	summary, isError := runSummary(a.config, a.busy, err)
	summaryColor := t.TextMuted
	if isError {
		summaryColor = t.Danger
	}
	ui.Column(c).Gap(6).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			if a.busy {
				workbenchButton(c, "cancel", "取消", components.ButtonSecondary, false, func() {
					if a.cancel != nil {
						a.cancel()
					}
					a.status = "正在取消"
				}).Height(42)
			}
			runBtn := workbenchButton(c, "run", "开始提取", components.ButtonPrimary, a.busy || err != nil, a.run).
				Height(42).Grow(1).Justify(ui.Center).FontSize(14).FontWeight(500)
			components.Tooltip(c, runBtn, hint)
		})
		ui.Row(c).Height(16).Gap(8).AlignItems(ui.Center).Children(func() {
			summaryEl := components.Text(c, summary).FontSize(12).FontFeatures("tnum").TextColor(summaryColor).SingleLine().Grow(1).MinWidth(0)
			components.Tooltip(c, summaryEl, summary)
			components.Text(c, extractShortcutLabel()).FontSize(11).FontFeatures("tnum").TextColor(t.TextMuted).Shrink(0)
		})
	})
}

func (a *application) settingsDialog(c *ui.Context) {
	t := c.Theme()
	components.Dialog(c.Key("settings-dialog"), &a.settingsOpen, components.DialogOptions{Label: "数据目录与测试文件"}, func(panel ui.Element) {
		panel.Width(560).Gap(16)
		components.Text(c, "数据目录与测试文件").FontSize(16).FontWeight(600)
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			fieldLabel(c, "数据根目录")
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				if a.config.Mode == 0 {
					components.TextBox(c.Key("module-root"), &a.config.ModuleRoot, components.TextBoxOptions{}).
						Label("模块根目录").Grow(1)
				} else {
					components.TextBox(c.Key("chip-roots"), &a.config.ChipRoots, components.TextBoxOptions{Multiline: true}).
						Label("芯片根目录，每行一个").Lines(4, 5).Grow(1).Height(112)
				}
				iconButton(c, "choose-root", "选择数据根目录", lucide.Folder(), a.busy).OnClick(a.chooseRoot)
			})
			components.Text(c, "完整路径可以直接提取；编号使用此根目录查找").FontSize(12).TextColor(t.TextMuted)
		})
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			fieldLabel(c, "测试文件")
			ui.Row(c).Gap(24).Children(func() {
				for i, name := range measurements {
					components.Checkbox(c.Key("measurement-"+name), &a.config.Measurements[i], name)
				}
			})
		})
		ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
			workbenchButton(c, "", "完成", components.ButtonSecondary, false, func() { a.settingsOpen = false })
		})
	})
}

func (a *application) customTestsDialog(c *ui.Context) {
	t := c.Theme()
	components.Dialog(c.Key("custom-tests-dialog"), &a.customOpen, components.DialogOptions{Label: "自定义测试站别"}, func(panel ui.Element) {
		panel.Width(460).Gap(14)
		ui.Column(c).FillWidth().Gap(4).Children(func() {
			components.Text(c, "自定义测试站别").FontSize(16).FontWeight(600)
			components.Text(c, "每行一个站别，目录名与测试文件夹一致").FontSize(12).TextColor(t.TextMuted)
		})
		components.TextBox(c.Key("custom-tests"), &a.customDraft, components.TextBoxOptions{Multiline: true}).
			Label("自定义测试站别清单").
			Lines(6, 6).FillWidth().Height(160)
		ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
			workbenchButton(c, "", "取消", components.ButtonSecondary, false, func() { a.customOpen = false })
			workbenchButton(c, "", "保存站别", components.ButtonPrimary, false, a.saveCustomTests)
		})
	})
}

func (a *application) diagnosticsDialog(c *ui.Context) {
	t := c.Theme()
	components.Dialog(c.Key("diagnostics-dialog"), &a.dialogOpen, components.DialogOptions{Label: a.dialogTitle}, func(panel ui.Element) {
		panel.Width(600).Gap(12)
		components.Text(c, a.dialogTitle).FontSize(16).FontWeight(600)
		components.ScrollView(c.Key("diagnostics-scroll"), components.ScrollViewOptions{}).
			FillWidth().Height(320).Padding(12).Border(1, t.Border).Radius(t.Radius).Background(t.Surface).Children(func() {
			components.Text(c, a.dialogText).Selectable().FontSize(13)
		})
		ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
			workbenchButton(c, "", "关闭", components.ButtonSecondary, false, func() { a.dialogOpen = false })
		})
	})
}

func (a *application) resultSearch(c *ui.Context, width float32) {
	if components.TextBox(c.Key("search"), &a.query, components.TextBoxOptions{LeadingIcon: lucide.Search()}).
		Label("搜索编号或站别").
		Placeholder("搜索编号或站别").
		Disabled(a.result == nil || a.resultTab == 2).
		Bind(&a.searchField).
		Width(width).Changed() {
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
	ui.Column(c).Grow(1).MinHeight(0).Border(1, t.Border).Radius(10).Background(t.Surface).Clip().Role(ui.RoleRegion).Label("提取结果工作区").Children(func() {
		ui.Column(c).Padding(10, 12).Gap(8).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				components.Tabs(
					c.Key("result-tabs"),
					&a.resultTab,
					components.TabsSegmented,
					fmt.Sprintf("提取结果  %d", n),
					"数据图表",
					fmt.Sprintf("查找提示  %d", issues),
				).List.Label("结果视图")
				ui.Spacer(c)
				if width >= 760 {
					a.resultSearch(c, 224)
				}
				label := "复制筛选后的记录"
				if a.selection.Len() > 0 {
					label = "复制选中记录"
				}
				iconButton(c, "copy-results", label, lucide.Copy(), len(a.rows) == 0 || a.resultTab == 2).OnClick(func() {
					rows := a.selectedRows()
					if len(rows) == 0 {
						rows = a.rows
					}
					a.copyRows(c, rows)
				})
				iconButton(c, "clear-results", "清空提取结果", lucide.Trash(), a.busy || a.result == nil).OnClick(a.clearResult)
			})
			if width < 760 {
				a.resultSearch(c, max(160, width-26))
			}
		})
		ui.Box(c).FillWidth().Height(1).Background(t.Border)
		switch a.resultTab {
		case 1:
			ui.Box(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
				a.chartTabView(c)
			})
		case 2:
			a.issues(c)
		default:
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
					emptyState(c, "empty-results", title, detail, action, onAction).
						Absolute().Top(33).Left(0).Right(0).Bottom(0).Background(t.Surface)
				}
			})
		}
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
			cols[i].Width = 132
		}
		if i > 1 {
			cols[i].Align = ui.End
		}
	}
	sorted := false
	a.table.Sort = &a.sort
	a.table.OnSort = func() {
		a.sortRows()
		sorted = true
	}
	a.table.List.Selection = &a.selection
	a.table.List.KeepOffset = true
	a.table.List.Key = func(i int) any {
		if i < 0 || i >= len(a.rows) {
			return nil
		}
		return a.rows[i].identity()
	}
	e := components.TableVirtualized(c.Key("results-table"), &a.table, cols, len(a.rows), func(row, col int) {
		if row < 0 || row >= len(a.rows) || col < 0 || col >= len(columnTitles) {
			return
		}
		text := formatValue(a.rows[row].values()[col])
		color := t.Text
		if text == "—" {
			color = t.TextMuted
		}
		cell := ui.Row(c).FillWidth().AlignItems(ui.Center).ContextMenu(func(m *ui.Menu) {
			if m.Item("复制单元格").Chosen() {
				v := plainValue(a.rows[row].values()[col])
				if col < 2 {
					v = safeText(v)
				}
				a.copyText(c, v, 0)
			}
			if m.Item("复制此行").Chosen() {
				a.copyRows(c, []record{a.rows[row]})
			}
			if m.Item("复制筛选后的记录").Chosen() {
				a.copyRows(c, a.rows)
			}
		})
		if cols[col].Align == ui.End {
			cell.Justify(ui.End)
		}
		cell.Children(func() {
			cellText := components.Text(c, text).FontSize(13).FontFeatures("tnum").TextColor(color).SingleLine()
			components.Tooltip(c, cellText, text)
		})
	}).Label("测试数据提取结果").Fill().MinHeight(0)
	if sorted {
		a.table.List.Focus(c)
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
	components.ScrollView(c.Key("issues-scroll"), components.ScrollViewOptions{}).Grow(1).Padding(16).Gap(10).Children(func() {
		if a.result == nil || len(a.result.Errors)+len(a.result.Infos) == 0 {
			emptyStateWithIcon(c, "empty-issues", lucide.Info(), "暂无查找提示", "本次提取没有文件缺失或读取告警", "", nil)
			return
		}
		for i, msg := range a.result.Errors {
			ui.Column(c.Key(fmt.Sprintf("error-%d", i))).Padding(10, 12).Radius(t.Radius).Border(1, t.Danger.Alpha(0.32)).Background(t.Danger.Alpha(0.06)).Gap(4).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(6, 6).Radius(3).Background(t.Danger)
					components.Text(c, "读取错误").FontSize(12).FontWeight(600).TextColor(t.Danger)
				})
				components.Text(c, msg).FontSize(13).Selectable()
			})
		}
		for i, msg := range a.result.Infos {
			ui.Column(c.Key(fmt.Sprintf("info-%d", i))).Padding(10, 12).Radius(t.Radius).Border(1, t.Border).Background(t.Surface).Gap(4).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(6, 6).Radius(3).Background(t.TextMuted)
					components.Text(c, "查找提示").FontSize(12).FontWeight(600).TextColor(t.TextMuted)
				})
				components.Text(c, msg).FontSize(13).Selectable()
			})
		}
	})
}

func (a *application) footer(c *ui.Context) {
	t := c.Theme()
	statusText := a.status
	if statusText == "" {
		statusText = "就绪"
	}
	dotColor := t.TextMuted
	textColor := t.TextMuted
	switch {
	case a.statusError:
		dotColor = t.Danger
		textColor = t.Danger
	case a.busy:
		dotColor = t.Accent
	case a.result != nil && len(a.result.Errors) == 0:
		dotColor = t.Success
	}
	ui.Row(c).Height(32).Padding(0, 20).Gap(12).AlignItems(ui.Center).Background(t.Surface).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
			if a.busy {
				components.Spinner(c, components.SpinnerRing).Size(12, 12).Label("提取进行中")
			} else {
				ui.Box(c).Size(6, 6).Radius(3).Background(dotColor).Shrink(0)
			}
			statusEl := components.Text(c, statusText).Role(ui.RoleStatus).FontSize(12).TextColor(textColor).SingleLine().Grow(1).MinWidth(0)
			components.Tooltip(c, statusEl, statusText)
		})
		if a.result != nil {
			if a.selection.Len() > 0 {
				ui.Textf(c, "选中 %d 条", a.selection.Len()).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted)
			}
			components.Text(c, fmt.Sprintf("显示 %d / %d 条", len(a.rows), len(a.result.Records))).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted)
		}
		if strings.TrimSpace(a.receipt) != "" {
			components.Text(c, a.receipt).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted)
		}
	})
}
