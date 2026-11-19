package main

import (
	"fmt"
	"math"
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
	if a.config.Theme == "light" || a.config.Theme == "dark" {
		base := *c.Theme()
		base.Dark = a.config.Theme == "dark"
		c.SetTheme(&base)
	}
	t := workbenchTheme(c)
	c.SetTheme(t)
	if a.zoom <= 0 {
		a.zoom = defaultWorkbenchZoom
	}
	if c.Shortcut(ui.Cmd, ui.KeyEqual) || c.Shortcut(ui.Cmd|ui.Shift, ui.KeyEqual) {
		a.adjustZoom(workbenchZoomStep)
	}
	if c.Shortcut(ui.Cmd, ui.KeyMinus) || c.Shortcut(ui.Cmd|ui.Shift, ui.KeyMinus) {
		a.adjustZoom(-workbenchZoomStep)
	}
	if c.Shortcut(ui.Cmd, ui.Key0) {
		a.setZoom(defaultWorkbenchZoom)
	}
	if c.Zoom() != a.zoom {
		c.SetZoom(a.zoom)
	}
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
	a.sidebarWidth = max(304, min(a.sidebarWidth, maxSidebar))
	ui.Column(c).Fill().Background(t.Background).ObserveInput(func(ev ui.InputEvent) {
		if ev.Kind == ui.InputKeyDown && ev.Mods&ui.Cmd != 0 {
			switch {
			case ev.Code == ui.CodeNumpadAdd && ev.Key != ui.KeyEqual:
				a.adjustZoom(workbenchZoomStep)
				c.SetZoom(a.zoom)
			case ev.Code == ui.CodeNumpadSubtract && ev.Key != ui.KeyMinus:
				a.adjustZoom(-workbenchZoomStep)
				c.SetZoom(a.zoom)
			}
		}
	}).Children(func() {
		a.header(c)
		split := components.Split(c.Key("workbench-split"), &a.sidebarWidth, func() {
			a.configuration(c)
		}, func() {
			ui.Column(c).Fill().Padding(0, 12, 0, 12).Children(func() {
				a.results(c, width-a.sidebarWidth-32)
			})
		}, components.SplitOptions{FirstMin: 304, SecondMin: 560, Default: 304})
		split.Grow(1).MinHeight(0).FillWidth()
		if split.Changed() {
			a.config.SidebarWidth = a.sidebarWidth
		}
		px, _, splitOver := split.PointerPosition()
		if !a.busy && ((splitOver && math.Abs(float64(px-a.sidebarWidth)) <= 5) || split.Changed()) {
			split.DrawOver(func(p *ui.Painter, r ui.Rect) {
				x := r.X + a.sidebarWidth
				p.Fill(ui.Rect{X: x, Y: r.Y, W: 1, H: r.H}, t.Accent.Alpha(0.55), 0)
				pillH := min(float32(28), r.H)
				p.Fill(ui.Rect{X: x - 1, Y: r.Y + (r.H-pillH)/2, W: 3, H: pillH}, t.Accent.Alpha(0.85), 1.5)
			})
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
		themeIcon, themeLabel := lucide.Moon(), "切换到深色模式"
		if t.Dark {
			themeIcon, themeLabel = lucide.Sun(), "切换到浅色模式"
		}
		smallIconButton(c, "theme-toggle", themeLabel, themeIcon, false).OnClick(func() {
			a.toggleTheme(t.Dark)
			c.Invalidate()
		})
		rootBtn := components.Button(c.Key("root-shortcut"), "", components.ButtonGhost).
			Disabled(a.busy).
			Label("当前数据目录："+rootText).
			Height(28).Padding(0, 8).MaxWidth(280).
			OnClick(func() { a.settingsOpen = true }).
			Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).MinWidth(0).Children(func() {
					components.Icon(c, lucide.Folder()).Size(14, 14).TextColor(t.TextMuted).Shrink(0)
					components.Text(c, rootText).FontSize(12).TextColor(t.TextMuted).SingleLine().MinWidth(0)
				})
			})
		components.Tooltip(c, rootBtn, rootTip)
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
	ui.Column(c).Fill().Padding(12, 16).Gap(16).Dividers(1, t.Border.Alpha(0.65)).Children(func() {
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
	ui.Column(c).Grow(1).Basis(0).MinHeight(162).Gap(6).Children(func() {
		ui.Row(c).Height(28).Gap(4).AlignItems(ui.Center).Children(func() {
			fieldLabel(c, "编号 / 完整路径")
			ui.Textf(c, "%d 个", len(entries)).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted).Grow(1)
			smallIconButton(c, "clear-entries", "清空编号", lucide.Trash(), a.busy || a.config.Entries == "").OnClick(func() {
				a.config.Entries = ""
				a.entriesField.Focus()
			})
		})
		box := components.TextBox(c.Key("entries"), &a.config.Entries, components.TextBoxOptions{Multiline: true}).
			Label("编号或完整路径清单").
			Placeholder("每行一个编号或完整路径").
			Disabled(a.busy).
			Bind(&a.entriesField).
			Lines(5, 5).
			Grow(1).MinHeight(120).FontSize(14)
		if !a.busy {
			if box.FileDragOver() {
				box.Border(1, t.Accent)
			}
			if dropped := box.DroppedFiles(); len(dropped) > 0 {
				a.handleDroppedEntries(dropped)
			}
		}
	})
}

func (a *application) currentInput(c *ui.Context) {
	t := c.Theme()
	if a.config.Currents != a.lastConfigCurrents {
		a.syncCurrentMode()
	}
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Row(c).Height(28).Gap(8).AlignItems(ui.Center).Children(func() {
			fieldLabel(c, "电流点")
			components.Text(c, "A").FontSize(12).TextColor(t.TextMuted)
			ui.Spacer(c)
			tabs := components.Tabs(c.Key("current-mode"), &a.currentMode, components.TabsSegmented, "最大值", "全部", "指定")
			tabs.List.Label("电流模式").Disabled(a.busy)
			if tabs.List.Changed() {
				switch a.currentMode {
				case 0:
					if cur := strings.TrimSpace(a.config.Currents); cur != "" && !strings.EqualFold(cur, "a") {
						a.customCurrents = a.config.Currents
					}
					a.config.Currents = ""
				case 1:
					if cur := strings.TrimSpace(a.config.Currents); cur != "" && !strings.EqualFold(cur, "a") {
						a.customCurrents = a.config.Currents
					}
					a.config.Currents = "a"
				case 2:
					a.config.Currents = a.customCurrents
					a.currentsField.Focus()
				}
				a.lastConfigCurrents = a.config.Currents
			}
		})
		if a.currentMode == 2 {
			_, err := parseCurrents(a.config.Currents)
			components.TextBox(c.Key("currents"), &a.config.Currents, components.TextBoxOptions{}).
				Label("电流点 / A").
				Placeholder("例如 2, 4, 8~12").
				Disabled(a.busy).
				Bind(&a.currentsField).
				Height(34).
				Invalid(err != nil)
			if cur := strings.TrimSpace(a.config.Currents); cur != "" && !strings.EqualFold(cur, "a") {
				a.customCurrents = a.config.Currents
			}
			a.lastConfigCurrents = a.config.Currents
		}
	})
}

