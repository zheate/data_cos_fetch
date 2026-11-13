package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func click(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func replaceText(t *testing.T, tt *ui.Tester, label, value string) {
	t.Helper()
	click(t, tt, label)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(value)
}

func TestCurrentValidationAndQuickActions(t *testing.T) {
	a := newApplication()
	a.config.Entries = "module-01"
	a.worker = func(ctx context.Context, _ string, _ extractRequest) (*extractResult, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	tt := ui.NewTester(a.view, 1000, 720)
	replaceText(t, tt, "电流点 / A", "2oops")
	current, ok := tt.AccessByRole(ui.RoleTextField, "电流点 / A")
	if !ok || !current.Invalid || !tt.HasText("无法读取电流：2oops") {
		t.Fatal("invalid current has no field feedback")
	}
	if run, _ := tt.Access("开始提取"); !run.Disabled {
		t.Fatal("invalid current allows extraction")
	}
	tt.Key(ui.Cmd, ui.KeyEnter)
	if a.busy {
		t.Fatal("keyboard bypasses current validation")
	}
	replaceText(t, tt, "电流点 / A", "8~10")
	if current, _ := tt.AccessByRole(ui.RoleTextField, "电流点 / A"); current.Invalid {
		t.Fatal("valid range still marked invalid")
	}
	click(t, tt, "全部")
	if a.config.Currents != "a" {
		t.Fatal("all-current action failed")
	}
	click(t, tt, "最大值")
	if a.config.Currents != "" {
		t.Fatal("maximum-current action failed")
	}
	tt.Key(ui.Cmd, ui.KeyL)
	if !tt.Focused("编号或完整路径清单") {
		t.Fatal("number-input shortcut did not focus input")
	}
	tt.Key(ui.Cmd, ui.KeyEnter)
	if !a.busy {
		t.Fatal("valid keyboard extraction did not start")
	}
	tt.Key(ui.Cmd, ui.KeyEnter)
	if button, _ := tt.Access("开始提取"); !button.Disabled {
		t.Fatal("busy task permits another extraction")
	}
	click(t, tt, "取消")
	waitTask(t, a)
}

func TestCustomStationDraftSaveAndDismiss(t *testing.T) {
	a := newApplication()
	a.config.CustomTests = "原有站别"
	tt := ui.NewTester(a.view, 1000, 720)
	for _, dismiss := range []string{"Escape", "取消", "outside"} {
		click(t, tt, "自定义测试站别")
		replaceText(t, tt, "自定义测试站别清单", "临时站别")
		switch dismiss {
		case "Escape":
			tt.Key(0, ui.KeyEscape)
		case "取消":
			click(t, tt, "取消")
		case "outside":
			tt.ClickAt(4, 4)
		}
		if a.customOpen || a.config.CustomTests != "原有站别" {
			t.Fatalf("%s applied an unsaved draft", dismiss)
		}
	}
	click(t, tt, "自定义测试站别")
	replaceText(t, tt, "自定义测试站别清单", "新站别\n新站别\n额外站别")
	click(t, tt, "保存站别")
	if a.customOpen || a.config.CustomTests != "新站别\n额外站别" {
		t.Fatal("save did not normalize and apply station draft")
	}
	if !tt.HasText("新站别") || !tt.HasText("额外站别") || tt.HasText("原有站别") {
		t.Fatal("saved station names are not reflected in the main view")
	}
	click(t, tt, "自定义测试站别")
	replaceText(t, tt, "自定义测试站别清单", " ")
	click(t, tt, "保存站别")
	if tt.HasText("新站别") || tt.HasText("额外站别") {
		t.Fatal("removed station names remain in the main view")
	}
}

func TestCustomStationsVisibleAtDesktopSizes(t *testing.T) {
	for _, size := range [][2]int{{1000, 720}, {1440, 900}} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%dx%d-dark-%t", size[0], size[1], dark), func(t *testing.T) {
				a := newApplication()
				a.config.Entries = "module-01"
				a.config.CustomTests = "老化后测试\n可靠性验证后高低温循环复测站别"
				tt := ui.NewTester(a.view, size[0], size[1])
				tt.SetDark(dark)
				if dir := os.Getenv("DATA_COS_SCREENSHOT_DIR"); dir != "" {
					if err := os.MkdirAll(dir, 0700); err != nil {
						t.Fatal(err)
					}
					f, err := os.Create(filepath.Join(dir, fmt.Sprintf("after-custom-stations-%dx%d-dark-%t.png", size[0], size[1], dark)))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, tt.Image())
					f.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
				for _, name := range lines(a.config.CustomTests) {
					bounds, ok := tt.Find(name)
					if !ok || bounds.W <= 0 || bounds.H <= 0 || bounds.X+bounds.W > a.sidebarWidth || bounds.Y+bounds.H > float32(size[1]) {
						t.Fatalf("custom station is missing or clipped: %s %+v", name, bounds)
					}
				}
				if !tt.HasText("8 个已选") || !tt.HasText("新增站别") {
					t.Fatal("station summary or visible add action is missing")
				}
				request, err := a.config.request()
				if err != nil || len(request.TestCategories) != 8 {
					t.Fatalf("visible custom stations are not included in extraction: %+v %v", request, err)
				}
				click(t, tt, "芯片")
				if tt.HasText("老化后测试") {
					t.Fatal("module custom stations leak into chip mode")
				}
				click(t, tt, "模块")
				if !tt.HasText("老化后测试") {
					t.Fatal("mode switch lost custom stations")
				}
				var many []string
				for i := 0; i < 40; i++ {
					many = append(many, fmt.Sprintf("可靠性测试-%02d", i))
				}
				a.config.CustomTests = strings.Join(many, "\n")
				tt.Frame()
				list, ok := tt.Find("测试站别列表")
				action, _ := tt.Find("开始提取")
				if !ok || list.H <= 152 || list.Y+list.H > action.Y {
					t.Fatalf("stations must use spare height above the primary action: %+v", list)
				}
				tt.Scroll(list.X+list.W/2, list.Y+list.H/2, 0, 2000)
				last, ok := tt.Find(many[len(many)-1])
				if !ok || last.Y < list.Y || last.Y+last.H > list.Y+list.H {
					t.Fatalf("custom station list cannot reach its last item: %+v in %+v", last, list)
				}
				run, ok := tt.Access("开始提取")
				if !ok || run.Disabled || run.Bounds.H < 40 || run.Bounds.Y+run.Bounds.H > float32(size[1])-24 {
					t.Fatalf("many stations displaced the primary action: %+v", run)
				}
				entries, _ := tt.Find("编号或完整路径清单")
				if entries.H < 120 {
					t.Fatalf("many stations collapsed the entry input: %+v", entries)
				}
				a.busy = true
				tt.Frame()
				if add, _ := tt.Access("自定义测试站别"); !add.Disabled {
					t.Fatal("busy extraction permits editing custom stations")
				}
			})
		}
	}
}

