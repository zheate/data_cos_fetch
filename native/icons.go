package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/smallnest/uix/components/button"
)

// Lucide icons, matching the existing product's icon set (ISC license).
func icon(path string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">` + path + `</svg>`))
}

var folderIcon = icon(`<path d="m6 14 1.5-3h14l-3 9H3a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1h5l2 2h8a1 1 0 0 1 1 1v4"/>`)
var importIcon = icon(`<path d="M12 3v12m-4-4 4 4 4-4M5 15v5h14v-5"/>`)
var trashIcon = icon(`<path d="M3 6h18M9 6V3h6v3M5 6l1 15h12l1-15M10 10v7m4-7v7"/>`)
var copyIcon = icon(`<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15H3V3h12v2"/>`)
var shieldIcon = icon(`<path d="M12 3 3 7v5c0 5 9 9 9 9s9-4 9-9V7l-9-4Z"/><path d="m8 12 3 3 5-6"/>`)
var slidersIcon = icon(`<path d="M4 21v-7m0-4V3m8 18v-9m0-4V3m8 18v-5m0-4V3M1 14h6m2-6h6m2 8h6"/>`)
var tableIcon = icon(`<rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18M9 9v12"/>`)
var tidyIcon = icon(`<path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/>`)

func iconButton(c *ui.Context, key, label string, svg *ui.SVG, disabled bool) ui.Element {
	return workbenchButton(c.Key(key), button.Props{Variant: button.Ghost, Size: button.Sm, Disabled: disabled}).Label(label).Tooltip(label).Size(32, 32).Padding(8).Children(func() { ui.Icon(c, svg).FontSize(16) })
}
