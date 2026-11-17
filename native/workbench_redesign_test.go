package main

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// Synthetic measurements keep before/after captures independent of local data.
func redesignFixture() *application {
	a := newApplication()
	var entries []string
	for dev := 1; dev <= 5; dev++ {
		entries = append(entries, fmt.Sprintf("FCP261113%02d", dev))
	}
	a.config.Entries = strings.Join(entries, "\n")
	a.config.Tests = [6]bool{true}
	a.config.CustomTests = "老化后测试"
	a.config.Currents = "a"
	var records []record
	for dev := 0; dev < 4; dev++ {
		for station, name := range []string{"耦合测试", "老化后测试"} {
			for _, cur := range []float64{1, 2, 4, 8} {
				records = append(records, record{Entry: entries[dev], Category: name,
					Current: pointer(cur), Power: pointer(cur*1.95 - float64(station)*0.068),
					Voltage: pointer(1.6 + cur*0.11), Efficiency: pointer(42.5),
					Lambda: pointer(964.6 + cur*0.3), Shift: pointer(-0.6), TwoA: pointer(965.2)})
			}
		}
	}
	a.completions <- taskOutcome{result: &extractResult{Records: records, Total: len(records), Errors: []string{
		"FCP26111305：耦合测试文件不存在", "FCP26111305：老化后测试文件不存在",
	}}}
	a.finishTasks()
	return a
}