func TestCustomStationCheckboxSelection(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprintf("dark-%t", dark), func(t *testing.T) {
			a := newApplication()
			a.config.Entries = "module-01"
			a.config.Tests = [6]bool{}
			a.config.CustomTests = "老化后测试"
			tt := ui.NewTester(a.view, 1000, 720)
			tt.SetDark(dark)
			station, ok := tt.AccessByRole(ui.RoleCheckBox, "老化后测试")
			common, _ := tt.AccessByRole(ui.RoleCheckBox, "耦合测试")
			if !ok || !station.Checked || station.Bounds.H != common.Bounds.H || station.Bounds.W != common.Bounds.W {
				t.Fatalf("custom station does not share the checkbox target: %+v %+v", station, common)
			}
			request, err := a.config.request()
			if err != nil || len(request.TestCategories) != 1 || request.TestCategories[0] != "老化后测试" {
				t.Fatalf("new custom station is not selected by default: %+v %v", request, err)
			}
			click(t, tt, "老化后测试")
			if a.config.customTestChecked("老化后测试") || !tt.HasText("0 个已选") {
				t.Fatal("custom checkbox did not uncheck")
			}
			if run, _ := tt.Access("开始提取"); !run.Disabled {
				t.Fatal("no selected stations still permits extraction")
			}
			tt.Key(0, ui.KeySpace)
			if !a.config.customTestChecked("老化后测试") || !tt.HasText("1 个已选") {
				t.Fatal("custom checkbox does not support keyboard selection")
			}
			click(t, tt, "全选测试站别")
			if !tt.HasText("7 个已选") {
				t.Fatal("select all omits custom stations")
			}
			click(t, tt, "清空测试站别")
			if !tt.HasText("0 个已选") || a.config.customTestChecked("老化后测试") {
				t.Fatal("clear selection omits custom stations")
			}
			click(t, tt, "自定义测试站别")
			replaceText(t, tt, "自定义测试站别清单", "老化后测试\n温循后测试")
			click(t, tt, "保存站别")
			if a.config.customTestChecked("老化后测试") || !a.config.customTestChecked("温循后测试") {
				t.Fatal("editing lost the existing selection or failed to select a new station")
			}
			a.busy = true
			tt.Frame()
			click(t, tt, "温循后测试")
			if n, _ := tt.AccessByRole(ui.RoleCheckBox, "温循后测试"); !n.Disabled || !n.Checked {
				t.Fatal("busy extraction allows custom checkbox changes")
			}
		})
	}
}

