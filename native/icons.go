package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
)

func iconButton(c *ui.Context, key, label string, svg *ui.SVG, disabled bool) ui.Element {
	t := c.Theme()
	b := components.IconButton(c.Key(key), svg, label, components.IconButtonPlain).
		Disabled(disabled)
	if disabled {
		return b.Opacity(0.38)
	}
	b.Cursor(ui.CursorPointer)
	if b.Hovered() || b.Pressed() {
		b.TextColor(t.Text)
	}
	return b
}

func smallIconButton(c *ui.Context, key, label string, svg *ui.SVG, disabled bool) ui.Element {
	return iconButton(c, key, label, svg, disabled).Size(28, 28)
}
