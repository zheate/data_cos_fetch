package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	base "github.com/egoist/mygo/ui/chart"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/components/chart"
)

type chartStat struct {
	label, value, detail string
}

func chartStats(ins chartInsights, metric metricDef, summary chartSummary) []chartStat {
	value := func(v float64) string { return formatMetricNumber(v, metric.Unit, summary.BaseStation != "") }
	if summary.BaseStation != "" {
		mean, worst := value(ins.Mean), value(ins.WorstAbsVal)
		if ins.HasMeanPct {
			mean += " (" + formatMetricNumber(ins.DeltaMeanPct, "%", true) + ")"
		}
		if ins.HasMaxAbsPct {
			worst += " (" + formatMetricNumber(ins.MaxAbsPct, "%", true) + ")"
		}
		return []chartStat{{"平均变化", mean, "基准：" + baseStationDisplayName(summary.BaseStation)},
			{"最大变化", worst, ins.WorstAbsOwner}, {"极差", formatMetricNumber(ins.Range, metric.Unit, false), "最大变化 − 最小变化"},
			{"有效点", fmt.Sprint(ins.Count), "当前统计范围"}}
	}
	stats := []chartStat{{"最高 / 最优", value(ins.MaxVal), ins.MaxOwner}, {"最低", value(ins.MinVal), ins.MinOwner}}
	switch metric.Direction {
	case dirLowerBest:
		stats = []chartStat{{"最低 / 最优", value(ins.MinVal), ins.MinOwner}, {"最高", value(ins.MaxVal), ins.MaxOwner}}
	case dirZeroBest:
		stats = []chartStat{{"最稳", formatMetricNumber(ins.BestAbsVal, metric.Unit, true), ins.BestAbsOwner},
			{"最大漂移", formatMetricNumber(ins.WorstAbsVal, metric.Unit, true), ins.WorstAbsOwner}}
	case dirRangeOnly:
		return []chartStat{{"均值", value(ins.Mean), fmt.Sprintf("%d 个有效点", ins.Count)},
			{"极差", formatRangeWithSpread(ins, metric), "最高值 − 最低值"},
			{"区间", value(ins.MinVal) + " ~ " + value(ins.MaxVal), "当前统计范围"}}
	}
	return append(stats, chartStat{"均值", value(ins.Mean), fmt.Sprintf("%d 个有效点", ins.Count)},
		chartStat{"极差", formatRangeWithSpread(ins, metric), "最高值 − 最低值"})
}

func renderChartStats(c *ui.Context, stats []chartStat, width float32) {
	t := c.Theme()
	columns := len(stats)
	if width < 560 {
		columns = min(columns, 2)
	}
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		for start := 0; start < len(stats); start += columns {
			ui.Row(c).FillWidth().AlignItems(ui.Stretch).Children(func() {
				for j := start; j < min(start+columns, len(stats)); j++ {
					stat := stats[j]
					box := ui.Column(c).Grow(1).Basis(0).MinWidth(0).Height(76).Padding(4, 12).Gap(4)
					if j > start {
						box.BorderWidth(0, 0, 0, 1).BorderColor(t.Border)
					}
					box.Children(func() {
						components.Text(c, stat.label).FontSize(12).TextColor(t.TextMuted).SingleLine()
						v := components.Text(c, stat.value).FontSize(19).FontWeight(600).FontFeatures("tnum").SingleLine().MinWidth(0)
						components.Tooltip(c, v, stat.value)
						d := components.Text(c, stat.detail).FontSize(11).TextColor(t.TextMuted).SingleLine().MinWidth(0)
						components.Tooltip(c, d, stat.detail)
					})
				}
			})
		}
	})
}

type chartBoxDetail struct {
	label, context string
	stats          boxStats
}

// A station comparison shows each station at the chosen current. A current
// distribution shows each visible series at the focused (or last) current.
func chartBoxDetails(payload chartPayload, state *chart.State, group int) []chartBoxDetail {
	var out []chartBoxDetail
	for j, series := range payload.Data.Series {
		if state.Hidden[series.ID] || j >= len(payload.Boxes) {
			continue
		}
		for i, category := range payload.Data.Labels {
			if group == 0 {
				target := state.Index
				if target < 0 || target >= len(payload.Data.Labels) {
					target = len(payload.Data.Labels) - 1
				}
				if i != target {
					continue
				}
			}
			if i >= len(payload.Boxes[j]) || !payload.Boxes[j][i].Valid {
				continue
			}
			label, context := category, series.Label
			if group == 0 {
				label, context = series.Label, category
			}
			out = append(out, chartBoxDetail{label, context, payload.Boxes[j][i]})
		}
	}
	return out
}

