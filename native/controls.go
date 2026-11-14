package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"github.com/oneo-me/onevra/source/theme"
)

// Install a minimalist neutral monochrome palette over Onevra's base metrics,
// keeping compact 13 DIP desktop typography and system contrast/motion settings.
func workbenchTheme(c *ui.Context) *ui.Theme {
	base := theme.Onevra(c)
	t := *base
	if base.Dark {
		accent := ui.Hex("#e4e4e7")
		t.Background = ui.Hex("#121214")
		t.Surface = ui.Hex("#19191c")
		t.SurfaceHover = ui.Hex("#232328")
		t.SurfacePressed = ui.Hex("#2e2e34")
		t.Border = ui.Hex("#323238")
		t.Text = ui.Hex("#f4f4f5")
		t.TextMuted = ui.Hex("#a1a1aa")
		t.Accent = accent
		t.AccentHover = ui.Hex("#ffffff")
		t.AccentPressed = ui.Hex("#d4d4d8")
		t.AccentText = ui.Hex("#18181b")
		t.Danger = ui.Hex("#ef7d7d")
		t.Warning = ui.Hex("#e5b567")
		t.Success = ui.Hex("#6ee7b7")
		t.Selection = accent.Alpha(0.16)
		t.Focus = accent.Alpha(0.55)
		t.Scrollbar = t.Text.Alpha(0.32)
	} else {
		accent := ui.Hex("#18181b")
		t.Background = ui.Hex("#f6f7f8")
		t.Surface = ui.Hex("#ffffff")
		t.SurfaceHover = ui.Hex("#f1f2f4")
		t.SurfacePressed = ui.Hex("#e4e6ea")
		t.Border = ui.Hex("#e4e4e7")
		t.Text = ui.Hex("#18181b")
		t.TextMuted = ui.Hex("#66666e")
		t.Accent = accent
		t.AccentHover = ui.Hex("#2e2e33")
		t.AccentPressed = ui.Hex("#3f3f46")
		t.AccentText = ui.Hex("#ffffff")
		t.Danger = ui.Hex("#c93b3b")
		t.Warning = ui.Hex("#a16207")
		t.Success = ui.Hex("#16804a")
		t.Selection = accent.Alpha(0.09)
		t.Focus = accent.Alpha(0.45)
		t.Scrollbar = t.Text.Alpha(0.26)
	}
	if p := c.Preferences(); p.HighContrast {
		t.Border = t.Border.Mix(t.Text, 0.45)
		t.TextMuted = t.TextMuted.Mix(t.Text, 0.4)
		t.Focus.A = 255
		t.Scrollbar = t.Scrollbar.Alpha(1.6)
	}
	if base.FontSize > 0 {
		t.FontSize = base.FontSize * (13.0 / 14.0)
	} else {
		t.FontSize = 13
	}
	t.Spacing = 4
	return &t
}

func fieldLabel(c *ui.Context, text string) {
	components.Text(c, text).FontSize(13).FontWeight(600)
}

func workbenchButton(c *ui.Context, key, label string, variant components.ButtonVariant, disabled bool, action func()) ui.Element {
	ctx := c
	if key != "" {
		ctx = c.Key(key)
	}
	e := components.Button(ctx, label, variant).Disabled(disabled)
	if action != nil {
		e.OnClick(action)
	}
	return e
}

func quietButton(c *ui.Context, key, text string, disabled bool, action func()) ui.Element {
	return workbenchButton(c, key, text, components.ButtonGhost, disabled, action).
		Height(28).Padding(4, 8).FontSize(12)
}

func stationCheckbox(c *ui.Context, key, name string, checked *bool, disabled bool) ui.Element {
	t := c.Theme()
	bg := t.Surface
	border := t.Border
	fg := t.TextMuted
	weight := 400
	if *checked {
		bg = t.Accent.Alpha(0.045)
		border = t.Accent.Alpha(0.26)
		if t.Dark {
			bg = t.Accent.Alpha(0.10)
			border = t.Accent.Alpha(0.32)
		}
		fg = t.Text
		weight = 500
	}
	if disabled {
		bg = t.Background
		fg = t.TextMuted
	}
	return components.Checkbox(c.Key(key), checked, name).
		Disabled(disabled).
		Grow(1).Basis(0).MinWidth(0).MinHeight(32).Padding(5, 8).Radius(6).
		Background(bg).Border(1, border).FontSize(13).TextColor(fg).FontWeight(weight).Clip()
}

