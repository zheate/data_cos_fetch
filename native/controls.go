package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	tokens "github.com/smallnest/uix/components/base"
	"github.com/smallnest/uix/components/button"
	"github.com/smallnest/uix/components/checkbox"
	"github.com/smallnest/uix/components/label"
	"github.com/smallnest/uix/components/togglegroup"
)

// Install uix's tokens once per view, retaining the product's accent and
// respecting system text scaling, contrast, and reduced-motion preferences.
func workbenchTheme(c *ui.Context) *ui.Theme {
	t := tokens.For(c.Theme().Dark)
	if !t.Dark {
		t.TextMuted = ui.Hex("#62626b")
	}
	t.FontSize = 13
	t.Spacing = 4
	t = t.WithPreferences(c.Preferences()).WithAccent(ui.Hex("#2459bd"))
	return t
}

func fieldLabel(c *ui.Context, text string) {
	label.Label(c, label.Props{Text: text}).FontSize(13)
}

func workbenchButton(c *ui.Context, p button.Props) ui.Element {
	// uix v0.2.3 rounds its custom face but leaves the element radius at
	// zero. Match it so the border and keyboard focus ring follow the face.
	return button.Button(c, p).Radius(c.Theme().Radius)
}

func quietButton(c *ui.Context, key, text string, disabled bool, action func()) ui.Element {
	return workbenchButton(c.Key(key), button.Props{Label: text, Variant: button.Ghost, Size: button.Sm, Disabled: disabled, OnClick: action}).Height(28).Padding(4, 8).FontSize(12)
}

func stationCheckbox(c *ui.Context, key, name string, checked *bool, disabled bool) ui.Element {
	e := checkbox.Checkbox(c.Key(key), checkbox.Props{Checked: checked, Disabled: disabled}).Label(name).Grow(1).Basis(0).MinWidth(0).Height(32).Padding(6).Radius(6).FontSize(13)
	e.Children(func() { ui.Text(c, name).SingleLine().Grow(1).MinWidth(0) })
	return e
}

func (a *application) currentPresets(c *ui.Context) {
	maximum := strings.TrimSpace(a.config.Currents) == ""
	all := strings.EqualFold(strings.TrimSpace(a.config.Currents), "a")
	wasMaximum, wasAll := maximum, all
	togglegroup.ToggleGroup(c.Key("current-presets"), togglegroup.Props{
		Items: []togglegroup.Item{
			{On: &maximum, Label: "最大值", Tip: "提取各文件的最大电流点"},
			{On: &all, Label: "全部", Tip: "提取全部电流点"},
		},
		Disabled: a.busy,
	}).Label("电流快捷值").FillEqually().Width(132).Height(28).FontSize(12)
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