func chartBoxStats(payload chartPayload, state *chart.State, metric metricDef, group int) []chartStat {
	details := chartBoxDetails(payload, state, group)
	var stats []chartStat
	if len(details) <= 2 {
		for _, d := range details {
			stats = append(stats, chartStat{d.label + "中位数", formatMetricNumber(d.stats.Median, metric.Unit, payload.Summary.BaseStation != ""), d.context})
		}
	} else {
		stats = append(stats, chartStat{"分布组", fmt.Sprint(len(details)), "当前展示范围"})
	}
	count, outliers, minCount, maxCount := 0, 0, int(^uint(0)>>1), 0
	for j, series := range payload.Data.Series {
		if state.Hidden[series.ID] || j >= len(payload.Boxes) {
			continue
		}
		for _, b := range payload.Boxes[j] {
			if b.Valid {
				count += b.Count
				outliers += len(b.Outliers)
				minCount, maxCount = min(minCount, b.Count), max(maxCount, b.Count)
			}
		}
	}
	nText := fmt.Sprintf("%d 个", maxCount)
	if minCount != maxCount && maxCount > 0 {
		nText = fmt.Sprintf("%d–%d 个", minCount, maxCount)
	}
	stats = append(stats, chartStat{"统计异常点", fmt.Sprintf("%d 个", outliers), "各分布组分别计算"}, chartStat{"每组样本", nText, fmt.Sprintf("共 %d 个有效点", count)})
	return stats
}

func chartTitle(metric metricDef, payload chartPayload, group int) string {
	name := strings.TrimSuffix(metric.Label, " ("+metric.Unit+")")
	if payload.IsBox {
		if group == 1 {
			return name + "分布 · " + payload.ActiveCompare.Label
		}
		return name + "—电流分布"
	}
	if group == 1 {
		return name + "—编号对比"
	}
	return name + "—电流曲线"
}

func (a *application) chartCaption(summary chartSummary, isBox bool) string {
	switch {
	case summary.FocusedSelection:
		if summary.TotalCategories > summary.ShownCategories && summary.ShownCategories > 0 {
			return fmt.Sprintf("已聚焦选中 (%d / %d 编号)", summary.ShownCategories, summary.TotalCategories)
		}
		if isBox {
			return fmt.Sprintf("已聚焦选中 (%d 编号)", summary.TotalCategories)
		}
		return fmt.Sprintf("已聚焦选中 (%d 组)", summary.ShownSeries)
	case isBox:
		return fmt.Sprintf("汇总 %d 编号 · %d 点", summary.TotalCategories, summary.ValidPoints)
	case a.chartGroup == 0 && summary.SampledExtremes:
		return fmt.Sprintf("极值代表 %d / %d 组", summary.ShownSeries, summary.TotalSeries)
	case a.chartGroup == 1 && summary.TotalCategories > summary.ShownCategories && summary.ShownCategories > 0:
		prefix := "前"
		switch a.chartCompareWindow {
		case 0:
			if a.chartSortByValue {
				prefix = "最优"
			}
		case 1:
			prefix = "后"
			if a.chartSortByValue {
				prefix = "末位"
			}
		case 2:
			prefix = "两头"
		}
		return fmt.Sprintf("%s %d / %d 编号", prefix, summary.ShownCategories, summary.TotalCategories)
	case summary.TotalSeries > summary.ShownSeries:
		return fmt.Sprintf("显示前 %d / %d 组", summary.ShownSeries, summary.TotalSeries)
	}
	return fmt.Sprintf("%d 组序列 · %d 个有效点", summary.ShownSeries, summary.ValidPoints)
}

