package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
)

func iconButton(c *ui.Context, key, label string, svg *ui.SVG, disabled bool) ui.Element {
	return components.IconButton(c.Key(key), svg, label, components.IconButtonPlain).
		Disabled(disabled)
}

func smallIconButton(c *ui.Context, key, label string, svg *ui.SVG, disabled bool) ui.Element {
	return components.IconButton(c.Key(key), svg, label, components.IconButtonPlain).
		Size(28, 28).Disabled(disabled)
}