func emptyState(c *ui.Context, key, title, detail, action string, onAction func()) ui.Element {
	return emptyStateWithIcon(c, key, lucide.Table2(), title, detail, action, onAction)
}

func emptyStateWithIcon(c *ui.Context, key string, iconSVG *ui.SVG, title, detail, action string, onAction func()) ui.Element {
	t := c.Theme()
	return ui.Column(c.Key(key)).FillWidth().Grow(1).Center().Gap(10).Children(func() {
		ui.Box(c).Size(44, 44).Radius(12).Background(t.Accent.Alpha(0.1)).Center().Children(func() {
			components.Icon(c, iconSVG).Size(20, 20).TextColor(t.Accent)
		})
		ui.Column(c).Center().Gap(4).Children(func() {
			components.Text(c, title).FontSize(14).FontWeight(600)
			if detail != "" {
				components.Text(c, detail).FontSize(12).TextColor(t.TextMuted)
			}
		})
		if action != "" {
			workbenchButton(c, "", action, components.ButtonSecondary, false, onAction)
		}
	})
}

func rootSummary(s settings) (label, tip string) {
	if s.Mode == 1 {
		roots := lines(s.ChipRoots)
		switch len(roots) {
		case 0:
			return "未设置芯片目录", "点击设置芯片数据根目录"
		case 1:
			return roots[0], "当前芯片根目录（点击设置）\n" + roots[0]
		default:
			return fmt.Sprintf("%s 等 %d 个目录", roots[0], len(roots)), "当前芯片根目录（点击设置）\n" + strings.Join(roots, "\n")
		}
	}
	root := strings.TrimSpace(s.ModuleRoot)
	if root == "" {
		return "未设置根目录", "点击设置模块数据根目录"
	}
	return root, "当前模块根目录（点击设置）\n" + root
}

func runSummary(s settings, busy bool, err error) (text string, isError bool) {
	if busy {
		return "正在读取测试文件，可随时取消", false
	}
	if err != nil {
		return err.Error(), len(lines(s.Entries)) > 0
	}
	entryCount := len(lines(s.Entries))
	currentDesc := "最大电流"
	if strings.EqualFold(strings.TrimSpace(s.Currents), "a") {
		currentDesc = "全部电流"
	} else if strings.TrimSpace(s.Currents) != "" {
		currentDesc = "指定电流"
	}
	if s.Mode == 1 {
		return fmt.Sprintf("%d 个编号 · 芯片 postQCW · %s", entryCount, currentDesc), false
	}
	return fmt.Sprintf("%d 个编号 · %d 个站别 · %s", entryCount, len(s.selectedTests()), currentDesc), false
}

func extractShortcutLabel() string {
	if runtime.GOOS == "darwin" {
		return "⌘↵"
	}
	return "Ctrl+↵"
}

func (a *application) currentPresets(c *ui.Context) {
	maximum := strings.TrimSpace(a.config.Currents) == ""
	all := strings.EqualFold(strings.TrimSpace(a.config.Currents), "a")
	wasMaximum, wasAll := maximum, all
	ui.Row(c.Key("current-presets")).Label("电流快捷值").FocusGroup(ui.Horizontal).Gap(4).Children(func() {
		maxBtn := components.ToggleButton(c.Key("preset-max"), &maximum, "最大值").
			Disabled(a.busy).Height(28).Padding(4, 10).FontSize(12)
		components.Tooltip(c, maxBtn, "提取各文件的最大电流点")
		allBtn := components.ToggleButton(c.Key("preset-all"), &all, "全部").
			Disabled(a.busy).Height(28).Padding(4, 10).FontSize(12)
		components.Tooltip(c, allBtn, "提取全部电流点")
	})
	// Derive the pressed state from the text. A custom range leaves both
	// presets off; reactivating a chosen preset keeps its meaning unchanged.
	if maximum != wasMaximum {
		a.config.Currents = ""
	}
	if all != wasAll {
		a.config.Currents = "a"
	}
}

func (a *application) copyRows(c *ui.Context, rows []record) {
	data, err := delimited(rows, '\t')
	if err != nil {
		a.status = "复制失败：" + err.Error()
		a.statusError = true
		return
	}
	a.copyText(c, string(data), len(rows))
}

func (a *application) copyText(c *ui.Context, text string, count int) {
	c.WriteClipboardThen(text, func(err error) {
		if err != nil {
			a.status = "复制失败：" + err.Error()
			a.statusError = true
			return
		}
		a.statusError = false
		if count > 0 {
			a.status = fmt.Sprintf("复制 %d 条记录", count)
		} else {
			a.status = "复制单元格内容"
		}
	})
}
