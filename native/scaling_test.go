package main

import (
	"bytes"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestWorkbenchDefaultZoom(t *testing.T) {
	for _, initial := range []float32{newApplication().zoom, 0, -1} {
		t.Run(fmt.Sprintf("initial-%g", initial), func(t *testing.T) {
			a := newApplication()
			a.zoom = initial
			var zoom, width float32
			tt := ui.NewTester(func(c *ui.Context) {
				a.view(c)
				zoom = c.Zoom()
				width, _ = c.Size()
			}, 1440, 900)
			if zoom != 1.3 || math.Abs(float64(width-1440/float32(1.3))) > 0.01 {
				t.Fatalf("first frame did not render at 130%%: zoom=%g width=%g", zoom, width)
			}
			a.setZoom(1)
			tt.Frame()
			tt.Key(ui.Cmd, ui.Key0)
			if zoom != 1.3 || !tt.HasText("界面缩放 130%") {
				t.Fatalf("reset did not restore 130%%: zoom=%g status=%q", zoom, a.status)
			}
		})
	}
	a := newApplication()
	a.zoom = 0
	a.adjustZoom(workbenchZoomStep)
	if math.Abs(float64(a.zoom-1.4)) > 0.0001 {
		t.Fatalf("zoom adjustment did not start from 130%%: %g", a.zoom)
	}
}

// Exercise the real workbench with both inherited component typography and
// explicit font sizes. Text-only desktop scaling must not enlarge just one.
func TestWorkbenchTextScaleKeepsDesignMetrics(t *testing.T) {
	for _, size := range [][2]int{{1000, 720}, {1440, 842}} {
		for _, chart := range []bool{false, true} {
			for _, dark := range []bool{false, true} {
				t.Run(fmt.Sprintf("%dx%d/chart-%t/dark-%t", size[0], size[1], chart, dark), func(t *testing.T) {
					var reference []byte
					for _, textScale := range []float32{1, 1.51, 2} {
						a := redesignFixture()
						if chart {
							a = chartDesignFixture(false)
						}
						var fontSize float32
						tt := ui.NewTesterWith(func(c *ui.Context) {
							a.view(c)
							fontSize = c.Theme().FontSize
						}, size[0], size[1], ui.TesterOptions{Dark: dark, Preferences: ui.Preferences{TextScale: textScale}})
						if fontSize != 13 {
							t.Errorf("text scale %.2f changed the design font: got %.3f DIP, want 13 DIP", textScale, fontSize)
						}
						var encoded bytes.Buffer
						if err := png.Encode(&encoded, tt.Image()); err != nil {
							t.Fatal(err)
						}
						if textScale == 1 {
							reference = append([]byte(nil), encoded.Bytes()...)
						} else if !bytes.Equal(reference, encoded.Bytes()) {
							t.Errorf("text scale %.2f changed workbench typography or layout at the same display scale", textScale)
						}
						if dir := os.Getenv("DATA_COS_SCALING_CAPTURE_DIR"); dir != "" && textScale == 1.51 {
							if err := os.MkdirAll(dir, 0700); err != nil {
								t.Fatal(err)
							}
							path := filepath.Join(dir, fmt.Sprintf("%dx%d-chart-%t-dark-%t.png", size[0], size[1], chart, dark))
							if err := os.WriteFile(path, encoded.Bytes(), 0600); err != nil {
								t.Fatal(err)
							}
						}
					}
				})
			}
		}
	}
}

func TestWorkbenchDisplayScaleKeepsLogicalLayout(t *testing.T) {
	a := redesignFixture()
	tt := ui.NewTester(a.view, 1000, 720)
	names := []string{"编号或完整路径清单", "开始提取", "数据模式", "电流模式", "提取结果工作区"}
	baseline := make(map[string]ui.Rect)
	for _, name := range names {
		bounds, ok := tt.Find(name)
		if !ok {
			t.Fatalf("missing control %q", name)
		}
		baseline[name] = bounds
	}
	for _, scale := range []float32{1, 1.5, 2} {
		tt.SetScale(scale)
		for _, name := range names {
			if got, ok := tt.Find(name); !ok || got != baseline[name] {
				t.Errorf("display scale %.2f changed %q logical bounds: got %+v, want %+v", scale, name, got, baseline[name])
			}
		}
		if image := tt.Image(); image.Bounds().Dx() != int(1000*scale) || image.Bounds().Dy() != int(720*scale) {
			t.Fatalf("display scale %.2f did not change the physical render size: %v", scale, image.Bounds())
		}
	}
}