func chartDetailRow(c *ui.Context, label, value string) {
	ui.Row(c).FillWidth().Gap(8).Children(func() {
		components.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted).NoWrap()
		ui.Spacer(c)
		v := components.Text(c, value).FontSize(12).FontFeatures("tnum").SingleLine().MinWidth(0)
		components.Tooltip(c, v, value)
	})
}

func (a *application) chartSidebar(c *ui.Context, data chart.Data, payload chartPayload, metric metricDef, height, width float32) {
	t := c.Theme()
	state := &a.chartState
	nextHover := ""
	ui.Column(c.Key("chart-details")).Width(width).Height(height).Padding(0, 0, 0, 16).BorderWidth(0, 0, 0, 1).BorderColor(t.Border).Gap(8).Label("图表图例与统计").Children(func() {
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
			heading := "序列"
			if payload.IsBox {
				heading = "分布统计"
			}
			components.Text(c, heading).FontSize(13).FontWeight(600)
			ui.Spacer(c)
			if !payload.IsBox && a.chartGroup == 0 && len(data.Labels) > 0 {
				components.Text(c, data.Labels[len(data.Labels)-1]+" / "+metric.Unit).FontSize(11).TextColor(t.TextMuted).NoWrap()
			}
		})
		components.ScrollView(c.Key("chart-legend-scroll"), components.ScrollViewOptions{}).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				for idx, ser := range data.Series {
					visible := !state.Hidden[ser.ID]
					b := components.Button(c.Key("chart-legend-"+ser.ID), "", components.ButtonGhost).Label(ser.Label).Selected(visible).
						FillWidth().MinWidth(0).Padding(8, 2).Radius(4).Cursor(ui.CursorPointer)
					bg := ui.Transparent
					if b.Pressed() {
						bg = t.SurfacePressed
					} else if b.Hovered() || (visible && a.chartHoverSeries == ser.ID) {
						bg = t.SurfaceHover
					}
					b.Background(bg)
					if b.Hovered() && visible {
						nextHover = ser.ID
					}
					fg, shade := t.Text, ser.Color
					if !visible {
						fg, shade = t.TextMuted, t.Border
					}
					b.Children(func() {
						ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Box(c).Size(20, 12).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
								if payload.IsBox {
									p.Fill(r, shade.Alpha(0.16), 0)
									p.Stroke(r, shade, 0, 1)
								} else if a.chartKind == 2 {
									p.Fill(r, shade, 0)
								} else {
									p.Line(r.X, r.Y+r.H/2, r.X+r.W, r.Y+r.H/2, 1.5, shade)
									drawSeriesMarker(p, r.X+r.W/2, r.Y+r.H/2, 6, idx, shade, t.Surface)
								}
							})
							parts := strings.SplitN(ser.Label, " · ", 2)
							ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(3).Children(func() {
								components.Text(c, parts[0]).FontSize(12).FontWeight(500).TextColor(fg).SingleLine()
								if len(parts) > 1 {
									components.Text(c, parts[1]).FontSize(11).TextColor(t.TextMuted).SingleLine()
								}
							})
							if !payload.IsBox && a.chartGroup == 0 && len(data.Labels) > 0 {
								text := "—"
								if i := len(data.Labels) - 1; i < len(ser.Values) {
									text = formatMetricNumber(ser.Values[i], "", payload.Summary.BaseStation != "")
								}
								components.Text(c, text).FontSize(12).FontFeatures("tnum").TextColor(fg).NoWrap()
							}
						})
					})
					// Double-click arrives on pointer-down; its following release
					// must not toggle the freshly soloed series a second time.
					skipRelease := ui.Local(b, "chart-legend-double-release", func() bool { return false })
					if b.DoubleClicked() {
						*skipRelease = true
						toggleSoloSeries(state, data.Series, ser.ID)
						nextHover = ""
						c.Invalidate()
					} else if b.Clicked() && !*skipRelease {
						if state.Hidden == nil {
							state.Hidden = make(map[string]bool)
						}
						state.Hidden[ser.ID] = visible
						nextHover = ""
						c.Invalidate()
					}
					if !b.Pressed() {
						*skipRelease = false
					}
				}
				if hasHiddenChartSeries(data.Series, state.Hidden) {
					if components.Button(c.Key("chart-show-all"), "全部显示", components.ButtonGhost).FillWidth().Height(28).Cursor(ui.CursorPointer).Clicked() {
						clear(state.Hidden)
						nextHover = ""
						c.Invalidate()
					}
				}
				if payload.IsBox {
					for _, d := range chartBoxDetails(payload, state, a.chartGroup) {
						ui.Column(c).FillWidth().Padding(6, 0).Gap(4).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
							components.Text(c, d.label).FontSize(12).FontWeight(600).SingleLine()
							if len(data.Series) > 1 || a.chartGroup == 0 {
								components.Text(c, d.context).FontSize(11).TextColor(t.TextMuted).SingleLine()
							}
							chartDetailRow(c, "中位数", formatMetricNumber(d.stats.Median, metric.Unit, payload.Summary.BaseStation != ""))
							chartDetailRow(c, "Q1 · Q3", formatMetricNumber(d.stats.Q1, "", false)+" · "+formatMetricNumber(d.stats.Q3, "", false))
							chartDetailRow(c, "样本 / 异常点", fmt.Sprintf("%d / %d", d.stats.Count, len(d.stats.Outliers)))
						})
					}
				}
			})
		})
		if payload.IsBox {
			help := "箱体：中间 50% 数据\n中线：中位数\n圆点：统计异常点\n须：1.5 倍四分位距内的最远样本"
			if width < 200 {
				key := components.Text(c, "箱体与异常点说明").FontSize(11).TextColor(t.TextMuted)
				components.Tooltip(c, key, help)
			} else {
				ui.Column(c).FillWidth().Gap(3).Children(func() {
					for _, text := range strings.Split(help, "\n") {
						components.Text(c, text).FontSize(11).TextColor(t.TextMuted)
					}
				})
			}
		}

		components.Text(c, "单击显隐 · 双击独显").FontSize(11).TextColor(t.TextMuted)
	})
	if nextHover != a.chartHoverSeries {
		a.chartHoverSeries = nextHover
		c.Invalidate()
	}
}

