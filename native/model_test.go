package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xuri/excelize/v2"
)

func pointer(n float64) *float64 { return &n }

func TestCurrentPoints(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want []float64
	}{
		{"", []float64{}}, {"a", nil}, {"A", nil}, {"2, 4，8~10", []float64{2, 4, 8, 9, 10}},
		{"3-1", []float64{3, 2, 1}}, {"-2~-1", []float64{-2, -1}}, {"1e-3,2e-3", []float64{.001, .002}},
		{"1.5~2.5", []float64{1.5, 2.5}}, {"2 2\n4", []float64{2, 4}}, {"8 ~ 10", []float64{8, 9, 10}},
	} {
		t.Run(test.raw, func(t *testing.T) {
			got, err := parseCurrents(test.raw)
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %v, %v; want %v", got, err, test.want)
			}
		})
	}
	for _, raw := range []string{"2foo", "NaN", "Inf", "1~1000000", "1~", "a,2", "1e400"} {
		if got, err := parseCurrents(raw); err == nil {
			t.Errorf("accepted %q: %v", raw, got)
		}
	}
}

func TestRequestModesAndValidation(t *testing.T) {
	s := defaultSettings()
	s.Entries = "模块-01\n模块-01，模块-02"
	r, err := s.request()
	if err != nil || len(r.Entries) != 2 || len(r.TestCategories) != 6 || r.CurrentPoints == nil {
		t.Fatalf("request %+v, %v", r, err)
	}
	s.Tests = [6]bool{}
	if _, err := s.request(); err == nil {
		t.Fatal("empty module categories accepted")
	}
	s.Mode = 1
	s.ChipRoots = "/root/a\n/root/b"
	r, err = s.request()
	if err != nil || r.Mode != "chip" || len(r.TestCategories) != 0 || len(r.ChipRoots) != 2 {
		t.Fatalf("chip request %+v, %v", r, err)
	}
	s.Measurements = [3]bool{}
	if _, err := s.request(); err == nil {
		t.Fatal("empty measurements accepted")
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := defaultSettings()
	s.Entries = "中文路径"
	s.Tests[2] = false
	s.CustomTests = "老化后测试\n温循后测试"
	s.setCustomTestChecked("老化后测试", false)
	s.deleteTest("Pre测试")
	s.Mode = 1
	if err := saveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := loadSettings(path)
	if err != nil || !reflect.DeepEqual(got, s) {
		t.Fatalf("%+v %v", got, err)
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSettings(path); err == nil {
		t.Fatal("corrupt settings ignored")
	}
}

func TestExcelRetainsNumbersAndStrings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.xlsx")
	r := &extractResult{Records: []record{{Entry: "=1+1", Category: "耦合测试", Current: pointer(2), Power: pointer(5.123456), Lambda: pointer(965.25)}}, Errors: []string{"missing file"}}
	if err := exportExcel(path, r); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	value, _ := f.GetCellValue("提取结果", "A2")
	formula, _ := f.GetCellFormula("提取结果", "A2")
	if value != "=1+1" || formula != "" {
		t.Fatalf("string interpreted as formula: %q %q", value, formula)
	}
	value, _ = f.GetCellValue("提取结果", "D2")
	if value != "5.123456" {
		t.Fatalf("precision lost: %s", value)
	}
	value, _ = f.GetCellValue("提取提示", "A1")
	if value != "missing file" {
		t.Fatalf("missing diagnostics: %s", value)
	}
	data, err := delimited(r.Records, ',')
	if err != nil || !strings.Contains(string(data), "'=1+1") || !strings.Contains(string(data), "5.123456") {
		t.Fatalf("unsafe CSV: %s %v", data, err)
	}
}

func waitTask(t *testing.T, a *application) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for a.busy && time.Now().Before(deadline) {
		a.finishTasks()
		time.Sleep(time.Millisecond)
	}
	if a.busy {
		t.Fatal("task did not finish")
	}
}

