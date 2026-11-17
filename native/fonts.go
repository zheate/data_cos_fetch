package main

import (
	_ "embed"
	"fmt"

	"github.com/egoist/mygo/ui"
)

const interfaceFont = "MiSans UI, Microsoft YaHei UI, Segoe UI, sans-serif"

// Reuse PLM's static Regular/Medium interface faces. Native renderers require
// TrueType/OpenType rather than PLM's web-only WOFF2 containers.
//
//go:embed assets/fonts/MiSans-Regular.ttf
var miSansRegular []byte

//go:embed assets/fonts/MiSans-Medium.ttf
var miSansMedium []byte

func init() {
	// Register before either an application window or a headless UI tester
	// measures text. Registration is process-local; no system install is needed.
	for _, face := range []struct {
		name string
		data []byte
	}{{"Regular", miSansRegular}, {"Medium", miSansMedium}} {
		if err := ui.RegisterFont(face.data, "MiSans UI"); err != nil {
			panic(fmt.Errorf("register bundled MiSans %s: %w", face.name, err))
		}
	}
}