// Precise data and interpolation stay in chart_view.go. These positions only
// select a linear current axis for the same original samples.
func chartCurrentScale(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 1
	}
	lo, hi := values[0], values[len(values)-1]
	if lo >= 0 && lo <= hi*0.25 {
		lo = 0
	}
	if lo == hi {
		hi = lo + 1
	}
	return lo, hi
}

func finiteChartValue(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Keep labels sparse without dropping any outlier markers or raw tooltip data.
func renderBoxAnnotations(c *ui.Context, payload chartPayload, visible []int, band base.Band, scale base.Linear, selected int, metric metricDef) {
	n := 0
	for _, j := range visible {
		if j < len(payload.Boxes) {
			for _, b := range payload.Boxes[j] {
				n += len(b.Outliers)
			}
		}
	}
	for k, j := range visible {
		if j >= len(payload.Boxes) {
			continue
		}
		for i, b := range payload.Boxes[j] {
			if !b.Valid {
				continue
			}
			span := band.Band(i)
			cx := span.Lo + (span.Hi-span.Lo)*(float32(k)+0.5)/float32(len(visible))
			if len(visible) == 1 {
				base.Anchor(c, cx, 0, ui.Center, ui.End, func() {
					components.Text(c, fmt.Sprintf("n=%d", b.Count)).FontSize(10).TextColor(c.Theme().TextMuted).NoWrap().Margin(0, 0, 4, 0)
				})
			}
			if n > 4 && (i != selected || len(b.Outliers) > 4) {
				continue
			}
			for _, op := range b.Outliers {
				if !finiteChartValue(op.Value) {
					continue
				}
				align := ui.Start
				if cx > 0.7 {
					align = ui.End
				}
				base.Anchor(c, cx, scale.At(op.Value), align, ui.End, func() {
					text := op.Entry + " · " + formatMetricNumber(op.Value, "", payload.Summary.BaseStation != "")
					label := components.Text(c, text).FontSize(10).TextColor(c.Theme().Text).NoWrap().Width(120).Margin(0, 0, 4, 6)
					components.Tooltip(c, label, text)
				})
			}
		}
	}
}

func hasHiddenChartSeries(series []chart.Series, hidden map[string]bool) bool {
	for _, s := range series {
		if hidden[s.ID] {
			return true
		}
	}
	return false
}