func TestStationTargetsAndEntryCleanup(t *testing.T) {
	a := newApplication()
	a.config.Entries = "编号-01\n\n编号-02\n编号-01"
	tt := ui.NewTester(a.view, 1000, 720)
	for _, name := range testCategories {
		n, ok := tt.AccessByRole(ui.RoleCheckBox, name)
		if !ok || n.Bounds.H < 32 || n.Bounds.W < 120 {
			t.Fatalf("small station target %s: %+v", name, n.Bounds)
		}
	}
	click(t, tt, "耦合测试")
	if a.config.Tests[0] {
		t.Fatal("station click did not toggle")
	}
	tt.Key(0, ui.KeySpace)
	if !a.config.Tests[0] {
		t.Fatal("station keyboard toggle did not work")
	}
	click(t, tt, "整理编号清单")
	if a.config.Entries != "编号-01\n编号-02" || !tt.Focused("编号或完整路径清单") {
		t.Fatal("entry cleanup lost order or focus")
	}
	click(t, tt, "清空测试站别")
	if run, _ := tt.Access("开始提取"); !run.Disabled {
		t.Fatal("no stations still allows extraction")
	}
	click(t, tt, "芯片")
	if run, _ := tt.Access("开始提取"); run.Disabled {
		t.Fatal("module-only validation blocked chip extraction")
	}
}

func TestStationDeletion(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprintf("dark-%t", dark), func(t *testing.T) {
			a := newApplication()
			a.config.Entries = "module-01"
			a.config.Tests = [6]bool{true, false, false, false, false, true}
			a.config.CustomTests = "老化后测试\n温循后测试"
			a.config.setCustomTestChecked("温循后测试", false)
			a.result = &extractResult{Records: []record{{Entry: "原始结果", Category: "老化后测试"}}}
			a.filter()
			tt := ui.NewTester(a.view, 1000, 720)
			tt.SetDark(dark)
			remove, ok := tt.AccessByRole(ui.RoleButton, "删除选中站别")
			add, _ := tt.Access("自定义测试站别")
			if !ok || remove.Disabled || remove.Bounds.W < 28 || remove.Bounds.X < add.Bounds.X+add.Bounds.W || remove.Description != "" {
				t.Fatalf("bulk delete is not an accessible button to the right of add: %+v", remove)
			}
			if !tt.HasText("删除") {
				t.Fatal("bulk delete does not have its Chinese text label")
			}
			a.sidebarWidth = 320
			tt.Frame()
			if remove, _ := tt.Access("删除选中站别"); remove.Bounds.X+remove.Bounds.W > a.sidebarWidth-16 {
				t.Fatal("bulk delete overflows the minimum sidebar width")
			}
			a.sidebarWidth = 360
			tt.Frame()
			for _, name := range a.config.stations() {
				station, _ := tt.AccessByRole(ui.RoleCheckBox, name)
				if station.Description != "" {
					t.Fatalf("station still has a hover hint: %s", name)
				}
				if _, ok := tt.AccessByRole(ui.RoleButton, "删除站别："+name); ok {
					t.Fatalf("station has a per-item delete button: %s", name)
				}
			}
			if err := tt.RightClick("老化后测试"); err != nil {
				t.Fatal(err)
			}
			if len(tt.Menu()) != 0 || !tt.HasText("3 个已选") {
				t.Fatal("station still has a right-click action")
			}
			click(t, tt, "删除选中站别")
			want := []string{"Pre测试", "低温储存后测试", "Post测试", "封盖测试", "温循后测试"}
			if !slices.Equal(a.config.stations(), want) || !tt.HasText("0 个已选") {
				t.Fatalf("bulk deletion did not preserve exactly the unchecked stations: %v", a.config.stations())
			}
			if !tt.Focused("编号或完整路径清单") || len(a.result.Records) != 1 || a.result.Records[0].Category != "老化后测试" {
				t.Fatal("deletion lost safe focus or changed existing results")
			}
			if remove, _ := tt.Access("删除选中站别"); !remove.Disabled {
				t.Fatal("empty selection still allows deletion")
			}
			a.deleteSelectedStations()
			if !slices.Equal(a.config.stations(), want) {
				t.Fatal("repeated deletion removed unchecked stations")
			}
			click(t, tt, "全选测试站别")
			request, err := a.config.request()
			if err != nil || !slices.Equal(request.TestCategories, want) {
				t.Fatalf("select all revived a deleted station: %+v %v", request, err)
			}
			a.busy = true
			tt.Frame()
			remove, _ = tt.Access("删除选中站别")
			click(t, tt, "删除选中站别")
			a.deleteSelectedStations()
			if !remove.Disabled || !slices.Equal(a.config.stations(), want) {
				t.Fatal("busy extraction permits bulk deletion")
			}
			a.busy = false
			tt.Frame()
			for i := 0; i < 40 && !tt.Focused("删除选中站别"); i++ {
				tt.Key(0, ui.KeyTab)
			}
			if !tt.Focused("删除选中站别") {
				t.Fatal("bulk delete cannot be reached with Tab")
			}
			tt.Key(0, ui.KeySpace)
			if !tt.HasText("暂无站别，点击「新增站别」添加") || !tt.HasText("0 个已选") {
				t.Fatal("keyboard deletion of the last stations has no empty state")
			}
			if run, _ := tt.Access("开始提取"); !run.Disabled {
				t.Fatal("deleting all stations still allows module extraction")
			}
			click(t, tt, "自定义测试站别")
			replaceText(t, tt, "自定义测试站别清单", "耦合测试\n重新添加站别")
			click(t, tt, "保存站别")
			request, err = a.config.request()
			if err != nil || len(request.TestCategories) != 2 || !slices.Contains(request.TestCategories, "耦合测试") {
				t.Fatalf("cannot re-add a deleted station: %+v %v", request, err)
			}
		})
	}
}

