package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func chartDesignFixture(box bool) *application {
	a := newApplication()
	a.config.Tests = [6]bool{true}
	a.config.CustomTests = "老化后测试"
	a.config.Currents = "a"
	var rows []record
	var entries []string
	if box {
		for i := 1; i <= 100; i++ {
			entry := fmt.Sprintf("M-%03d", i)
			entries = append(entries, entry)
			for _, cur := range []float64{2, 8} {
				p1 := cur*1.8 + float64(i%11)*0.05
				p2 := p1 * 0.96
				if i == 99 {
					p1 += 1.4
					p2 += 1.3
				}
				rows = append(rows,
					record{Entry: entry, Category: "耦合测试", Current: pointer(cur), Power: pointer(p1)},
					record{Entry: entry, Category: "老化后测试", Current: pointer(cur), Power: pointer(p2)})
			}
		}
	} else {
		entries = []string{"FCP-01", "FCP-02"}
		for s, spec := range []struct{ entry, station string }{{"FCP-01", "耦合测试"}, {"FCP-01", "老化后测试"}, {"FCP-02", "耦合测试"}} {
			powers := [3][4]float64{{1.42, 3.18, 6.45, 7.92}, {1.38, 3.10, 6.28, 7.70}, {1.50, 3.31, 6.78, 8.24}}
			for i, cur := range []float64{2, 4, 8, 10} {
				rows = append(rows, record{Entry: spec.entry, Category: spec.station, Current: pointer(cur), Power: pointer(powers[s][i])})
			}
		}
	}
	a.config.Entries = strings.Join(entries, "\n")
	a.result = &extractResult{Records: rows, Total: len(rows)}
	a.filter()
	a.resetChartDefaults()
	a.resultTab = 1
	if box {
		a.chartKind, a.chartGroup = 3, 1
		a.chartCompareCurrent = "cur:8"
	}
	return a
}

