package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestBuiltInStationDeletionProtection(t *testing.T) {
	for _, name := range testCategories {
		t.Run(name, func(t *testing.T) {
			s := defaultSettings()
			s.CustomTests = name + "\n老化后测试"
			before := s
			s.deleteTest(name)
			if !reflect.DeepEqual(s, before) {
				t.Fatalf("deleting a built-in station changed settings: %+v", s)
			}
		})
	}
}

func TestLegacyDeletedBuiltInStationsRemainVisible(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	data := []byte(`{"tests":[false,true,true,true,true,false],"deletedTests":["耦合测试","高温测试"],"customTests":"老化后测试"}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := loadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	want := append(slices.Clone(testCategories), "老化后测试")
	if !slices.Equal(s.stations(), want) || s.stationChecked("耦合测试") || s.stationChecked("高温测试") {
		t.Fatalf("legacy deletion must not hide built-in stations or change selection: %+v", s)
	}
	s.selectAllTests(true)
	if !slices.Equal(s.selectedTests(), want) {
		t.Fatalf("select all omitted restored built-in stations: %v", s.selectedTests())
	}
	if err := saveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := loadSettings(path)
	if err != nil || !reflect.DeepEqual(got, s) {
		t.Fatalf("protected stations did not survive reload: %+v %v", got, err)
	}
}

func TestBuiltInOnlySelectionDisablesDeletion(t *testing.T) {
	for _, size := range [][2]int{{1000, 720}, {1440, 900}} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%dx%d-dark-%t", size[0], size[1], dark), func(t *testing.T) {
				a := newApplication()
				a.config.Entries = "module-01"
				a.config.CustomTests = "老化后测试"
				a.config.setCustomTestChecked("老化后测试", false)
				tt := ui.NewTester(a.view, size[0], size[1])
				tt.SetDark(dark)
				if dir := os.Getenv("DATA_COS_STATION_CAPTURE_DIR"); dir != "" {
					if err := os.MkdirAll(dir, 0700); err != nil {
						t.Fatal(err)
					}
					f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%dx%d-dark-%t.png", size[0], size[1], dark)))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, tt.Image())
					f.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
				remove, ok := tt.AccessByRole(ui.RoleButton, "删除选中站别")
				if !ok || !remove.Disabled {
					t.Fatal("deletion must be disabled when only built-in stations are selected")
				}
				before := a.config
				a.deleteSelectedStations()
				if !reflect.DeepEqual(a.config, before) {
					t.Fatal("direct bulk deletion changed protected stations")
				}
			})
		}
	}
}