func saveRedesignCapture(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("DATA_COS_REDESIGN_SCREENSHOT_DIR")
	if dir == "" {
		return
	}
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

func TestApprovedWorkbenchCaptures(t *testing.T) {
	for _, size := range [][2]int{{1440, 842}, {1000, 720}} {
		for _, dark := range []bool{false, true} {
			a := redesignFixture()
			tt := ui.NewTesterWith(a.view, size[0], size[1], ui.TesterOptions{Dark: dark})
			if a.resultTab != 0 || a.issueCount() != 2 || !tt.HasText("查找提示  2") {
				t.Fatal("partial success must retain its table and counted issues tab")
			}
			for _, n := range tt.AccessAll() {
				if n.Role == ui.RoleStatus && n.Name != "32 条记录" {
					t.Fatalf("error detail leaked into status: %q", n.Name)
				}
				if strings.Contains(n.Name, "文件不存在") || strings.Contains(n.Name, "部分完成") {
					t.Fatalf("diagnostic leaked outside issues tab: %q", n.Name)
				}
			}
			for _, label := range []string{"开始提取", "自定义测试站别", "删除选中站别"} {
				n, ok := tt.Access(label)
				if !ok || n.Bounds.X+n.Bounds.W > a.sidebarWidth || n.Bounds.Y+n.Bounds.H > float32(size[1])-28 {
					t.Fatalf("control clipped at approved sidebar width: %s %+v", label, n)
				}
			}
			header, _ := tt.AccessByRole(ui.RoleColumnHeader, "编号")
			region, _ := tt.Access("提取结果工作区")
			maxToolbarHeight := float32(56)
			if size[0] == 1000 {
				maxToolbarHeight = 92
			}
			if header.Bounds.Y-region.Bounds.Y > maxToolbarHeight {
				t.Fatal("unused warning-banner space remains above table")
			}
			saveRedesignCapture(t, tt, fmt.Sprintf("results-%dx%d-dark-%t", size[0], size[1], dark))
			click(t, tt, "查找提示  2")
			tt.Advance(250 * time.Millisecond)
			if !tt.HasText(a.result.Errors[0]) || !tt.HasText(a.result.Errors[1]) {
				t.Fatal("issues tab lost diagnostic details")
			}
			saveRedesignCapture(t, tt, fmt.Sprintf("issues-%dx%d-dark-%t", size[0], size[1], dark))
		}
	}
}

func TestDiagnosticsStayInIssuesTab(t *testing.T) {
	a := redesignFixture()
	previous := a.result
	a.completions <- taskOutcome{err: errors.New("engine process failed")}
	a.finishTasks()
	if a.result != previous || a.resultTab != 2 || a.issueCount() != 3 {
		t.Fatal("engine failure must preserve results and appear in issues")
	}
	a.reportError("engine process failed")
	if a.issueCount() != 3 {
		t.Fatal("repeated failure duplicated the diagnostic")
	}
	tt := ui.NewTester(a.view, 1000, 720)
	if !tt.HasText("engine process failed") {
		t.Fatal("fatal engine error missing from issues tab")
	}
	a.clearResult()
	tt.Frame()
	if a.issueCount() != 0 || tt.HasText("engine process failed") {
		t.Fatal("clear left stale error messages")
	}
	a.config.Currents = "invalid-current"
	tt.Frame()
	if a.issueCount() != 1 || tt.HasText("无法读取电流：invalid-current") {
		t.Fatal("validation text must stay inside the issues tab")
	}
	current, _ := tt.AccessByRole(ui.RoleTextField, "电流点 / A")
	if !current.Invalid {
		t.Fatal("validation must retain the field's invalid state")
	}
	click(t, tt, "查找提示  1")
	if !tt.HasText("无法读取电流：invalid-current") {
		t.Fatal("invalid field has no diagnostic explanation")
	}
	a.config.Currents = "a"
	tt.Frame()
	if a.issueCount() != 0 || !tt.HasText("暂无查找提示") {
		t.Fatal("corrected input left stale validation issues")
	}
	a.completions <- taskOutcome{result: &extractResult{Errors: []string{"no source"}}}
	a.finishTasks()
	tt.Frame()
	if a.resultTab != 2 || !tt.HasText("no source") {
		t.Fatal("zero-record failures must open issues automatically")
	}
	a.completions <- taskOutcome{result: &extractResult{Records: previous.Records}}
	a.finishTasks()
	tt.Frame()
	if a.resultTab != 0 || a.issueCount() != 0 || tt.HasText("no source") {
		t.Fatal("successful retry left stale diagnostics")
	}
}

func TestStationGroupingSurvivesCopySortAndScroll(t *testing.T) {
	a := redesignFixture()
	a.query = "FCP26111301"
	a.filter()
	tt := ui.NewTester(a.view, 1440, 842)
	countStationLabels := func() int {
		n := 0
		for _, node := range tt.AccessAll() {
			if node.Role == ui.RoleText && node.Bounds.X > a.sidebarWidth && (node.Name == "耦合测试" || node.Name == "老化后测试") {
				n++
			}
		}
		return n
	}
	if n := countStationLabels(); n != 2 {
		t.Fatalf("expected one label per station, got %d", n)
	}
	header, _ := tt.AccessByRole(ui.RoleColumnHeader, "测试站别")
	current, _ := tt.AccessByRole(ui.RoleColumnHeader, "电流 / A")
	var secondRow ui.Rect
	rows := 0
	for _, node := range tt.AccessAll() {
		if node.Role == ui.RoleRow && node.Bounds.Y > header.Bounds.Y+header.Bounds.H {
			if rows == 1 {
				secondRow = node.Bounds
				break
			}
			rows++
		}
	}
	tt.RightClickAt(header.Bounds.X+header.Bounds.W/2, secondRow.Y+secondRow.H/2)
	if err := tt.ChooseMenuItem("复制单元格"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "耦合测试" {
		t.Fatal("hidden continuation label lost its real station value")
	}
	tt.ClickAt(current.Bounds.X+current.Bounds.W/2, current.Bounds.Y+current.Bounds.H/2)
	if n := countStationLabels(); n != 8 {
		t.Fatalf("numeric sort must restore every station label, got %d", n)
	}
	a.sort = ui.SortOrder{}
	a.query = ""
	a.filter()
	tt.Frame()
	a.table.List.ScrollTo(3, ui.Start)
	tt.Frame()
	tt.Frame()
	first, _ := a.table.List.Visible()
	if first == 0 {
		t.Fatal("scroll did not reach a continuation row")
	}
	if !tt.HasText(a.rows[first].Entry) || !tt.HasText(a.rows[first].Category) {
		t.Fatal("virtual scrolling lost the first visible row's group context")
	}
}