func TestTableSelectionSortAndPreciseCopy(t *testing.T) {
	a := newApplication()
	a.result = &extractResult{Total: 3, Records: []record{
		{Entry: "部件-01", Category: "耦合测试", Current: pointer(10), Power: pointer(5.123456)},
		{Entry: "部件-02", Category: "耦合测试", Current: pointer(2), Power: pointer(8.987654)},
		{Entry: "部件-03", Category: "Post测试", Current: pointer(3)},
	}}
	a.filter()
	tt := ui.NewTester(a.view, 1440, 900)
	click(t, tt, "部件-01")
	if err := tt.ClickWith(ui.Cmd, "部件-02"); err != nil {
		t.Fatal(err)
	}
	if a.selection.Len() != 2 {
		t.Fatalf("selected %d rows", a.selection.Len())
	}
	tt.Key(ui.Cmd, ui.KeyC)
	text := tt.Clipboard()
	if !strings.Contains(text, "部件-01") || !strings.Contains(text, "部件-02") || strings.Contains(text, "部件-03") || !strings.Contains(text, "5.123456") {
		t.Fatalf("wrong copied selection: %s", text)
	}
	header, ok := tt.AccessByRole(ui.RoleColumnHeader, "电流 / A")
	if !ok {
		t.Fatal("missing current header")
	}
	tt.ClickAt(header.Bounds.X+header.Bounds.W/2, header.Bounds.Y+header.Bounds.H/2)
	if a.rows[0].Entry != "部件-02" || a.selection.Len() != 2 {
		t.Fatal("numeric sorting lost row identity")
	}
	tt.Key(0, ui.KeyEscape)
	if a.selection.Len() != 0 {
		t.Fatal("Escape did not clear row selection")
	}
	if err := tt.RightClick("5.123"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("复制单元格"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != "5.123456" {
		t.Fatalf("cell copy lost precision: %s", got)
	}
	tt.Key(ui.Cmd, ui.KeyF)
	if !tt.Focused("搜索编号或站别") {
		t.Fatal("search shortcut did not focus search")
	}
	tt.Type("部件-03")
	if len(a.rows) != 1 || a.selection.Len() != 0 {
		t.Fatal("search left stale selection")
	}
}

func TestTableEmptyHeadersAndVirtualization(t *testing.T) {
	a := newApplication()
	tt := ui.NewTester(a.view, 1440, 900)
	if _, ok := tt.Access("测试数据提取结果"); !ok {
		t.Fatal("empty state has no real table")
	}
	headers := 0
	for _, n := range tt.AccessAll() {
		if n.Role == ui.RoleColumnHeader {
			headers++
		}
	}
	if headers != len(columnTitles) {
		t.Fatalf("empty header count %d", headers)
	}
	click(t, tt, "填写编号")
	if !tt.Focused("编号或完整路径清单") {
		t.Fatal("empty-state action did not focus entry input")
	}
	a.result = &extractResult{Total: 10000}
	for i := 0; i < 10000; i++ {
		a.result.Records = append(a.result.Records, record{Entry: fmt.Sprintf("SN-%05d", i), Current: pointer(float64(i))})
	}
	a.filter()
	tt.Frame()
	rows := 0
	for _, n := range tt.AccessAll() {
		if n.Role == ui.RoleRow {
			rows++
		}
	}
	if rows == 0 || rows > 50 {
		t.Fatalf("table builds %d rows for 10000 records", rows)
	}
}

func TestWorkbenchDarkThemeAndValidation(t *testing.T) {
	a := newApplication()
	a.config.Entries = "中文编号-01"
	a.config.Currents = "invalid"
	var dark bool
	tt := ui.NewTester(func(c *ui.Context) { a.view(c); dark = c.Theme().Dark }, 1000, 720)
	tt.SetDark(true)
	if !dark {
		t.Fatal("workbench did not follow the dark appearance")
	}
	if current, _ := tt.AccessByRole(ui.RoleTextField, "电流点 / A"); !current.Invalid {
		t.Fatal("dark theme lost current validation")
	}
	for _, name := range []string{"开始提取", "编号或完整路径清单", "电流点 / A", "高温测试"} {
		bounds, ok := tt.Find(name)
		if !ok || bounds.H <= 0 || bounds.Y+bounds.H > 720 {
			t.Fatalf("dark primary control %s is clipped", name)
		}
	}
	if dir := os.Getenv("DATA_COS_SCREENSHOT_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "after-native-dark-1000x720.png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, tt.Image())
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLongCurrentErrorKeepsPrimaryActionVisible(t *testing.T) {
	a := newApplication()
	a.config.Entries = "module-01"
	a.config.Currents = strings.Repeat("invalid-", 100)
	tt := ui.NewTester(a.view, 1000, 720)
	run, ok := tt.Access("开始提取")
	if !ok || run.Bounds.H < 40 || run.Bounds.Y+run.Bounds.H > 720 {
		t.Fatalf("long input displaced extraction action: %+v", run.Bounds)
	}
}

func TestCurrentPresetsSelectionAndKeyboard(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprintf("dark-%t", dark), func(t *testing.T) {
			a := newApplication()
			a.config.Currents = "8~12"
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(workbenchTheme(c))
				ui.Column(c).Fill().Padding(8).Children(func() { a.currentInput(c) })
			}, 360, 112)
			tt.SetDark(dark)
			checked := func(label string) bool {
				info, ok := tt.AccessByRole(ui.RoleToggleButton, label)
				if !ok {
					t.Fatalf("missing preset %s", label)
				}
				return info.Checked
			}
			if checked("最大值") || checked("全部") || a.config.Currents != "8~12" {
				t.Fatal("custom currents selected a preset")
			}
			click(t, tt, "最大值")
			click(t, tt, "最大值")
			if a.config.Currents != "" || !checked("最大值") || checked("全部") {
				t.Fatal("reactivating maximum changed the preset")
			}
			snapshot := func(name string) {
				if dir := os.Getenv("DATA_COS_SCREENSHOT_DIR"); dir != "" {
					tt.SetScale(2)
					bounds, ok := tt.Find("电流快捷值")
					if !ok {
						t.Fatal("missing preset group")
					}
					r := image.Rect(int((bounds.X-4)*2), int((bounds.Y-4)*2), int((bounds.X+bounds.W+4)*2), int((bounds.Y+bounds.H+4)*2)).Intersect(tt.Image().Bounds())
					if err := os.MkdirAll(dir, 0700); err != nil {
						t.Fatal(err)
					}
					file, err := os.Create(filepath.Join(dir, fmt.Sprintf("presets-%s-dark-%t.png", name, dark)))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(file, tt.Image().SubImage(r))
					file.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			snapshot("maximum")
			tt.Key(0, ui.KeyRight)
			if !tt.Focused("全部") || a.config.Currents != "" {
				t.Fatal("arrow navigation changed the preset before activation")
			}
			tt.Key(0, ui.KeySpace)
			if a.config.Currents != "a" || checked("最大值") || !checked("全部") {
				t.Fatal("keyboard activation failed")
			}
			snapshot("all")
			a.config.Currents = "2, 4"
			tt.Frame()
			if checked("最大值") || checked("全部") {
				t.Fatal("typing a custom current left a stale selection")
			}
			a.busy = true
			tt.Frame()
			tt.Key(0, ui.KeySpace)
			if a.config.Currents != "2, 4" {
				t.Fatal("busy preset changed the current input")
			}
			if info, _ := tt.Access("全部"); !info.Disabled {
				t.Fatal("busy preset is not disabled")
			}
		})
	}
}
