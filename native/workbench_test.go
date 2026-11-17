package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components/chart"
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
	click(t, tt, "指定")
	replaceText(t, tt, "电流点 / A", "2oops")
	current, ok := tt.AccessByRole(ui.RoleTextField, "电流点 / A")
	if !ok || !current.Invalid || a.issueCount() != 1 || tt.HasText("无法读取电流：2oops") {
		t.Fatal("invalid current must mark its field and keep its detail in the issues tab")
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
	if _, visible := tt.AccessByRole(ui.RoleTextField, "电流点 / A"); visible {
		t.Fatal("all-current mode exposed raw 'a' text field")
	}
	click(t, tt, "指定")
	if a.config.Currents != "8~10" {
		t.Fatalf("switching back to custom did not restore previous range: %q", a.config.Currents)
	}
	click(t, tt, "最大值")
	if a.config.Currents != "" {
		t.Fatal("maximum-current action failed")
	}
	if _, visible := tt.AccessByRole(ui.RoleTextField, "电流点 / A"); visible {
		t.Fatal("maximum-current mode exposed text field")
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
		tt.Advance(250 * time.Millisecond)
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
				if add, ok := tt.Access("自定义测试站别"); !tt.HasText("8 个已选") || !ok || add.Disabled {
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
			if !ok || remove.Disabled || remove.Bounds.W < 28 || remove.Bounds.X < add.Bounds.X+add.Bounds.W || remove.Description != "删除选中站别" {
				t.Fatalf("bulk delete is not an accessible button to the right of add: %+v", remove)
			}
			if _, ok := tt.Access("删除选中站别"); !ok {
				t.Fatal("bulk delete does not have its Chinese accessible name")
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
			selected := func(label string) bool {
				info, ok := tt.AccessByRole(ui.RoleTab, label)
				if !ok {
					t.Fatalf("missing mode tab %s", label)
				}
				return info.Selected
			}
			if selected("最大值") || selected("全部") || !selected("指定") || a.config.Currents != "8~12" {
				t.Fatal("custom currents did not select 指定 mode")
			}
			if _, ok := tt.AccessByRole(ui.RoleTextField, "电流点 / A"); !ok {
				t.Fatal("custom mode missing current input")
			}
			click(t, tt, "最大值")
			if a.config.Currents != "" || !selected("最大值") || selected("全部") || selected("指定") {
				t.Fatal("switching to 最大值 failed")
			}
			if _, ok := tt.AccessByRole(ui.RoleTextField, "电流点 / A"); ok {
				t.Fatal("最大值 mode left current input visible")
			}
			snapshot := func(name string) {
				if dir := os.Getenv("DATA_COS_SCREENSHOT_DIR"); dir != "" {
					tt.SetScale(2)
					bounds, ok := tt.Find("电流模式")
					if !ok {
						t.Fatal("missing current mode group")
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
			if !tt.Focused("全部") || !selected("全部") || a.config.Currents != "a" {
				t.Fatalf("arrow navigation to 全部 failed: currents=%q selected=%v", a.config.Currents, selected("全部"))
			}
			if _, ok := tt.AccessByRole(ui.RoleTextField, "电流点 / A"); ok {
				t.Fatal("全部 mode exposed raw 'a' input")
			}
			snapshot("all")
			tt.Key(0, ui.KeyRight)
			if !selected("指定") || a.config.Currents != "8~12" {
				t.Fatalf("arrow navigation to 指定 did not restore custom currents: %q", a.config.Currents)
			}
			a.config.Currents = "2, 4"
			tt.Frame()
			if selected("最大值") || selected("全部") || !selected("指定") {
				t.Fatal("typing a custom current changed mode selection")
			}
			a.busy = true
			tt.Frame()
			if info, _ := tt.Access("全部"); !info.Disabled {
				t.Fatal("busy mode tab is not disabled")
			}
		})
	}
}

func TestTableEntryGroupingAndTrimmedDecimals(t *testing.T) {
	a := newApplication()
	a.result = &extractResult{Total: 3, Records: []record{
		{Entry: "DEV-01", Category: "耦合测试", Current: pointer(1.0), Power: pointer(3.5), Voltage: pointer(1.8), Lambda: pointer(964.8)},
		{Entry: "DEV-01", Category: "耦合测试", Current: pointer(2.0), Power: pointer(5.1234), Voltage: pointer(2.0), Lambda: pointer(965.2)},
		{Entry: "DEV-02", Category: "耦合测试", Current: pointer(1.0), Power: pointer(3.6), Voltage: pointer(1.82), Lambda: pointer(965.0)},
	}}
	a.filter()
	tt := ui.NewTester(a.view, 1440, 900)
	if !tt.HasText("3.5") || tt.HasText("3.500") || !tt.HasText("5.123") || tt.HasText("1.000") {
		t.Fatal("table decimal formatting did not trim trailing zeros while preserving 3-decimal precision")
	}
	countEntryLabels := func() int {
		count := 0
		for _, n := range tt.AccessAll() {
			if n.Name == "DEV-01" && n.Bounds.X > a.sidebarWidth {
				count++
			}
		}
		return count
	}
	if got := countEntryLabels(); got != 1 {
		t.Fatalf("expected grouped DEV-01 label once in default order, got %d", got)
	}
	// Right-clicking the continuation row's cell must still copy the full entry ID.
	header, _ := tt.AccessByRole(ui.RoleColumnHeader, "编号")
	var secondRow ui.Rect
	rowsSeen := 0
	for _, n := range tt.AccessAll() {
		if n.Role == ui.RoleRow && n.Bounds.Y > header.Bounds.Y+header.Bounds.H {
			if rowsSeen == 1 {
				secondRow = n.Bounds
				break
			}
			rowsSeen++
		}
	}
	tt.RightClickAt(header.Bounds.X+header.Bounds.W/2, secondRow.Y+secondRow.H/2)
	if err := tt.ChooseMenuItem("复制单元格"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != "DEV-01" {
		t.Fatalf("grouped continuation cell lost full entry copy: %q", got)
	}
	// Sorting by a numeric column must show entry IDs on every row again.
	curHeader, _ := tt.AccessByRole(ui.RoleColumnHeader, "电流 / A")
	tt.ClickAt(curHeader.Bounds.X+curHeader.Bounds.W/2, curHeader.Bounds.Y+curHeader.Bounds.H/2)
	if got := countEntryLabels(); got != 2 {
		t.Fatalf("expected DEV-01 shown on both rows when sorted by current, got %d", got)
	}
}

func TestTableStationDisplayOrder(t *testing.T) {
	a := newApplication()
	a.config.CustomTests = "老化后测试\n温循后测试"
	a.result = &extractResult{Total: 8, Records: []record{
		{Entry: "DEV-01", Category: "温循后测试", Current: pointer(2)},
		{Entry: "DEV-01", Category: "高温测试", Current: pointer(2)},
		{Entry: "DEV-01", Category: "老化后测试", Current: pointer(2)},
		{Entry: "DEV-01", Category: "封盖测试", Current: pointer(2)},
		{Entry: "DEV-01", Category: "Post测试", Current: pointer(2)},
		{Entry: "DEV-01", Category: "低温储存后测试", Current: pointer(2)},
		{Entry: "DEV-01", Category: "Pre测试", Current: pointer(2)},
		{Entry: "DEV-01", Category: "耦合测试", Current: pointer(2)},
	}}
	a.filter()
	want := []string{"耦合测试", "Pre测试", "低温储存后测试", "Post测试", "封盖测试", "高温测试", "老化后测试", "温循后测试"}
	got := make([]string, len(a.rows))
	for i, r := range a.rows {
		got[i] = r.Category
	}
	if !slices.Equal(got, want) {
		t.Fatalf("default station order mismatch: got %v, want %v", got, want)
	}
	// Clicking 测试站别 header sorts by configured station order, not UTF-8 alphabetical order.
	tt := ui.NewTester(a.view, 1440, 900)
	stationHeader, ok := tt.AccessByRole(ui.RoleColumnHeader, "测试站别")
	if !ok {
		t.Fatal("missing 测试站别 header")
	}
	tt.ClickAt(stationHeader.Bounds.X+stationHeader.Bounds.W/2, stationHeader.Bounds.Y+stationHeader.Bounds.H/2)
	for i, r := range a.rows {
		got[i] = r.Category
	}
	if !slices.Equal(got, want) {
		t.Fatalf("ascending station sort mismatch: got %v, want %v", got, want)
	}
	tt.ClickAt(stationHeader.Bounds.X+stationHeader.Bounds.W/2, stationHeader.Bounds.Y+stationHeader.Bounds.H/2)
	slices.Reverse(want)
	for i, r := range a.rows {
		got[i] = r.Category
	}
	if !slices.Equal(got, want) {
		t.Fatalf("descending station sort mismatch: got %v, want %v", got, want)
	}
}

func TestChartViewModesMetricsAndSelection(t *testing.T) {
	save := func(t *testing.T, tt *ui.Tester, name string) {
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

	sampleRecords := []record{
		{Entry: "FCP-01", Category: "耦合测试", Current: pointer(2), Power: pointer(1.42), Voltage: pointer(1.85), Efficiency: pointer(38.4), Lambda: pointer(975.2)},
		{Entry: "FCP-01", Category: "耦合测试", Current: pointer(4), Power: pointer(3.18), Voltage: pointer(2.02), Efficiency: pointer(39.4), Lambda: pointer(976.1)},
		{Entry: "FCP-01", Category: "耦合测试", Current: pointer(8), Power: pointer(6.45), Voltage: pointer(2.24), Efficiency: pointer(36.0), Lambda: pointer(977.8)},
		{Entry: "FCP-01", Category: "耦合测试", Current: pointer(10), Power: pointer(7.92), Voltage: pointer(2.35), Efficiency: pointer(33.7), Lambda: pointer(978.9)},
		{Entry: "FCP-02", Category: "耦合测试", Current: pointer(2), Power: pointer(1.50), Voltage: pointer(1.83), Efficiency: pointer(41.0), Lambda: pointer(974.8)},
		{Entry: "FCP-02", Category: "耦合测试", Current: pointer(4), Power: pointer(3.31), Voltage: pointer(2.00), Efficiency: pointer(41.4), Lambda: pointer(975.7)},
		{Entry: "FCP-02", Category: "耦合测试", Current: pointer(8), Power: pointer(6.78), Voltage: pointer(2.21), Efficiency: pointer(38.3), Lambda: pointer(977.3)},
		{Entry: "FCP-02", Category: "耦合测试", Current: pointer(10), Power: pointer(8.24), Voltage: pointer(2.31), Efficiency: pointer(35.7), Lambda: pointer(978.4)},
		{Entry: "FCP-01", Category: "老化后测试", Current: pointer(2), Power: pointer(1.38), Voltage: pointer(1.86), Efficiency: pointer(37.1), Lambda: pointer(975.5)},
		{Entry: "FCP-01", Category: "老化后测试", Current: pointer(4), Power: pointer(3.10), Voltage: pointer(2.04), Efficiency: pointer(38.0), Lambda: pointer(976.4)},
		{Entry: "FCP-01", Category: "老化后测试", Current: pointer(8), Power: pointer(6.28), Voltage: pointer(2.27), Efficiency: pointer(34.6), Lambda: pointer(978.1)},
		{Entry: "FCP-01", Category: "老化后测试", Current: pointer(10), Power: pointer(7.70), Voltage: pointer(2.38), Efficiency: pointer(32.4), Lambda: pointer(979.2)},
	}

	for _, size := range [][2]int{{1440, 900}, {1000, 720}} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%dx%d-dark-%t", size[0], size[1], dark), func(t *testing.T) {
				a := newApplication()
				a.config.Entries = "FCP-01\nFCP-02"
				a.result = &extractResult{Total: len(sampleRecords), Records: sampleRecords}
				a.filter()
				a.resetChartDefaults()
				if a.chartGroup != 0 || a.chartKind != 0 {
					t.Fatalf("multi-current sweep default mismatch: group=%d kind=%d", a.chartGroup, a.chartKind)
				}

				tt := ui.NewTesterWith(a.view, size[0], size[1], ui.TesterOptions{Dark: dark})
				click(t, tt, "数据图表")
				tt.Advance(400 * time.Millisecond)

				if _, ok := tt.Access("功率 (W) 数据图表"); !ok {
					t.Fatal("missing default power chart")
				}
				if !tt.HasText("FCP-01 · 耦合测试") || !tt.HasText("FCP-02 · 耦合测试") {
					t.Fatal("missing series legends on line chart")
				}
				for _, label := range []string{"分析指标", "对比基准", "按电流曲线", "按编号对比", "折线", "面积", "柱状"} {
					r, ok := tt.Find(label)
					if !ok || r.X+r.W > float32(size[0]) || r.Y+r.H > float32(size[1]) {
						t.Fatalf("chart control %s missing or clipped at %dx%d: %+v", label, size[0], size[1], r)
					}
				}
				if !tt.HasText("最高 / 最优") || (!tt.HasText("FCP-02 · 耦合测试 · 10 A") || !tt.HasText("8.24 W")) || !tt.HasText("均值") || !tt.HasText("极差") {
					t.Fatal("missing curve summary strip stats for power chart")
				}
				save(t, tt, fmt.Sprintf("after-chart-line-%dx%d-dark-%t", size[0], size[1], dark))

				// Switch metric via Select popup: wavelength must use adaptive tight Y scale (not 0..1000 nm)
				click(t, tt, "分析指标")
				tt.Advance(200 * time.Millisecond)
				click(t, tt, "波长 (nm)")
				tt.Advance(400 * time.Millisecond)
				if a.chartMetric != 3 {
					t.Fatalf("metric did not switch to lambda: got %d", a.chartMetric)
				}
				if _, ok := tt.Access("波长 (nm) 数据图表"); !ok {
					t.Fatal("chart did not update label to lambda")
				}
				if chartHasYTick(tt, "0") || chartHasYTick(tt, "500") {
					t.Fatal("wavelength tight-scale chart still rendered zero baseline")
				}
				if !tt.HasText("976") && !tt.HasText("978") {
					t.Fatal("wavelength tight-scale chart missing nanometer detail ticks")
				}

				// Switch to baseline station delta comparison (老化后测试 - 耦合测试)
				click(t, tt, "对比基准")
				tt.Advance(200 * time.Millisecond)
				click(t, tt, "较 耦合测试 (Δ)")
				tt.Advance(400 * time.Millisecond)
				if a.chartBaseStation != "耦合测试" {
					t.Fatalf("baseline station did not update: got %q", a.chartBaseStation)
				}
				deltaPayload := a.buildChartPayload(dark)
				if len(deltaPayload.Data.Series) != 1 || deltaPayload.Data.Series[0].Label != "FCP-01 · 老化后测试" {
					t.Fatalf("unexpected delta series: %+v", deltaPayload.Data.Series)
				}
				if v := deltaPayload.Data.Series[0].Values[0]; v < 0.29 || v > 0.31 {
					t.Fatalf("unexpected lambda delta at 2A: got %v, want ~+0.3", v)
				}
				if meta := deltaPayload.Points[deltaPayload.Data.Series[0].ID][0]; meta.Baseline == nil || meta.Record == nil {
					t.Fatal("delta point missing baseline metadata for tooltip")
				}
				if !tt.HasText("平均变化") || !tt.HasText("最大变化") || !tt.HasText("基准 0") {
					t.Fatal("missing delta summary strip or zero baseline label")
				}
				click(t, tt, "对比基准")
				tt.Advance(200 * time.Millisecond)
				click(t, tt, "原始数值")
				tt.Advance(200 * time.Millisecond)
				if a.chartBaseStation != "" {
					t.Fatalf("baseline station did not reset: got %q", a.chartBaseStation)
				}

				// Switch to entry comparison bar chart (single-current focus + mean rule + tight scale)
				click(t, tt, "按编号对比")
				click(t, tt, "柱状")
				tt.Advance(400 * time.Millisecond)
				if a.chartGroup != 1 || a.chartKind != 2 {
					t.Fatalf("chart mode switch failed: group=%d kind=%d", a.chartGroup, a.chartKind)
				}
				if !tt.HasText("对比: 10 A") || !tt.HasText("按值排序") || !tt.HasText("均值 978.833 nm") {
					t.Fatal("entry comparison bar mode missing single-current control, sort toggle, or mean rule")
				}
				if chartHasYTick(tt, "0") || chartHasYTick(tt, "500") {
					t.Fatal("wavelength tight-scale bar chart still rendered zero baseline")
				}
				save(t, tt, fmt.Sprintf("after-chart-bar-%dx%d-dark-%t", size[0], size[1], dark))

				// Switch comparison current to 2 A
				click(t, tt, "对比电流")
				tt.Advance(200 * time.Millisecond)
				click(t, tt, "2 A")
				tt.Advance(200 * time.Millisecond)
				if a.chartCompareCurrent != "cur:2" || !tt.HasText("对比: 2 A") {
					t.Fatalf("failed switching compare current to 2 A: key=%q", a.chartCompareCurrent)
				}

				// Selecting a single row focuses matching series/device in the chart
				a.selection.Clear()
				a.selection.Add(a.rows[0].identity())
				click(t, tt, "按电流曲线")
				tt.Advance(400 * time.Millisecond)
				if !tt.HasText("已聚焦选中 (1 组)") {
					t.Fatal("chart did not reflect focused table selection in curve mode")
				}
				click(t, tt, "按编号对比")
				tt.Advance(200 * time.Millisecond)
				data, _ := a.buildChartData()
				if len(data.Labels) != 1 || data.Labels[0] != "FCP-01" {
					t.Fatalf("selection did not isolate FCP-01 in entry comparison: %+v", data)
				}

				// Series count above maxChartSeries should count distinct series without double-counting skipped rows
				many := make([]record, 0, 20)
				for e := 1; e <= 2; e++ {
					for i := 1; i <= 10; i++ {
						many = append(many, record{
							Entry:    fmt.Sprintf("DEV-%02d", e),
							Category: "耦合测试",
							Current:  pointer(float64(i)),
							Power:    pointer(float64(i) * 1.5),
						})
					}
				}
				overflow := buildEntryComparisonData(many, chartMetrics[0], chartSummary{}, "", nil, dark)
				if overflow.Summary.ShownSeries != maxChartSeries || overflow.Summary.TotalSeries != 10 {
					t.Fatalf("overflow series count mismatch: got shown=%d total=%d, want shown=%d total=10", overflow.Summary.ShownSeries, overflow.Summary.TotalSeries, maxChartSeries)
				}
				seenColors := make(map[ui.Color]struct{}, maxChartSeries)
				for _, ser := range overflow.Data.Series {
					seenColors[ser.Color] = struct{}{}
				}
				if len(seenColors) != maxChartSeries {
					t.Fatalf("expected %d distinct series colors, got %d", maxChartSeries, len(seenColors))
				}

				// Switching back to a missing metric renders domain empty state instead of broken axes
				a.selection.Clear()
				a.chartMetric = 6 // Cold (not populated in sampleRecords)
				tt.Frame()
				if !tt.HasText("当前结果不含「冷波长 (nm)」数据") {
					t.Fatal("missing empty-metric message on chart")
				}
			})
		}
	}
}

func TestChartInsightsRankingAndLegendSolo(t *testing.T) {
	rows := []record{
		{Entry: "DEV-A", Category: "耦合测试", Current: pointer(2), Power: pointer(1.2), Voltage: pointer(1.80), Shift: pointer(-0.5)},
		{Entry: "DEV-A", Category: "耦合测试", Current: pointer(10), Power: pointer(6.0), Voltage: pointer(2.10), Shift: pointer(-0.5)},
		{Entry: "DEV-B", Category: "耦合测试", Current: pointer(2), Power: pointer(1.6), Voltage: pointer(1.92), Shift: pointer(0.2)},
		{Entry: "DEV-B", Category: "耦合测试", Current: pointer(10), Power: pointer(8.0), Voltage: pointer(2.28), Shift: pointer(0.2)},
		{Entry: "DEV-C", Category: "耦合测试", Current: pointer(2), Power: pointer(1.4), Voltage: pointer(1.86), Shift: pointer(0.8)},
		{Entry: "DEV-C", Category: "耦合测试", Current: pointer(10), Power: pointer(7.0), Voltage: pointer(2.20), Shift: pointer(0.8)},
	}

	// 1. Curve mode uses proportional numeric X-axis when all points are numeric; falls back otherwise
	curve := buildCurrentCurveData(rows, chartMetrics[0], chartSummary{}, "", nil, false)
	if !curve.HasLinearX || len(curve.XValues) != 2 || curve.XValues[0] != 2 || curve.XValues[1] != 10 {
		t.Fatalf("expected HasLinearX=true with [2, 10], got HasLinearX=%v XValues=%v", curve.HasLinearX, curve.XValues)
	}
	mixed := append(slices.Clone(rows), record{Entry: "DEV-D", Category: "耦合测试", Current: nil, Power: pointer(7.5)})
	mixedCurve := buildCurrentCurveData(mixed, chartMetrics[0], chartSummary{}, "", nil, false)
	if mixedCurve.HasLinearX {
		t.Fatalf("expected HasLinearX=false when non-numeric current point is present")
	}

	// 2. Entry comparison defaults to largest current and sorts by metric direction
	active, opts := resolveCompareOption(rows, "")
	if active.Key != "cur:10" || active.Label != "10 A" {
		t.Fatalf("expected default compare option cur:10, got %+v", active)
	}
	rankedPower := buildEntryComparisonView(rows, chartMetrics[0], chartSummary{CompareCurrent: active.Label}, "", nil, false, active, true)
	if !slices.Equal(rankedPower.Data.Labels, []string{"DEV-B", "DEV-C", "DEV-A"}) {
		t.Fatalf("higher-best power ranking mismatch: got %v", rankedPower.Data.Labels)
	}
	rankedVolt := buildEntryComparisonView(rows, chartMetrics[1], chartSummary{CompareCurrent: active.Label}, "", nil, false, active, true)
	if !slices.Equal(rankedVolt.Data.Labels, []string{"DEV-A", "DEV-C", "DEV-B"}) {
		t.Fatalf("lower-best voltage ranking mismatch: got %v", rankedVolt.Data.Labels)
	}
	unranked := buildEntryComparisonView(rows, chartMetrics[0], chartSummary{CompareCurrent: active.Label}, "", nil, false, active, false)
	if !slices.Equal(unranked.Data.Labels, []string{"DEV-A", "DEV-B", "DEV-C"}) {
		t.Fatalf("unsorted order mismatch: got %v", unranked.Data.Labels)
	}
	allOpt := opts[len(opts)-1]
	if !allOpt.IsAll {
		t.Fatalf("expected trailing compare option to be IsAll, got %+v", allOpt)
	}
	allView := buildEntryComparisonView(rows, chartMetrics[0], chartSummary{CompareCurrent: allOpt.Label}, "", nil, false, allOpt, true)
	if len(allView.Data.Series) != 2 || len(allView.Data.Labels) != 3 {
		t.Fatalf("expected multi-current grouped view with 2 series and 3 labels, got %d series / %d labels", len(allView.Data.Series), len(allView.Data.Labels))
	}

	// 3. computeChartInsights handles dirZeroBest (negative vs positive smallest magnitude) and hidden filtering
	shiftView := buildEntryComparisonView(rows, chartMetrics[4], chartSummary{CompareCurrent: active.Label}, "", nil, false, active, true)
	ins := computeChartInsights(shiftView, chart.State{}, chartMetrics[4], false)
	if ins.BestAbsVal != 0.2 || !strings.Contains(ins.BestAbsOwner, "DEV-B") {
		t.Fatalf("expected DEV-B (+0.2) to be cleanest shift, got val=%v owner=%q", ins.BestAbsVal, ins.BestAbsOwner)
	}
	if ins.WorstAbsVal != 0.8 || !strings.Contains(ins.WorstAbsOwner, "DEV-C") {
		t.Fatalf("expected DEV-C (+0.8) to be worst shift, got val=%v owner=%q", ins.WorstAbsVal, ins.WorstAbsOwner)
	}
	// Check signed-negative BestAbsVal when negative delta has smaller |v| than positive delta
	negRows := []record{
		{Entry: "DEV-N", Category: "耦合测试", Current: pointer(10), Shift: pointer(-0.5)},
		{Entry: "DEV-P", Category: "耦合测试", Current: pointer(10), Shift: pointer(0.3)},
	}
	negIns := computeChartInsights(buildEntryComparisonView(negRows, chartMetrics[4], chartSummary{}, "", nil, false, active, true), chart.State{}, chartMetrics[4], false)
	if negIns.BestAbsVal != 0.3 || negIns.WorstAbsVal != -0.5 {
		t.Fatalf("negative shift magnitude comparison bug: got best=%v worst=%v", negIns.BestAbsVal, negIns.WorstAbsVal)
	}

	// Dynamic insight recomputation when hiding top series in curve mode
	curveInsAll := computeChartInsights(curve, chart.State{}, chartMetrics[0], true)
	if math.Abs(curveInsAll.MaxVal-8.0) > 1e-6 || curveInsAll.MaxOwner != "DEV-B · 10 A" {
		t.Fatalf("unexpected curve max insight: %+v", curveInsAll)
	}
	state := chart.State{Hidden: map[string]bool{}}
	toggleSoloSeries(&state, curve.Data.Series, curve.Data.Series[0].ID) // Solo DEV-A (max 6.0 W)
	if !state.Hidden[curve.Data.Series[1].ID] || !state.Hidden[curve.Data.Series[2].ID] || state.Hidden[curve.Data.Series[0].ID] {
		t.Fatalf("toggleSoloSeries did not solo target series: %+v", state.Hidden)
	}
	curveInsSolo := computeChartInsights(curve, state, chartMetrics[0], true)
	if math.Abs(curveInsSolo.MaxVal-6.0) > 1e-6 || !strings.Contains(curveInsSolo.MaxOwner, "10 A") {
		t.Fatalf("insights did not recompute for soloed series: %+v", curveInsSolo)
	}
	toggleSoloSeries(&state, curve.Data.Series, curve.Data.Series[0].ID) // Un-solo restores all
	if len(state.Hidden) != 0 {
		t.Fatalf("second solo toggle did not restore all series: %+v", state.Hidden)
	}

	// 4. siblingContextText includes percentage change against baseline
	baseRec := record{Entry: "DEV-A", Category: "耦合测试", Power: pointer(5.0)}
	curRec := record{Entry: "DEV-A", Category: "老化后", Power: pointer(4.5)}
	ctxText := siblingContextText(&curRec, chartMetrics[0], &baseRec)
	if !strings.Contains(ctxText, "现值 4.5 W") || !strings.Contains(ctxText, "基准 5 W") || !strings.Contains(ctxText, "-10 %") {
		t.Fatalf("unexpected delta context text: %q", ctxText)
	}

	// 5. UI legend solo reset ("全部显示") and dynamic summary strip update
	a := newApplication()
	a.result = &extractResult{Total: len(rows), Records: slices.Clone(rows)}
	a.filter()
	a.resetChartDefaults()
	tt := ui.NewTesterWith(a.view, 1200, 800, ui.TesterOptions{})
	click(t, tt, "数据图表")
	tt.Advance(200 * time.Millisecond)
	toggleSoloSeries(&a.chartState, curve.Data.Series, curve.Data.Series[0].ID)
	tt.Frame()
	if !tt.HasText("全部显示") || (!tt.HasText("10 A") || !tt.HasText("6 W")) || tt.HasText("DEV-B · 10 A") {
		t.Fatalf("solo UI mismatch: reset=%v max6=%v max8=%v hidden=%+v", tt.HasText("全部显示"), tt.HasText("10 A 6 W"), tt.HasText("DEV-B · 10 A"), a.chartState.Hidden)
	}
	click(t, tt, "全部显示")
	tt.Advance(200 * time.Millisecond)
	if len(a.chartState.Hidden) != 0 || !tt.HasText("DEV-B · 10 A") {
		t.Fatalf("clicking 全部显示 did not restore all series and summary: hidden=%+v", a.chartState.Hidden)
	}
}

func TestChartTightSpreadZoomAndMeanDelta(t *testing.T) {
	save := func(tt *ui.Tester, name string) {
		if dir := os.Getenv("DATA_COS_TIGHT_SCREENSHOT_DIR"); dir != "" {
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

	currents := []float64{1, 2, 4, 8}
	basePowers := []float64{2.10, 4.30, 8.60, 16.49}
	spanPowers := []float64{0.015, 0.030, 0.055, 0.080}
	rows := make([]record, 0, 100*len(currents))
	var entries []string
	for i := 0; i < 100; i++ {
		entry := fmt.Sprintf("DEV-%03d", i+1)
		entries = append(entries, entry)
		norm := (float64(i) - 49.5) / 49.5 // [-1, +1], cohort mean untouched
		for ci, cur := range currents {
			p := basePowers[ci] + spanPowers[ci]*norm
			// Push DEV-100 up and pull DEV-001/DEV-015 down at 8 A so global 8 A spread is 0.18 W (~1.1%) around 16.49 W.
			if ci == 3 {
				if i == 99 {
					p = 16.58
				} else if i == 0 {
					p = 16.40
				} else if i == 2 {
					p = 16.405
				}
			}
			v := 1.85 + 0.08*float64(ci) + 0.008*norm
			eff := 44.5 - 0.6*float64(ci) + 0.25*norm
			lam := 976.0 + 0.8*float64(ci) + 0.05*norm
			rows = append(rows, record{
				Entry:      entry,
				Category:   "老化测试",
				Current:    pointer(cur),
				Power:      pointer(p),
				Voltage:    pointer(v),
				Efficiency: pointer(eff),
				Lambda:     pointer(lam),
			})
		}
	}

	a := newApplication()
	a.config.Entries = strings.Join(entries, "\n")
	a.result = &extractResult{Total: len(rows), Records: rows}
	a.filter()
	a.resetChartDefaults()

	tt := ui.NewTesterWith(a.view, 1440, 900, ui.TesterOptions{Dark: false})
	click(t, tt, "数据图表")
	tt.Advance(400 * time.Millisecond)

	// Single-station cohort should still expose "对比基准" with "较 全量均值 (Δ)".
	click(t, tt, "对比基准")
	tt.Advance(200 * time.Millisecond)
	click(t, tt, "较 全量均值 (Δ)")
	tt.Advance(350 * time.Millisecond)

	if a.chartBaseStation != baseMeanSentinel {
		t.Fatalf("expected mean baseline sentinel, got %q", a.chartBaseStation)
	}
	if !tt.HasText("基准 0") || !tt.HasText("平均变化") || !tt.HasText("最大变化") {
		t.Fatal("mean-delta mode missing zero reference line or delta summary stats")
	}
	if _, has15 := tt.Find("15"); has15 {
		t.Fatal("mean-delta curve view did not strip current ramp scale")
	}
	if _, has10 := tt.Find("10"); has10 {
		t.Fatal("mean-delta curve view did not strip current ramp scale")
	}

	payload := a.buildChartPayload(false)
	var metas []chartPointMeta
	for _, s := range payload.Data.Series {
		if strings.HasPrefix(s.Label, "DEV-100") {
			metas = payload.Points[s.ID]
			break
		}
	}
	if len(metas) != 4 || metas[3].Baseline == nil {
		t.Fatalf("expected 4 synthetic mean-baseline points for DEV-100, got %d", len(metas))
	}
	tipCtx := siblingContextText(metas[3].Record, chartMetrics[0], metas[3].Baseline)
	if !strings.Contains(tipCtx, "现值 16.58 W") || !strings.Contains(tipCtx, "基准 16.49 W") || !strings.Contains(tipCtx, "+0.546 %") {
		t.Fatalf("unexpected mean-delta tooltip context: %q", tipCtx)
	}
	save(tt, "tight-1-mean-delta-curves")

	// Return to raw values and switch to entry comparison bars: narrow 1.1% spread should auto-enable "放大差异".
	click(t, tt, "对比基准")
	tt.Advance(200 * time.Millisecond)
	click(t, tt, "原始数值")
	tt.Advance(250 * time.Millisecond)
	click(t, tt, "按编号对比")
	tt.Advance(250 * time.Millisecond)
	click(t, tt, "柱状")
	tt.Advance(350 * time.Millisecond)
	if !a.chartSortByValue {
		click(t, tt, "按值排序")
		tt.Advance(250 * time.Millisecond)
	}

	if !tt.HasText("放大差异") {
		t.Fatal("missing 放大差异 toggle button in comparison mode")
	}
	if !tt.HasText("(1.1 %)") {
		t.Fatalf("missing relative spread percentage next to range in insights strip")
	}
	if _, hasZero := tt.Find("0"); hasZero {
		t.Fatal("tight 1.1% spread bar chart should auto-zoom away from 0")
	}
	if !tt.HasText("全量均值") {
		t.Fatal("missing global mean reference line on zoomed comparison bars")
	}

	// User toggle off must immediately restore 0 baseline.
	click(t, tt, "放大差异")
	tt.Advance(300 * time.Millisecond)
	if _, hasZero := tt.Find("0"); !hasZero {
		t.Fatal("toggling 放大差异 off did not restore 0 baseline")
	}

	// Toggle back on and switch window to 两头各 12 for best vs worst diverging view.
	click(t, tt, "放大差异")
	tt.Advance(250 * time.Millisecond)
	click(t, tt, "对比范围")
	tt.Advance(200 * time.Millisecond)
	click(t, tt, "两头各 12")
	tt.Advance(350 * time.Millisecond)

	_, hasZero := tt.Find("0")
	if !tt.HasText("两头 24 / 100 编号") || !tt.HasText("全量均值") || hasZero {
		t.Fatal("两头各 12 tight-spread zoomed bar view missing expected labels or still anchored at 0")
	}
	save(tt, "tight-2-auto-zoom-bars")
}

func TestChartHundredEntriesScaleAndExtremes(t *testing.T) {
	currents := []float64{1, 2, 4, 8}
	rows := make([]record, 0, 400)
	var entryList []string
	for n := 1; n <= 100; n++ {
		entry := fmt.Sprintf("DEV-%03d", n)
		entryList = append(entryList, entry)
		// Realistic production variation, plus hidden champion #65 and worst outlier #88/#42.
		quality := math.Sin(float64(n)*0.37)*1.1 + math.Cos(float64(n)*0.19)*0.5
		switch n {
		case 65:
			quality = 3.2
		case 42:
			quality = -3.4
		case 88:
			quality = -4.6
		}
		for _, cur := range currents {
			baseP := 1.85 * cur
			if cur == 1 {
				baseP = 1.9
			}
			p := math.Round((baseP+quality*(cur/8.0)*1.15)*100) / 100
			v := math.Round((1.55+0.12*cur-quality*0.025)*1000) / 1000
			eff := math.Round((56.0-math.Pow(cur-4.0, 2)*0.45+quality*2.6)*10) / 10
			lam := math.Round((969.6+0.32*cur-quality*0.55)*100) / 100
			rows = append(rows, record{
				Entry:      entry,
				Category:   "耦合测试",
				Current:    pointer(cur),
				Power:      pointer(p),
				Voltage:    pointer(v),
				Efficiency: pointer(eff),
				Lambda:     pointer(lam),
			})
		}
	}

	a := newApplication()
	a.config.Entries = strings.Join(entryList, "\n")
	a.config.Currents = "1, 2, 4, 8"
	a.config.Tests = [6]bool{true, false, false, false, false, false}
	a.result = &extractResult{Total: len(rows), Records: rows}
	a.filter()
	a.resetChartDefaults()

	save := func(tt *ui.Tester, name string) {
		t.Helper()
		if dir := os.Getenv("DATA_COS_HUNDRED_SCREENSHOT_DIR"); dir != "" {
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

	tt := ui.NewTesterWith(a.view, 1440, 900, ui.TesterOptions{Dark: false})
	click(t, tt, "数据图表")
	tt.Advance(300 * time.Millisecond)

	// 1. Curve mode: samples 4 best + 4 worst out of 100, while GlobalInsights reflects all 100 (champion DEV-065 + worst DEV-088).
	curve := a.buildChartPayload(false)
	if curve.Summary.TotalSeries != 100 || curve.Summary.ShownSeries != maxChartSeries || !curve.Summary.SampledExtremes {
		t.Fatalf("expected extreme sampling 8/100 in curve mode, got %+v", curve.Summary)
	}
	var seen65, seen88 bool
	for _, s := range curve.Data.Series {
		if strings.HasPrefix(s.Label, "DEV-065") {
			seen65 = true
		}
		if strings.HasPrefix(s.Label, "DEV-088") {
			seen88 = true
		}
	}
	if !seen65 || !seen88 {
		t.Fatalf("extreme sampling missed champion DEV-065 (%v) or worst DEV-088 (%v): %+v", seen65, seen88, curve.Data.Series)
	}
	if !tt.HasText("极值代表 8 / 100 组") || !tt.HasText("DEV-065") || !tt.HasText("DEV-088") {
		t.Fatalf("missing 100-device curve sampling badge or extreme owners")
	}
	save(tt, "hundred-1-curve-extremes")

	// 2. Entry comparison mode @ 8 A: default Top 24 of 100, global mean line, and window switchers.
	click(t, tt, "按编号对比")
	click(t, tt, "柱状")
	tt.Advance(300 * time.Millisecond)
	top := a.buildChartPayload(false)
	if top.Summary.TotalCategories != 100 || top.Summary.ShownCategories != maxChartCategories {
		t.Fatalf("expected 24/100 categories, got %+v", top.Summary)
	}
	if top.Data.Labels[0] != "DEV-065" {
		t.Fatalf("expected champion DEV-065 first in Top 24, got %q", top.Data.Labels[0])
	}
	if !tt.HasText("最优 24 / 100 编号") || !tt.HasText("全量均值") {
		t.Fatalf("missing Top 24/100 badge or global mean line")
	}
	save(tt, "hundred-2-top24-bar")

	// Switch 对比范围 -> 最差后 24
	click(t, tt, "对比范围")
	tt.Advance(200 * time.Millisecond)
	click(t, tt, "最差后 24")
	tt.Advance(300 * time.Millisecond)
	bottom := a.buildChartPayload(false)
	if bottom.Data.Labels[len(bottom.Data.Labels)-1] != "DEV-088" {
		t.Fatalf("expected worst outlier DEV-088 at tail of Bottom 24, got %q", bottom.Data.Labels[len(bottom.Data.Labels)-1])
	}
	if !tt.HasText("末位 24 / 100 编号") || !tt.HasText("全量均值") {
		t.Fatalf("missing Bottom 24/100 badge or global mean line")
	}
	// Global best DEV-065 must STILL remain in top summary strip even while viewing Bottom 24!
	if !tt.HasText("DEV-065") {
		t.Fatalf("summary strip lost global champion DEV-065 while viewing Bottom 24")
	}
	save(tt, "hundred-3-bottom24-bar")

	// Switch 对比范围 -> 两头各 12 + 波长 (nm) diverging against global mean
	click(t, tt, "对比范围")
	tt.Advance(200 * time.Millisecond)
	click(t, tt, "两头各 12")
	tt.Advance(200 * time.Millisecond)
	click(t, tt, "分析指标")
	tt.Advance(200 * time.Millisecond)
	click(t, tt, "波长 (nm)")
	tt.Advance(300 * time.Millisecond)
	ext := a.buildChartPayload(false)
	if len(ext.Data.Labels) != 24 || ext.Data.Labels[0] != "DEV-088" || ext.Data.Labels[23] != "DEV-065" {
		t.Fatalf("expected extremes 12+12 spanning DEV-088..DEV-065 for lambda, got first=%q last=%q", ext.Data.Labels[0], ext.Data.Labels[len(ext.Data.Labels)-1])
	}
	if !tt.HasText("两头 24 / 100 编号") || !tt.HasText("全量均值") {
		t.Fatalf("missing Extremes 24/100 badge or global mean line for lambda")
	}
	save(tt, "hundred-4-lambda-extremes-bar")

	// 5. Box plot mode aggregates all 100 devices without 24-entry or 8-curve truncation.
	click(t, tt, "箱线")
	tt.Advance(250 * time.Millisecond)
	click(t, tt, "按电流分布")
	tt.Advance(250 * time.Millisecond)
	click(t, tt, "分析指标")
	tt.Advance(200 * time.Millisecond)
	click(t, tt, "功率 (W)")
	tt.Advance(300 * time.Millisecond)
	boxCur := a.buildChartPayload(false)
	if !boxCur.IsBox || boxCur.Summary.TotalCategories != 100 || boxCur.Summary.ValidPoints != 400 {
		t.Fatalf("expected 100-device 400-point box plot, got %+v", boxCur.Summary)
	}
	if len(boxCur.Boxes) != 1 || len(boxCur.Boxes[0]) != 4 || boxCur.Boxes[0][3].Count != 100 {
		t.Fatalf("unexpected box count at 8 A: %+v", boxCur.Boxes)
	}
	if len(boxCur.Boxes[0][3].Outliers) == 0 {
		t.Fatalf("expected outliers DEV-065/DEV-088 at 8 A, got none: %+v", boxCur.Boxes[0][3])
	}
	if !tt.HasText("汇总 100 编号 · 400 点") || !tt.HasText("按站别分布") {
		t.Fatalf("box plot UI missing full-cohort badge or dimension switcher")
	}
	save(tt, "hundred-5-box-currents")

	click(t, tt, "按站别分布")
	tt.Advance(300 * time.Millisecond)
	boxSt := a.buildChartPayload(false)
	if !boxSt.IsBox || len(boxSt.Data.Labels) != 1 || boxSt.Data.Labels[0] != "耦合测试" || boxSt.Boxes[0][0].Count != 100 {
		t.Fatalf("unexpected station box plot payload: %+v", boxSt)
	}
	save(tt, "hundred-6-box-station")
}

func TestBoxPlotStatsAndDimensions(t *testing.T) {
	// 1. Quartiles, Tukey 1.5*IQR whiskers, and outlier detection
	pts := []boxPoint{
		{Entry: "D1", Value: 10},
		{Entry: "D2", Value: 11},
		{Entry: "D3", Value: 12},
		{Entry: "D4", Value: 13},
		{Entry: "D5", Value: 14},
		{Entry: "D6", Value: 15},
		{Entry: "D7", Value: 16},
		{Entry: "D8", Value: 30}, // upper outlier (> Q3 + 1.5*IQR)
	}
	b := computeBoxStats(pts)
	if !b.Valid || b.Count != 8 || b.Min != 10 || b.Max != 30 || b.MaxEntry != "D8" || b.MinEntry != "D1" {
		t.Fatalf("unexpected basic box stats: %+v", b)
	}
	if math.Abs(b.Median-13.5) > 1e-9 {
		t.Fatalf("unexpected median: %v, want 13.5", b.Median)
	}
	if len(b.Outliers) != 1 || b.Outliers[0].Entry != "D8" || b.WhiskerHigh != 16 {
		t.Fatalf("unexpected Tukey outlier detection: %+v", b)
	}

	// 2. Multi-station 100-device box plot across currents and stations
	var rows []record
	for i := 1; i <= 100; i++ {
		e := fmt.Sprintf("M-%03d", i)
		for _, cur := range []float64{2, 8} {
			p1 := cur*1.8 + float64(i%11)*0.05
			p2 := p1 * 0.96
			if i == 99 {
				p1 += 1.4
				p2 += 1.3
			}
			rows = append(rows,
				record{Entry: e, Category: "耦合测试", Current: pointer(cur), Power: pointer(p1)},
				record{Entry: e, Category: "老化后测试", Current: pointer(cur), Power: pointer(p2)},
			)
		}
	}
	a := newApplication()
	a.config.CustomTests = "老化后测试"
	a.result = &extractResult{Total: len(rows), Records: rows}
	a.filter()
	a.chartKind = 3
	a.chartGroup = 0
	pCur := a.buildChartPayload(false)
	if !pCur.IsBox || len(pCur.Data.Series) != 2 || pCur.Data.Series[0].Label != "耦合测试" || pCur.Data.Series[1].Label != "老化后测试" {
		t.Fatalf("unexpected multi-station series in box current mode: %+v", pCur.Data.Series)
	}
	if len(pCur.Data.Labels) != 2 || pCur.Data.Labels[0] != "2 A" || pCur.Data.Labels[1] != "8 A" {
		t.Fatalf("unexpected X labels in box current mode: %v", pCur.Data.Labels)
	}

	a.chartGroup = 1
	pSt := a.buildChartPayload(false)
	if !pSt.IsBox || !slices.Equal(pSt.Data.Labels, []string{"耦合测试", "老化后测试"}) {
		t.Fatalf("unexpected station X labels in box station mode: %v", pSt.Data.Labels)
	}
	if pSt.Boxes[0][0].Median <= pSt.Boxes[0][1].Median {
		t.Fatalf("expected 耦合测试 median > 老化后测试 median: %v <= %v", pSt.Boxes[0][0].Median, pSt.Boxes[0][1].Median)
	}
	if dir := os.Getenv("DATA_COS_HUNDRED_SCREENSHOT_DIR"); dir != "" {
		a.resultTab = 1
		tt := ui.NewTesterWith(a.view, 1440, 900, ui.TesterOptions{Dark: false})
		tt.Advance(250 * time.Millisecond)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "hundred-7-box-multistation.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()
	}
}

func TestChartPayloadCacheAndLTTBDownsampling(t *testing.T) {
	// 1. Verify LTTB preserves endpoints, sharp isolated spikes, and NaN breaks.
	const n = 10000
	xs := make([]float32, n)
	vals := make([]float64, n)
	for i := 0; i < n; i++ {
		xs[i] = float32(i) / float32(n-1)
		vals[i] = 10.0 + math.Sin(float64(i)*0.01)*0.5
	}
	vals[4321] = 42.0 // isolated spike that uniform sampling would skip
	dxs, dvals := downsampleSeriesLTTB(xs, vals, 120)
	if len(dxs) != 120 || len(dvals) != 120 {
		t.Fatalf("expected 120 LTTB points, got %d", len(dvals))
	}
	if dxs[0] != xs[0] || dvals[0] != vals[0] || dxs[len(dxs)-1] != xs[n-1] || dvals[len(dvals)-1] != vals[n-1] {
		t.Fatalf("LTTB failed to preserve exact start/end points")
	}
	foundSpike := false
	for _, v := range dvals {
		if v == 42.0 {
			foundSpike = true
			break
		}
	}
	if !foundSpike {
		t.Fatal("LTTB dropped isolated spike at index 4321")
	}

	gxs := make([]float32, 1000)
	gvals := make([]float64, 1000)
	for i := range gvals {
		gxs[i] = float32(i) / 999
		gvals[i] = float64(i) * 0.01
	}
	gvals[500] = math.NaN()
	gxsOut, gvalsOut := downsampleSeriesLTTB(gxs, gvals, 80)
	nanBreaks := 0
	for i, v := range gvalsOut {
		if math.IsNaN(v) {
			nanBreaks++
		}
		if i > 0 && gxsOut[i] < gxsOut[i-1] {
			t.Fatalf("LTTB output out of monotonic X order at %d", i)
		}
	}
	if nanBreaks != 1 {
		t.Fatalf("expected 1 preserved NaN break, got %d", nanBreaks)
	}

	// 2. Verify full 10,000-record continuous sweep (4 devices x 2,500 currents),
	//    payload caching across non-data interactions, and safe LTTB/bar thinning in UI.
	const steps = 2500
	devices := []string{"CH-01", "CH-02", "CH-03", "CH-04"}
	rows := make([]record, 0, steps*len(devices))
	for i := 0; i < steps; i++ {
		cur := 0.01 * float64(i+1) // 0.01 A .. 25.00 A
		for di, dev := range devices {
			p := cur*(0.68+0.02*float64(di)) - 0.008*cur*cur
			if di == 1 && i == 1233 {
				p += 3.5 // narrow thermal spike at 12.34 A
			}
			v := 1.8 + 0.04*cur
			eff := 42.0 + 6.0*math.Sin(cur/10.0) - float64(di)*0.8
			rows = append(rows, record{
				Entry:      dev,
				Category:   "老化测试",
				Current:    pointer(cur),
				Power:      pointer(p),
				Voltage:    pointer(v),
				Efficiency: pointer(eff),
			})
		}
	}

	a := newApplication()
	a.config.Entries = strings.Join(devices, "\n")
	a.result = &extractResult{Total: len(rows), Records: rows}
	a.filter()
	a.resetChartDefaults()

	payload := a.buildChartPayload(false)
	if payload.Summary.ValidPoints != len(rows) {
		t.Fatalf("expected all %d points preserved without 24-point truncation, got %d", len(rows), payload.Summary.ValidPoints)
	}
	if !payload.HasLinearX || len(payload.XValues) != steps || payload.XValues[0] != 0.01 || payload.XValues[steps-1] != 25.0 {
		t.Fatalf("continuous sweep truncated or invalid X range: len=%d", len(payload.XValues))
	}

	tt := ui.NewTesterWith(a.view, 1280, 820, ui.TesterOptions{})
	click(t, tt, "数据图表")
	tt.Advance(350 * time.Millisecond)

	if len(a.chartCache.lttbSeries) != len(devices) {
		t.Fatalf("expected %d cached LTTB series, got %d", len(devices), len(a.chartCache.lttbSeries))
	}
	for id, cached := range a.chartCache.lttbSeries {
		if len(cached.xs) < 80 || len(cached.xs) > 320 {
			t.Fatalf("unexpected LTTB point budget for %q: got %d", id, len(cached.xs))
		}
	}

	rebuildsBefore := a.chartCache.rebuilds
	// Hovering/changing chartKind/hoverSeries must NOT rebuild aggregated chart payload.
	a.chartState.Index = 500
	a.chartHoverSeries = payload.Data.Series[0].ID
	tt.Advance(100 * time.Millisecond)
	click(t, tt, "面积")
	tt.Advance(250 * time.Millisecond)
	if a.chartCache.rebuilds != rebuildsBefore {
		t.Fatalf("expected payload cache hit on hover & curve/area switch, got rebuilds %d -> %d", rebuildsBefore, a.chartCache.rebuilds)
	}

	// Bar switch on 2,500 continuous points must thin gracefully instead of creating 10k rects.
	click(t, tt, "柱状")
	tt.Advance(250 * time.Millisecond)
	if a.chartCache.rebuilds != rebuildsBefore {
		t.Fatalf("expected payload cache hit switching to bar view, got rebuilds %d -> %d", rebuildsBefore, a.chartCache.rebuilds)
	}

	// Switching metric and modifying table selection MUST invalidate payload cache.
	click(t, tt, "分析指标")
	tt.Advance(200 * time.Millisecond)
	click(t, tt, "电压 (V)")
	tt.Advance(300 * time.Millisecond)
	if a.chartCache.rebuilds != rebuildsBefore+1 {
		t.Fatalf("expected 1 payload rebuild on metric change, got %d -> %d", rebuildsBefore, a.chartCache.rebuilds)
	}

	rebuildsMetric := a.chartCache.rebuilds
	a.selection.Add(a.rows[0].identity())
	selPayload := a.buildChartPayload(false)
	if !selPayload.Summary.FocusedSelection || a.chartCache.rebuilds != rebuildsMetric+1 {
		t.Fatalf("selection change did not invalidate chart payload cache")
	}
}
