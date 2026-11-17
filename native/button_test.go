package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
)

func saveButtonFrame(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	if dir := os.Getenv("DATA_COS_SCREENSHOT_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
}

func channelDiff(a, b color.RGBA) uint8 {
	d := func(x, y uint8) uint8 {
		if x > y {
			return x - y
		}
		return y - x
	}
	return max(d(a.R, b.R), max(d(a.G, b.G), max(d(a.B, b.B), d(a.A, b.A))))
}

// Check rendered pixels, not style metadata: a square border must not leak
// outside the rounded face in any corner, including hover/pressed/disabled.
func assertRoundedButton(t *testing.T, tt *ui.Tester, label string, scale int) {
	t.Helper()
	n, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("missing button %q", label)
	}
	r := n.Bounds
	x0, y0 := int(r.X*float32(scale)), int(r.Y*float32(scale))
	x1, y1 := int((r.X+r.W)*float32(scale))-1, int((r.Y+r.H)*float32(scale))-1
	img := tt.Image()
	want := img.RGBAAt(x0-4*scale, y0-4*scale)
	for _, p := range []image.Point{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}} {
		if got := img.RGBAAt(p.X, p.Y); channelDiff(got, want) > 3 {
			t.Errorf("%s has a square corner at %v: got %v, surrounding surface %v", label, p, got, want)
		}
	}
}

func assertRoundedFocusRing(t *testing.T, tt *ui.Tester, label string, scale int) {
	t.Helper()
	if !tt.Focused(label) {
		t.Fatalf("keyboard focus did not reach %s", label)
	}
	n, _ := tt.AccessByRole(ui.RoleButton, label)
	r := n.Bounds
	img := tt.Image()
	x, y := int(r.X*float32(scale)), int(r.Y*float32(scale))
	want := img.RGBAAt(x-8*scale, y-8*scale)
	if got := img.RGBAAt(x-2*scale, y-2*scale); got != want {
		t.Errorf("%s has a square focus corner: got %v, surrounding surface %v", label, got, want)
	}
	if img.RGBAAt(int((r.X+r.W/2)*float32(scale)), y-2*scale) == want {
		t.Error("keyboard focus has no visible ring")
	}
}

func TestButtonRoundedCorners(t *testing.T) {
	variants := []components.ButtonVariant{
		components.ButtonPrimary,
		components.ButtonSecondary,
		components.ButtonOutline,
		components.ButtonGhost,
		components.ButtonToolbar,
	}
	for _, dark := range []bool{false, true} {
		for _, scale := range []int{1, 2} {
			for _, variant := range variants {
				t.Run(fmt.Sprintf("dark-%t/scale-%d/variant-%d", dark, scale, variant), func(t *testing.T) {
					disabled := false
					activations := 0
					tt := ui.NewTesterWith(func(c *ui.Context) {
						c.SetTheme(workbenchTheme(c))
						ui.Column(c).Fill().Padding(20).Background(c.Theme().Background).Children(func() {
							workbenchButton(c, "", "取消", variant, disabled, func() { activations++ })
						})
					}, 160, 100, ui.TesterOptions{Dark: dark})
					tt.SetScale(float32(scale))
					assertRoundedButton(t, tt, "取消", scale)
					r, _ := tt.Find("取消")
					x, y := r.X+r.W/2, r.Y+r.H/2
					tt.Move(x, y)
					assertRoundedButton(t, tt, "取消", scale)
					tt.Press(x, y)
					assertRoundedButton(t, tt, "取消", scale)
					tt.Release(x, y)
					if activations != 1 {
						t.Fatalf("pointer click activated %d times", activations)
					}
					disabled = true
					tt.Frame()
					assertRoundedButton(t, tt, "取消", scale)
					tt.Key(0, ui.KeyEnter)
					tt.ClickAt(x, y)
					if activations != 1 {
						t.Fatal("disabled button activated")
					}
					disabled = false
					tt.Frame()
					tt.Move(4, 4)
					tt.Key(0, ui.KeyTab)
					assertRoundedFocusRing(t, tt, "取消", scale)
					if variant == components.ButtonSecondary && scale == 2 {
						saveButtonFrame(t, tt, fmt.Sprintf("button-focus-dark-%t", dark))
					}
					tt.Key(0, ui.KeyEnter)
					if activations != 2 {
						t.Fatalf("keyboard activation count: got %d, want 2", activations)
					}
				})
			}
		}
	}
}

func TestDialogButtonRoundedCorners(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, size := range [][2]int{{1000, 720}, {1440, 900}} {
			t.Run(fmt.Sprintf("%dx%d/dark-%t", size[0], size[1], dark), func(t *testing.T) {
				a := newApplication()
				// Pixel corner samples use the approved 100% geometry.
				a.zoom = 1
				tt := ui.NewTesterWith(a.view, size[0], size[1], ui.TesterOptions{Dark: dark})
				tt.SetScale(2)
				click(t, tt, "自定义测试站别")
				tt.Advance(250 * time.Millisecond)
				saveButtonFrame(t, tt, fmt.Sprintf("custom-stations-%dx%d-dark-%t", size[0], size[1], dark))
				assertRoundedButton(t, tt, "取消", 2)
				click(t, tt, "取消")
				tt.Advance(250 * time.Millisecond)
				if a.customOpen {
					t.Fatal("cancel did not dismiss the dialog")
				}
				click(t, tt, "数据目录与测试文件设置")
				tt.Advance(250 * time.Millisecond)
				assertRoundedButton(t, tt, "完成", 2)
				click(t, tt, "完成")
				tt.Advance(250 * time.Millisecond)
				if a.settingsOpen {
					t.Fatal("done did not dismiss settings")
				}
				a.dialogOpen = true
				a.dialogTitle = "检测结果"
				a.dialogText = "测试内容"
				tt.Frame()
				tt.Advance(250 * time.Millisecond)
				assertRoundedButton(t, tt, "关闭", 2)
				click(t, tt, "关闭")
				tt.Advance(250 * time.Millisecond)
				if a.dialogOpen {
					t.Fatal("close did not dismiss diagnostics")
				}
				a.config.Entries = "module-01"
				a.busy = true
				tt.Frame()
				assertRoundedButton(t, tt, "取消", 2)
			})
		}
	}
}