func saveChartDesign(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("DATA_COS_CHART_REDESIGN_DIR")
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

func TestApprovedChartDesignCaptures(t *testing.T) {
	for _, box := range []bool{false, true} {
		for _, size := range [][2]int{{1440, 842}, {1000, 720}} {
			for _, dark := range []bool{false, true} {
				a := chartDesignFixture(box)
				// Preserve the approved 100% layout captures at fixed window sizes.
				a.zoom = 1
				tt := ui.NewTesterWith(a.view, size[0], size[1], ui.TesterOptions{Dark: dark})
				for _, label := range []string{"分析指标", "对比基准", "图表类型", "图表维度", "图表图例与统计", "功率 (W) 数据图表"} {
					n, ok := tt.Access(label)
					if !ok || n.Bounds.W <= 0 || n.Bounds.X+n.Bounds.W > float32(size[0]) || n.Bounds.Y+n.Bounds.H > float32(size[1])-28 {
						t.Fatalf("missing or clipped chart component %q at %v: %+v", label, size, n)
					}
				}
				if box && (!tt.HasText("14.65 W") || !tt.HasText("14.064 W") || !tt.HasText("M-099 · 15.124") || !tt.HasText("n=100")) {
					t.Fatal("box plot lost calculated statistics or outlier labels")
				}
				if !box {
					for _, ser := range a.buildChartPayload(dark).Data.Series {
						count := 0
						for _, n := range tt.AccessAll() {
							if n.Role == ui.RoleButton && n.Name == ser.Label {
								count++
							}
						}
						if count != 1 {
							t.Fatalf("series %q must have exactly one interactive legend, got %d", ser.Label, count)
						}
					}
				}
				saveChartDesign(t, tt, fmt.Sprintf("box-%t-%dx%d-dark-%t", box, size[0], size[1], dark))
			}
		}
	}
}

func TestApprovedChartInteractionsAndPrecision(t *testing.T) {
	for _, box := range []bool{false, true} {
		a := chartDesignFixture(box)
		if !box {
			a.result.Records[0].Power = pointer(1.420123456789)
			a.filter()
		}
		before, _ := json.Marshal(a.result.Records)
		clipboard, _ := delimited(a.result.Records, '\t')
		tt := ui.NewTester(a.view, 1440, 842)
		payload := a.buildChartPayload(false)
		ser := payload.Data.Series[0]
		button, _ := tt.AccessByRole(ui.RoleButton, ser.Label)
		tt.Move(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
		if a.chartHoverSeries != ser.ID {
			t.Fatal("legend hover did not highlight its series")
		}
		clickChartLegend(t, tt, ser.Label)
		if !a.chartState.Hidden[ser.ID] || !tt.HasText("全部显示") {
			t.Fatalf("single-click legend visibility failed: box=%t hidden=%v reset=%t", box, a.chartState.Hidden, tt.HasText("全部显示"))
		}
		click(t, tt, "全部显示")
		if hasHiddenChartSeries(payload.Data.Series, a.chartState.Hidden) {
			t.Fatal("show-all left hidden series")
		}
		tt.Advance(time.Second)
		clickChartLegend(t, tt, ser.Label)
		clickChartLegend(t, tt, ser.Label)
		for _, other := range payload.Data.Series {
			if a.chartState.Hidden[other.ID] != (other.ID != ser.ID) {
				t.Fatalf("double-click legend did not solo the requested series: box=%t target=%q hidden=%v", box, ser.ID, a.chartState.Hidden)
			}
		}
		tt.Advance(time.Second)
		clickChartLegend(t, tt, ser.Label)
		clickChartLegend(t, tt, ser.Label)
		if hasHiddenChartSeries(payload.Data.Series, a.chartState.Hidden) {
			t.Fatal("second double-click did not restore the other series")
		}
		if hasHiddenChartSeries(payload.Data.Series, a.chartState.Hidden) {
			click(t, tt, "全部显示")
		}
		plot, _ := tt.Access("功率 (W) 数据图表")
		tt.ClickAt(plot.Bounds.X+plot.Bounds.W/2, plot.Bounds.Y+plot.Bounds.H/2)
		tt.Key(0, ui.KeyHome)
		if a.chartState.Index != 0 {
			t.Fatal("Home did not select the first original sample")
		}
		tt.Key(0, ui.KeyEnd)
		if a.chartState.Index != len(payload.Data.Labels)-1 {
			t.Fatal("End did not select the last original sample")
		}
		tt.Key(0, ui.KeyEscape)
		if a.chartState.Index != -1 {
			t.Fatal("Escape did not clear the focused sample")
		}
		after, _ := json.Marshal(a.result.Records)
		afterClipboard, _ := delimited(a.result.Records, '\t')
		if !bytes.Equal(before, after) || !bytes.Equal(clipboard, afterClipboard) {
			t.Fatal("chart interactions changed original records or export/copy precision")
		}
		if !box && !bytes.Contains(afterClipboard, []byte("1.420123456789")) {
			t.Fatal("original numeric precision was rounded")
		}
	}
}

func TestApprovedBoxStatisticsRemainExact(t *testing.T) {
	a := chartDesignFixture(true)
	p := a.buildChartPayload(false)
	want := []struct{ median, q1, q3, low, high, outlier float64 }{
		{14.65, 14.5, 14.8, 14.4, 14.9, 15.8},
		{14.064, 13.92, 14.208, 13.824, 14.304, 15.124},
	}
	if len(p.Boxes) != 1 || len(p.Boxes[0]) != 2 {
		t.Fatal("station comparison changed aggregation dimensions")
	}
	for i, w := range want {
		b := p.Boxes[0][i]
		if b.Count != 100 || len(b.Outliers) != 1 || b.Outliers[0].Entry != "M-099" {
			t.Fatal("box plot changed sample/outlier membership")
		}
		for j, got := range []float64{b.Median, b.Q1, b.Q3, b.WhiskerLow, b.WhiskerHigh, b.Outliers[0].Value} {
			expect := []float64{w.median, w.q1, w.q3, w.low, w.high, w.outlier}[j]
			if math.Abs(got-expect) > 1e-12 {
				t.Fatalf("box statistics lost precision: got %.15g want %.15g", got, expect)
			}
		}
	}
	ui.NewTester(a.view, 1440, 842)
	a.query = "M-099"
	a.filter()
	filtered := a.buildChartPayload(false)
	for _, boxes := range filtered.Boxes {
		for _, b := range boxes {
			if b.Valid && (b.Count != 1 || len(b.Outliers) != 0 || b.Q1 != b.Q3) {
				t.Fatal("single-sample filtering used stale full-cohort statistics")
			}
		}
	}
	// Displaying the same cached payload never mutates its series colors/data.
	if len(p.Data.Series) != 1 || p.Data.Series[0].Color != chartSeriesColor(false, 0) {
		t.Fatal("box presentation mutated the cached payload")
	}
}

// The current axis may legitimately include zero. Inspect only Y-axis ticks.
func chartHasYTick(tt *ui.Tester, text string) bool {
	var plot ui.Rect
	for _, n := range tt.AccessAll() {
		if strings.HasSuffix(n.Name, " 数据图表") {
			plot = n.Bounds
			break
		}
	}
	for _, n := range tt.AccessAll() {
		if n.Name == text && n.Bounds.X < plot.X && n.Bounds.X >= plot.X-60 && n.Bounds.Y >= plot.Y-10 && n.Bounds.Y < plot.Y+plot.H {
			return true
		}
	}
	return false
}

func clickChartLegend(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	b, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok || b.Bounds.W <= 0 {
		t.Fatalf("missing legend button %q", label)
	}
	tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
}