func (a *application) stationInput(c *ui.Context) {
	t := c.Theme()
	stations := a.config.stations()
	rows := (len(stations) + 1) / 2
	// Reserve the station content height before assigning spare space to entries.
	stationHeight := float32(36 + max(1, rows)*32 + max(0, rows-1)*6)
	if a.config.Mode == 1 {
		stationHeight = 48
	}
	ui.Column(c).Height(stationHeight).MinHeight(0).Shrink(1).Gap(8).Disabled(a.busy).Children(func() {
		if a.config.Mode == 1 {
			fieldLabel(c, "芯片测试")
			components.Text(c, "自动查找 postQCW 目录").FontSize(13).TextColor(t.TextMuted)
			return
		}
		selected := len(a.config.selectedTests())
		deletable := len(a.config.selectedDeletableTests())
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
			smallIconButton(c, "custom-button", "自定义测试站别", lucide.Plus(), false).OnClick(a.openCustomTests)
			delBtn := smallIconButton(c, "delete-tests", "删除选中站别", lucide.Trash(), a.busy || deletable == 0).OnClick(a.deleteSelectedStations)
			if !a.busy && deletable > 0 && delBtn.Hovered() {
				delBtn.TextColor(t.Danger)
			}
		})
		count := len(stations)
		if count == 0 {
			components.Text(c, "暂无站别，点击「新增站别」添加").FontSize(12).TextColor(t.TextMuted).MinHeight(32)
			return
		}
		// Scroll only after the entry input reaches its minimum height.
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
	hint := extractShortcutLabel() + " 提取"
	if err != nil {
		hint = "请检查查找提示"
	}
	if a.busy {
		hint = "正在提取；取消会保留上次结果"
	}
	summary := ""
	if a.busy {
		summary = "正在读取测试文件，可随时取消"
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
		if summary != "" {
			ui.Row(c).Height(16).Gap(8).AlignItems(ui.Center).Children(func() {
				summaryEl := components.Text(c, summary).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted).SingleLine().Grow(1).MinWidth(0)
				components.Tooltip(c, summaryEl, summary)
			})
		}
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
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			workbenchButton(c, "diagnostics", "读取环境检测", components.ButtonGhost, a.busy, a.diagnostics)
			ui.Spacer(c)
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
	searchBox := components.TextBox(c.Key("search"), &a.query, components.TextBoxOptions{LeadingIcon: lucide.Search()}).
		Label("搜索编号或站别").
		Placeholder("搜索编号或站别").
		Disabled(a.result == nil || a.resultTab == 2).
		Bind(&a.searchField).
		Width(width)
	if searchBox.Changed() {
		a.filter()
	}
	if searchBox.Shortcut(0, ui.KeyEscape) && a.query != "" {
		a.query = ""
		a.filter()
	}
}

