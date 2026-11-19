package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func themeToggleLabel(dark bool) string {
	if dark {
		return "切换到浅色模式"
	}
	return "切换到深色模式"
}

func TestWorkbenchThemeToggle(t *testing.T) {
	for _, size := range [][2]int{{1000, 720}, {1440, 842}} {
		for _, zoom := range []float32{1, 1.3} {
			for _, systemDark := range []bool{false, true} {
				t.Run(fmt.Sprintf("%dx%d-zoom-%g-dark-%t", size[0], size[1], zoom, systemDark), func(t *testing.T) {
					a := redesignFixture()
					a.zoom = zoom
					a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
					var source mygo.ThemeSource
					a.setThemeSource = func(s mygo.ThemeSource) { source = s }
					var frame ui.Theme
					tt := ui.NewTesterWith(func(c *ui.Context) {
						a.view(c)
						frame = *c.Theme()
					}, size[0], size[1], ui.TesterOptions{Dark: systemDark})
					before, result := a.config, a.result
					rootText, _ := rootSummary(a.config)
					for step := 0; step < 3; step++ {
						dark := frame.Dark
						button, ok := tt.AccessByRole(ui.RoleButton, themeToggleLabel(dark))
						if !ok || button.Disabled {
							t.Fatal("theme toggle missing or disabled")
						}
						root, ok := tt.Find("当前数据目录：" + rootText)
						if !ok || button.Bounds.X+button.Bounds.W > root.X || button.Bounds.Y != root.Y {
							t.Fatalf("theme toggle must sit directly left of the path: button=%+v root=%+v", button.Bounds, root)
						}
						if root.X+root.W > float32(size[0]) {
							t.Fatal("theme toggle displaced the path outside the window")
						}
						if step == 1 {
							// The theme remains usable while extraction owns other controls.
							a.busy = true
							tt.Frame()
						}
						click(t, tt, themeToggleLabel(dark))
						tt.Frame()
						want := "dark"
						if dark {
							want = "light"
						}
						if frame.Dark == dark || a.config.Theme != want || source != mygo.ThemeSource(want) {
							t.Fatalf("theme did not update across content, preferences and native source: frame=%t config=%s source=%s", frame.Dark, a.config.Theme, source)
						}
						loaded, err := loadSettings(a.settingsPath)
						if err != nil || loaded.Theme != want {
							t.Fatalf("theme preference not persisted: %+v %v", loaded, err)
						}
						unchanged := a.config
						unchanged.Theme = before.Theme
						if !reflect.DeepEqual(unchanged, before) || a.result != result || a.busy != (step > 0) {
							t.Fatal("theme toggle changed extraction inputs, results or task state")
						}
						if step == 0 {
							saveRedesignCapture(t, tt, fmt.Sprintf("toggle-%dx%d-zoom-%g-dark-%t", size[0], size[1], zoom, frame.Dark))
						}
					}
					// An explicit choice survives a change in the desktop appearance.
					chosen := frame.Dark
					tt.SetDark(!chosen)
					if frame.Dark != chosen {
						t.Fatal("system appearance overrode the user's saved choice")
					}
				})
			}
		}
	}
}

func TestSavedThemePreferences(t *testing.T) {
	for _, preference := range []string{"", "light", "dark", "invalid"} {
		t.Run(preference, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"theme":%q}`, preference)), 0600); err != nil {
				t.Fatal(err)
			}
			s, err := loadSettings(path)
			if err != nil {
				t.Fatal(err)
			}
			if preference == "invalid" {
				preference = ""
			}
			if s.Theme != preference {
				t.Fatalf("theme %q loaded as %q", preference, s.Theme)
			}
			for _, systemDark := range []bool{false, true} {
				a := newApplication()
				a.config = s
				var dark bool
				tt := ui.NewTesterWith(func(c *ui.Context) {
					a.view(c)
					dark = c.Theme().Dark
				}, 1000, 720, ui.TesterOptions{Dark: systemDark})
				want := preference == "dark" || preference == "" && systemDark
				if dark != want {
					t.Fatalf("saved theme did not apply on the first frame: got %t want %t", dark, want)
				}
				label := themeToggleLabel(dark)
				for i := 0; i < 30 && !tt.Focused(label); i++ {
					tt.Key(0, ui.KeyTab)
				}
				if !tt.Focused(label) {
					t.Fatal("theme toggle cannot be reached with Tab")
				}
				tt.Key(0, ui.KeySpace)
				tt.Frame()
				if dark == want {
					t.Fatal("Space did not activate the theme toggle")
				}
			}
		})
	}
}

func TestThemeToggleSecondaryViews(t *testing.T) {
	a := chartDesignFixture(false)
	var frame ui.Theme
	prefs := ui.Preferences{HighContrast: true, ReduceMotion: true}
	tt := ui.NewTesterWith(func(c *ui.Context) {
		a.view(c)
		frame = *c.Theme()
		if got := c.Preferences(); got.HighContrast != prefs.HighContrast || got.ReduceMotion != prefs.ReduceMotion {
			t.Fatal("theme toggle changed accessibility preferences")
		}
	}, 1440, 900, ui.TesterOptions{Preferences: prefs})
	for _, dark := range []bool{true, false} {
		click(t, tt, themeToggleLabel(!dark))
		tt.Frame()
		if frame.Dark != dark || frame.Focus.A != 255 {
			t.Fatal("theme toggle lost the chart theme or high contrast focus ring")
		}
		if _, ok := tt.Access("功率 (W) 数据图表"); !ok {
			t.Fatal("theme toggle lost the chart")
		}
		// Check secondary dialogs with the results table as their background.
		a.resultTab = 0
		tt.Frame()
		click(t, tt, "数据目录与测试文件设置")
		if !a.settingsOpen || frame.Dark != dark {
			t.Fatal("settings dialog lost the chosen theme")
		}
		tt.Advance(250 * time.Millisecond)
		click(t, tt, "完成")
		tt.Frame()
		tt.Advance(250 * time.Millisecond)
		if a.settingsOpen {
			t.Fatal("settings dialog did not close")
		}
		click(t, tt, "自定义测试站别")
		if !a.customOpen || frame.Dark != dark {
			t.Fatal("custom station dialog lost the chosen theme")
		}
		tt.Advance(250 * time.Millisecond)
		click(t, tt, "取消")
		tt.Frame()
		tt.Advance(250 * time.Millisecond)
		if a.settingsOpen || a.customOpen {
			t.Fatal("theme toggle prevented dialogs from closing")
		}
		a.resultTab = 1
		tt.Frame()
	}
}

func TestThemeToggleSaveFailure(t *testing.T) {
	a := newApplication()
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	a.settingsPath = filepath.Join(blocked, "settings.json")
	tt := ui.NewTester(a.view, 1000, 720)
	click(t, tt, "切换到深色模式")
	if a.config.Theme != "dark" || !a.statusError || len(a.operationErrors) != 1 || !strings.Contains(a.operationErrors[0], "保存亮暗模式失败") {
		t.Fatalf("save failure must preserve the in-memory choice and report the error: %+v", a.operationErrors)
	}
	a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
	click(t, tt, "切换到浅色模式")
	got, err := loadSettings(a.settingsPath)
	if err != nil || got.Theme != "light" {
		t.Fatalf("theme could not be saved after recovery: %+v %v", got, err)
	}
}
