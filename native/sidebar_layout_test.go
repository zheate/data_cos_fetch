package main

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSidebarHeightAllocation(t *testing.T) {
	for _, scenario := range []struct {
		width, height int
		zoom          float32
	}{{1000, 720, 1.3}, {1440, 900, 1.3}, {1440, 900, 1.7}, {1440, 842, 1}} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%dx%d-zoom-%g-dark-%t", scenario.width, scenario.height, scenario.zoom, dark), func(t *testing.T) {
				a := newApplication()
				a.zoom = scenario.zoom
				tt := ui.NewTesterWith(a.view, scenario.width, scenario.height, ui.TesterOptions{Dark: dark})
				entries, ok := tt.Find("编号或完整路径清单")
				if !ok || entries.H < 120*a.zoom {
					t.Fatalf("entry input lost its usable minimum height: %+v", entries)
				}
				if scenario.height == 842 || scenario.zoom == 1.3 && scenario.height == 900 {
					if entries.H < 240*a.zoom {
						t.Fatalf("unused station space did not enlarge the entry input: %+v", entries)
					}
				}
				list, ok := tt.Find("测试站别列表")
				if !ok || list.H <= 0 {
					t.Fatalf("station list has no usable viewport: %+v", list)
				}
				run, _ := tt.Find("开始提取")
				if run.Y+run.H > float32(scenario.height)-24 || list.Y+list.H > run.Y {
					t.Fatalf("station list displaced the run action: list=%+v run=%+v", list, run)
				}
				tt.Scroll(list.X+list.W/2, list.Y+list.H/2, 0, 2000)
				last, ok := tt.Find("高温测试")
				if !ok || last.H < 31*a.zoom || last.Y < list.Y-1 || last.Y+last.H > list.Y+list.H+1 {
					t.Fatalf("cannot reach the complete final station: last=%+v list=%+v", last, list)
				}
				if gap := run.Y - last.Y - last.H; gap > 24*a.zoom {
					t.Fatalf("unused space remains below stations: %g", gap)
				}
				click(t, tt, "高温测试")
				if a.config.stationChecked("高温测试") {
					t.Fatal("station selection stopped responding")
				}
				click(t, tt, "高温测试")
				if !a.config.stationChecked("高温测试") {
					t.Fatal("repeated station selection lost its state")
				}
				saveRedesignCapture(t, tt, fmt.Sprintf("sidebar-%dx%d-zoom-%g-dark-%t", scenario.width, scenario.height, scenario.zoom, dark))
				for _, mode := range []string{"芯片", "模块"} {
					click(t, tt, mode)
					got, _ := tt.Find("开始提取")
					if math.Abs(float64(got.Y-run.Y)) > 1 {
						t.Fatalf("mode %s moved the run action: %+v -> %+v", mode, run, got)
					}
				}
			})
		}
	}
}

func TestAddingStationsShrinksEntryInput(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprintf("dark-%t", dark), func(t *testing.T) {
			a := newApplication()
			tt := ui.NewTesterWith(a.view, 1440, 900, ui.TesterOptions{Dark: dark})
			initialEntries, _ := tt.Find("编号或完整路径清单")
			initialRun, _ := tt.Find("开始提取")
			previousHeight := initialEntries.H
			for _, count := range []int{0, 2, 6, 10, 40, 0} {
				var names []string
				for i := 0; i < count; i++ {
					names = append(names, fmt.Sprintf("新增站别-%02d", i))
				}
				a.config.CustomTests = strings.Join(names, "\n")
				tt.Frame()
				entries, _ := tt.Find("编号或完整路径清单")
				list, _ := tt.Find("测试站别列表")
				run, _ := tt.Find("开始提取")
				if entries.H < 120*a.zoom || run != initialRun {
					t.Fatalf("adding %d stations displaced essential controls: entries=%+v run=%+v", count, entries, run)
				}
				if count == 0 {
					if math.Abs(float64(entries.H-initialEntries.H)) > 1 {
						t.Fatalf("removing custom stations did not restore entry height: %+v", entries)
					}
				} else if entries.H > previousHeight+1 {
					t.Fatalf("adding stations enlarged the entry input: %g -> %g", previousHeight, entries.H)
				}
				lastName := "高温测试"
				if count > 0 {
					lastName = names[len(names)-1]
				}
				last, _ := tt.Find(lastName)
				if entries.H > 130*a.zoom && (last.H < 31*a.zoom || last.Y+last.H > list.Y+list.H+1) {
					t.Fatalf("stations scroll before the entry input reaches its minimum: entries=%+v last=%+v", entries, last)
				}
				saveRedesignCapture(t, tt, fmt.Sprintf("station-growth-%02d-dark-%t", count, dark))
				tt.Scroll(list.X+list.W/2, list.Y+list.H/2, 0, 2000)
				last, _ = tt.Find(lastName)
				if last.H < 31*a.zoom || last.Y < list.Y-1 || last.Y+last.H > list.Y+list.H+1 {
					t.Fatalf("cannot scroll to final added station: last=%+v list=%+v", last, list)
				}
				previousHeight = entries.H
			}
		})
	}
}