func TestTaskCancellationKeepsPreviousResult(t *testing.T) {
	a := newApplication()
	old := &extractResult{Records: []record{{Entry: "previous"}}}
	a.result = old
	a.config.Entries = "sample"
	a.worker = func(ctx context.Context, _ string, _ extractRequest) (*extractResult, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	a.run()
	if !a.busy {
		t.Fatal("task did not start")
	}
	a.cancel()
	waitTask(t, a)
	if a.result != old || a.statusError {
		t.Fatalf("cancel discarded previous result: %+v", a)
	}
}

func TestTaskSnapshotAndFiltering(t *testing.T) {
	a := newApplication()
	a.config.Entries = "first"
	requests := make(chan extractRequest, 1)
	a.worker = func(ctx context.Context, _ string, r extractRequest) (*extractResult, error) {
		requests <- r
		return &extractResult{Total: 2, Records: []record{{Entry: "first", Category: "Post测试", Current: pointer(10)}, {Entry: "second", Category: "耦合测试", Current: pointer(2)}}}, nil
	}
	a.run()
	a.config.Entries = "changed"
	waitTask(t, a)
	if r := <-requests; r.Entries[0] != "first" {
		t.Fatal("in-flight request changed")
	}
	a.sort = ui.SortOrder{Column: "电流 / A"}
	a.sortRows()
	if *a.rows[0].Current != 2 {
		t.Fatal("numeric sort used string order")
	}
	a.query = "POST"
	a.filter()
	if len(a.rows) != 1 || a.rows[0].Entry != "first" {
		t.Fatalf("search failed: %+v", a.rows)
	}
	a.clearResult()
	if a.result != nil || a.query != "" || len(a.rows) != 0 {
		t.Fatal("clear left stale data")
	}
}

func TestNativeViewAtDesktopSizes(t *testing.T) {
	for _, size := range [][2]float32{{1440, 900}, {1000, 720}} {
		t.Run(fmtSize(size), func(t *testing.T) {
			a := newApplication()
			tt := ui.NewTester(a.view, int(size[0]), int(size[1]))
			if !tt.HasText("从一组编号开始") {
				t.Fatal("missing empty state")
			}
			if run, ok := tt.Access("开始提取"); !ok || !run.Disabled {
				t.Fatal("run enabled for empty entries")
			}
			for _, name := range []string{"开始提取", "编号或完整路径清单", "电流点 / A", "耦合测试", "低温储存后测试", "高温测试"} {
				bounds, ok := tt.Find(name)
				if !ok || bounds.W <= 0 || bounds.H <= 0 || bounds.X < 0 || bounds.Y < 0 || bounds.X+bounds.W > size[0] || bounds.Y+bounds.H > size[1] {
					t.Fatalf("primary control %s out of viewport: %+v", name, bounds)
				}
			}
			input, _ := tt.Find("编号或完整路径清单")
			if input.H < 120 || input.H > 160 {
				t.Fatalf("entry area must remain compact: %+v", input)
			}
			results, ok := tt.Find("提取结果工作区")
			if !ok || results.H < size[1]-120 {
				t.Fatalf("result area lost vertical space: %+v", results)
			}
			if dir := os.Getenv("DATA_COS_SCREENSHOT_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				file, err := os.Create(filepath.Join(dir, fmt.Sprintf("after-native-%dx%d.png", int(size[0]), int(size[1]))))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, tt.Image())
				file.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := tt.Click("编号或完整路径清单"); err != nil {
				t.Fatal(err)
			}
			tt.Type("中文编号-01\n中文编号-02")
			if a.config.Entries != "中文编号-01\n中文编号-02" {
				t.Fatalf("Chinese input failed: %s", a.config.Entries)
			}
			if run, _ := tt.Access("开始提取"); run.Disabled {
				t.Fatal("run still disabled")
			}
			if err := tt.Click("清空测试站别"); err != nil {
				t.Fatal(err)
			}
			if _, err := a.config.request(); err == nil {
				t.Fatal("empty station selection accepted")
			}
			if err := tt.Click("全选测试站别"); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("自定义测试站别"); err != nil {
				t.Fatal(err)
			}
			if !a.customOpen {
				t.Fatal("dialog did not open")
			}
			tt.Key(0, ui.KeyEscape)
			if a.customOpen {
				t.Fatal("Escape did not dismiss dialog")
			}
			if err := tt.Click("数据目录与测试文件设置"); err != nil {
				t.Fatal(err)
			}
			if !a.settingsOpen {
				t.Fatal("settings did not open")
			}
			if err := tt.Click("完成"); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("芯片"); err != nil {
				t.Fatal(err)
			}
			if a.config.Mode != 1 {
				t.Fatal("mode did not change")
			}
			for _, name := range []string{"开始提取", "编号或完整路径清单", "电流点 / A"} {
				bounds, ok := tt.Find(name)
				if !ok || bounds.W <= 0 || bounds.H <= 0 || bounds.X < 0 || bounds.Y < 0 || bounds.X+bounds.W > size[0] || bounds.Y+bounds.H > size[1] {
					t.Fatalf("%s out of viewport: %+v", name, bounds)
				}
			}
			for _, node := range tt.AccessAll() {
				if node.Role == ui.RoleButton && node.Name == "" {
					t.Fatal("unnamed button")
				}
			}
		})
	}
}

func fmtSize(s [2]float32) string { data, _ := json.Marshal(s); return string(data) }

func TestRealEngineFixtures(t *testing.T) {
	engine := os.Getenv("DATA_COS_TEST_ENGINE")
	if engine == "" {
		t.Skip("set DATA_COS_TEST_ENGINE to the built extraction worker")
	}
	root := os.Getenv("DATA_COS_FIXTURES_DIR")
	if root == "" {
		root = t.TempDir()
	}
	module, chip := filepath.Join(root, "module-01"), filepath.Join(root, "chip-01")
	for _, folder := range []string{filepath.Join(module, "耦合测试", "测试"), filepath.Join(chip, "postQCW")} {
		if err := os.MkdirAll(folder, 0700); err != nil {
			t.Fatal(err)
		}
		for _, data := range []struct {
			name string
			skip int
			rows [][]any
		}{
			{"LVI", 18, [][]any{{1., 3.5, 1.8, .41}, {2., 5., 2., .48}, {4., 7.3, 2.3, .565}}},
			{"Rth", 8, [][]any{{964.8, 1.1, 1.}, {965.2, 2.2, 2.}, {965.7, 3.1, 4.}}},
		} {
			f := excelize.NewFile()
			for i := 1; i <= data.skip; i++ {
				if err := f.SetCellStr("Sheet1", fmt.Sprintf("A%d", i), "metadata"); err != nil {
					t.Fatal(err)
				}
			}
			for i, row := range data.rows {
				if err := f.SetSheetRow("Sheet1", fmt.Sprintf("A%d", data.skip+i+1), &row); err != nil {
					t.Fatal(err)
				}
			}
			if err := f.SaveAs(filepath.Join(folder, "202611131200="+data.name+".xlsx")); err != nil {
				t.Fatal(err)
			}
			f.Close()
		}
		f := excelize.NewFile()
		for i, row := range [][]any{{"current", "lambda"}, {1., 970.1}, {2., 970.2}, {4., 970.4}} {
			if err := f.SetSheetRow("Sheet1", fmt.Sprintf("A%d", i+1), &row); err != nil {
				t.Fatal(err)
			}
		}
		if err := f.SaveAs(filepath.Join(folder, "202611131200=lambd.xlsx")); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	for _, mode := range []string{"module", "chip"} {
		for _, points := range [][]float64{[]float64{}, nil, {2}} {
			t.Run(mode+fmt.Sprint(points, points == nil), func(t *testing.T) {
				entry := module
				if mode == "chip" {
					entry = chip
				}
				r, err := runEngine(context.Background(), engine, extractRequest{Mode: mode, Entries: []string{entry}, TestCategories: []string{"耦合测试"}, Measurements: []string{"LVI", "Rth"}, CurrentPoints: points})
				if err != nil || len(r.Errors) > 0 {
					t.Fatalf("engine: %+v %v", r, err)
				}
				expected := 1
				if points == nil {
					expected = 3
				}
				if len(r.Records) != expected {
					t.Fatalf("got %d records, want %d", len(r.Records), expected)
				}
				if len(points) > 0 {
					row := r.Records[0]
					if row.Current == nil || *row.Current != 2 || row.Power == nil || *row.Power != 5 || row.Voltage == nil || *row.Voltage != 2 || row.Lambda == nil || *row.Lambda != 965.2 || row.Efficiency == nil || *row.Efficiency != 48 {
						t.Fatalf("wrong merged data: %+v", row)
					}
				}
			})
		}
		t.Run(mode+"-lambd", func(t *testing.T) {
			entry := module
			if mode == "chip" {
				entry = chip
			}
			r, err := runEngine(context.Background(), engine, extractRequest{Mode: mode, Entries: []string{entry}, TestCategories: []string{"耦合测试"}, Measurements: []string{"lambd"}, CurrentPoints: []float64{2}})
			if err != nil {
				t.Fatal(err)
			}
			if len(r.Errors) > 0 || len(r.Records) != 1 || r.Records[0].Lambda == nil || *r.Records[0].Lambda != 970.2 {
				t.Fatalf("lambd extraction: %+v", r)
			}
		})
	}
}