func (a *application) results(c *ui.Context, width float32) {
	t := c.Theme()
	n, issues := 0, a.issueCount()
	if a.result != nil {
		n = len(a.result.Records)
	}
	ui.Column(c).Grow(1).MinHeight(0).Background(t.Surface).Clip().Role(ui.RoleRegion).Label("提取结果工作区").Children(func() {
		ui.Column(c).Padding(8, 0).Gap(8).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				resultLabel := "提取结果"
				if a.result != nil {
					resultLabel = fmt.Sprintf("提取结果  %d", n)
				}
				issuesLabel := "查找提示"
				if issues > 0 {
					issuesLabel = fmt.Sprintf("查找提示  %d", issues)
				}
				components.Tabs(
					c.Key("result-tabs"),
					&a.resultTab,
					components.TabsUnderline,
					resultLabel,
					"数据图表",
					issuesLabel,
				).List.Label("结果视图")
				ui.Spacer(c)
				if width >= 760 {
					a.resultSearch(c, 224)
					if a.query != "" && a.resultTab != 2 {
						smallIconButton(c, "clear-search", "清空搜索条件", lucide.X(), false).OnClick(func() {
							a.query = ""
							a.filter()
							a.searchField.Focus()
						})
					}
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
				ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
					searchW := max(160, width-26)
					if a.query != "" && a.resultTab != 2 {
						searchW = max(128, searchW-32)
					}
					a.resultSearch(c, searchW)
					if a.query != "" && a.resultTab != 2 {
						smallIconButton(c, "clear-search", "清空搜索条件", lucide.X(), false).OnClick(func() {
							a.query = ""
							a.filter()
							a.searchField.Focus()
						})
					}
				})
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
				a.resultTable(c, width+7)
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

func (a *application) resultTable(c *ui.Context, availWidth float32) {
	t := c.Theme()
	baseWidths := [10]float32{94, 84, 64, 64, 64, 64, 72, 96, 90, 84}
	minWidths := [10]float32{88, 80, 60, 60, 60, 60, 68, 92, 86, 80}
	const totalBaseWidth float32 = 776
	targetWidth := float32(math.Floor(float64(availWidth)))
	scale := float32(1)
	if targetWidth > totalBaseWidth {
		scale = targetWidth / totalBaseWidth
	}
	cols := make([]ui.TableColumn, len(columnTitles))
	var sumWidth float32
	for i, title := range columnTitles {
		w := baseWidths[i]
		if scale > 1 {
			if i == len(columnTitles)-1 {
				w = max(baseWidths[i], targetWidth-sumWidth)
			} else {
				w = float32(math.Floor(float64(baseWidths[i] * scale)))
			}
		}
		sumWidth += w
		cols[i] = ui.TableColumn{
			Title:    title,
			Width:    w,
			MinWidth: minWidths[i],
			Sortable: true,
		}
		if i > 1 {
			cols[i].Align = ui.Center
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
	a.table.List.OnSubmit = a.focusRowInChart
	a.table.List.Key = func(i int) any {
		if i < 0 || i >= len(a.rows) {
			return nil
		}
		return a.rows[i].identity()
	}
	groupedByEntry := a.sort.Column == "" || a.sort.Column == columnTitles[0]
	groups := tableGroups(a.rows, groupedByEntry)
	var tableOver bool
	var py, tableTop float32
	e := components.TableVirtualized(c.Key("results-table"), &a.table, cols, len(a.rows), func(row, col int) {
		if row < 0 || row >= len(a.rows) || col < 0 || col >= len(columnTitles) {
			return
		}
		text := formatValue(a.rows[row].values()[col])
		firstVisible, _ := a.table.List.Visible()
		group := groups[row]
		ghostText := ""
		if groupedByEntry && row != firstVisible {
			if col == 0 && !group.entryStart {
				ghostText = text
				text = ""
			} else if col == 1 && !group.stationStart {
				ghostText = text
				text = ""
			}
		}
		color := t.Text
		if text == "—" {
			color = t.TextMuted
		}
		cell := ui.Row(c).FillWidth().Height(20).AlignItems(ui.Center)
		// Cover the library's per-row zebra stripe with station-level grouping.
		bg := t.Surface
		if group.shaded && col > 0 {
			bg = t.Surface.Mix(t.Text, 0.025)
		}
		selectedRow := a.selection.Has(a.rows[row].identity())
		if selectedRow {
			bg = t.Surface.Mix(t.Text, 0.1)
		}
		line := t.Border.Alpha(0.5)
		if row == 0 {
			line = ui.Transparent
		} else if groupedByEntry && group.entryStart {
			line = t.TextMuted.Alpha(0.45)
		} else if groupedByEntry && group.stationStart && col > 0 {
			line = t.Border
		} else if groupedByEntry && (col == 0 || col == 1) {
			line = ui.Transparent
		}
		// Onevra cells own 6px vertical and 10px horizontal padding. Paint
		// through that padding while retaining its table layout and input behavior.
		cell.Draw(func(p *ui.Painter, r ui.Rect) {
			fullRect := ui.Rect{X: r.X - 10, Y: r.Y - 6, W: r.W + 20, H: r.H + 12}
			rowHovered := tableOver && py >= 32 && (tableTop+py) >= fullRect.Y && (tableTop+py) < fullRect.Y+fullRect.H
			drawBg := bg
			if rowHovered {
				if selectedRow {
					drawBg = t.Surface.Mix(t.Text, 0.13)
				} else {
					drawBg = bg.Mix(t.Text, 0.045)
				}
			}
			p.Fill(fullRect, drawBg, 0)
			p.Fill(ui.Rect{X: fullRect.X, Y: fullRect.Y, W: fullRect.W, H: 1}, line, 0)
			if rowHovered && ghostText != "" {
				span := ui.Span{Text: ghostText, Size: 13, Color: t.TextMuted.Alpha(0.65)}
				_, h := p.MeasureText(0, span)
				p.Clip(r, 0, func() {
					p.RichText(r.X, r.Y+(r.H-h)/2, 0, span)
				})
			}
		}).ContextMenu(func(m *ui.Menu) {
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
			if a.selection.Len() > 1 {
				if m.Item(fmt.Sprintf("复制选中记录（%d 条）", a.selection.Len())).Chosen() {
					a.copyRows(c, a.selectedRows())
				}
			}
			if m.Item("复制筛选后的记录").Chosen() {
				a.copyRows(c, a.rows)
			}
		})
		switch cols[col].Align {
		case ui.Center:
			cell.Justify(ui.Center)
		case ui.End:
			cell.Justify(ui.End)
		}
		cell.Children(func() {
			cellText := components.Text(c, text).FontSize(13).FontFeatures("tnum").TextColor(color).SingleLine()
			if col == 0 {
				cellText.FontWeight(600)
			}
			if text != "" {
				components.Tooltip(c, cellText, text)
			}
		})
	}).Label("测试数据提取结果").Fill().MinHeight(0)
	_, py, tableOver = e.PointerPosition()
	if a.settingsOpen || a.customOpen || a.dialogOpen {
		tableOver = false
	}
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		tableTop = r.Y
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 32}, t.Surface.Mix(t.Text, 0.025), 0)
	})
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
		if a.issueCount() == 0 {
			emptyStateWithIcon(c, "empty-issues", lucide.Info(), "暂无查找提示", "本次提取没有文件缺失或读取告警", "", nil)
			return
		}
		for i, msg := range a.issueErrors() {
			ui.Column(c.Key(fmt.Sprintf("error-%d", i))).Padding(10, 12).Radius(t.Radius).Border(1, t.Danger.Alpha(0.32)).Background(t.Danger.Alpha(0.06)).Gap(4).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(6, 6).Radius(3).Background(t.Danger)
					components.Text(c, "错误").FontSize(12).FontWeight(600).TextColor(t.Danger)
				})
				components.Text(c, msg).FontSize(13).Selectable()
			})
		}
		var infos []string
		if a.result != nil {
			infos = a.result.Infos
		}
		for i, msg := range infos {
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
	case a.busy:
		dotColor = t.Accent
	case a.statusError:
		dotColor = t.Danger
		textColor = t.Danger
	case a.result != nil && len(a.result.Errors) == 0:
		dotColor = t.Success
	}
	ui.Row(c).Height(28).Padding(0, 20).Gap(12).AlignItems(ui.Center).Background(t.Surface).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
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
			if len(a.rows) != len(a.result.Records) {
				components.Text(c, fmt.Sprintf("显示 %d / %d 条", len(a.rows), len(a.result.Records))).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted)
			}
		}
	})
}
