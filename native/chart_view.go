package main

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
	base "github.com/egoist/mygo/ui/chart"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/components/chart"
	"github.com/oneo-me/onevra/source/icons/lucide"
)

const (
	maxChartSeries     = 8
	maxChartCategories = 24
)

type metricDef struct {
	ID         string
	Label      string
	Unit       string
	TightScale bool
	Extract    func(record) *float64
}

var chartMetrics = []metricDef{
	{ID: "power", Label: "功率 (W)", Unit: "W", Extract: func(r record) *float64 { return r.Power }},
	{ID: "voltage", Label: "电压 (V)", Unit: "V", TightScale: true, Extract: func(r record) *float64 { return r.Voltage }},
	{ID: "efficiency", Label: "效率 (%)", Unit: "%", Extract: func(r record) *float64 { return r.Efficiency }},
	{ID: "lambda", Label: "波长 (nm)", Unit: "nm", TightScale: true, Extract: func(r record) *float64 { return r.Lambda }},
	{ID: "shift", Label: "波长漂移 (nm)", Unit: "nm", Extract: func(r record) *float64 { return r.Shift }},
	{ID: "twoa", Label: "2A 波长 (nm)", Unit: "nm", TightScale: true, Extract: func(r record) *float64 { return r.TwoA }},
	{ID: "cold", Label: "冷波长 (nm)", Unit: "nm", TightScale: true, Extract: func(r record) *float64 { return r.Cold }},
}

type chartSummary struct {
	FocusedSelection bool
	ShownSeries      int
	TotalSeries      int
	ValidPoints      int
	BaseStation      string
}

type chartPointMeta struct {
	Record   *record
	Baseline *record
}

type chartPayload struct {
	Data    chart.Data
	Summary chartSummary
	Points  map[string][]chartPointMeta
}

type baselineKey struct {
	entry      string
	hasCurrent bool
	current    float64
}

type tipSource struct {
	x, y          float32
	over, pointer bool
	index         int
}

// 8 restrained, distinct colors aligned with the monochrome minimalist theme.
func chartSeriesColor(dark bool, index int) ui.Color {
	light := [maxChartSeries]ui.Color{
		ui.Hex("#27272a"),
		ui.Hex("#4f7396"),
		ui.Hex("#4f7d6a"),
		ui.Hex("#9e7b42"),
		ui.Hex("#85638c"),
		ui.Hex("#488287"),
		ui.Hex("#a65f52"),
		ui.Hex("#636996"),
	}
	darkSet := [maxChartSeries]ui.Color{
		ui.Hex("#e4e4e7"),
		ui.Hex("#7ba2c7"),
		ui.Hex("#79ad97"),
		ui.Hex("#cfab6d"),
		ui.Hex("#b491ba"),
		ui.Hex("#78b4b9"),
		ui.Hex("#d48c7e"),
		ui.Hex("#9298c4"),
	}
	if dark {
		return darkSet[index%maxChartSeries]
	}
	return light[index%maxChartSeries]
}

func (a *application) resetChartDefaults() {
	if a.chartMetric < 0 || a.chartMetric >= len(chartMetrics) {
		a.chartMetric = 0
	}
	if len(a.rows) == 0 {
		a.chartBaseStation = ""
		return
	}
	stations := chartStations(a.rows)
	if len(stations) < 2 || !slices.Contains(stations, a.chartBaseStation) {
		a.chartBaseStation = ""
	}
	if !metricHasValues(a.rows, chartMetrics[a.chartMetric]) {
		for i, m := range chartMetrics {
			if metricHasValues(a.rows, m) {
				a.chartMetric = i
				break
			}
		}
	}
	if distinctCurrentCount(a.rows) >= 2 {
		a.chartGroup = 0
		a.chartKind = 0
	} else {
		a.chartGroup = 1
		a.chartKind = 2
	}
}

func metricHasValues(rows []record, m metricDef) bool {
	for _, r := range rows {
		if v := m.Extract(r); isFiniteFloat(v) {
			return true
		}
	}
	return false
}

func isFiniteFloat(v *float64) bool {
	return v != nil && !math.IsNaN(*v) && !math.IsInf(*v, 0)
}

func distinctCurrentCount(rows []record) int {
	seen := make(map[float64]struct{})
	for _, r := range rows {
		if isFiniteFloat(r.Current) {
			seen[*r.Current] = struct{}{}
		}
	}
	return len(seen)
}

func chartStations(rows []record) []string {
	var stations []string
	seen := make(map[string]struct{})
	for _, r := range rows {
		if r.Category == "" {
			continue
		}
		if _, ok := seen[r.Category]; !ok {
			seen[r.Category] = struct{}{}
			stations = append(stations, r.Category)
		}
	}
	return stations
}

func formatCurrentLabel(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64) + " A"
}

func formatMetricNumber(v float64, unit string, signed bool) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "—"
	}
	rounded := math.Round(v*1000) / 1000
	if rounded == 0 {
		rounded = 0
	}
	text := strconv.FormatFloat(rounded, 'f', -1, 64)
	if signed && rounded > 0 {
		text = "+" + text
	}
	if unit != "" {
		return text + " " + unit
	}
	return text
}

func makeBaselineKey(r record) baselineKey {
	if isFiniteFloat(r.Current) {
		return baselineKey{entry: r.Entry, hasCurrent: true, current: *r.Current}
	}
	return baselineKey{entry: r.Entry}
}

func (a *application) chartSourceRows() ([]record, bool) {
	selected := a.selectedRows()
	if len(selected) == 0 {
		return a.rows, false
	}
	if a.chartGroup == 0 {
		// Selecting any point on a curve isolates the whole sweep for that device+station.
		type pairKey struct{ entry, cat string }
		pairs := make(map[pairKey]struct{}, len(selected))
		for _, r := range selected {
			pairs[pairKey{r.Entry, r.Category}] = struct{}{}
		}
		var out []record
		for _, r := range a.rows {
			if _, ok := pairs[pairKey{r.Entry, r.Category}]; ok {
				out = append(out, r)
			}
		}
		return out, true
	}
	entries := make(map[string]struct{}, len(selected))
	for _, r := range selected {
		entries[r.Entry] = struct{}{}
	}
	var out []record
	for _, r := range a.rows {
		if _, ok := entries[r.Entry]; ok {
			out = append(out, r)
		}
	}
	return out, true
}

func resolvePlotPoint(r record, metric metricDef, baseStation string, baselines map[baselineKey]record) (float64, chartPointMeta, bool) {
	recCopy := r
	meta := chartPointMeta{Record: &recCopy}
	raw := metric.Extract(r)
	if !isFiniteFloat(raw) {
		return 0, meta, false
	}
	if baseStation == "" {
		return *raw, meta, true
	}
	baseRec, ok := baselines[makeBaselineKey(r)]
	if !ok {
		return 0, meta, false
	}
	baseVal := metric.Extract(baseRec)
	if !isFiniteFloat(baseVal) {
		return 0, meta, false
	}
	baseCopy := baseRec
	meta.Baseline = &baseCopy
	return *raw - *baseVal, meta, true
}

func buildBaselineMap(rows []record, baseStation string) map[baselineKey]record {
	if baseStation == "" {
		return nil
	}
	out := make(map[baselineKey]record)
	for _, r := range rows {
		if r.Category == baseStation {
			k := makeBaselineKey(r)
			if _, exists := out[k]; !exists {
				out[k] = r
			}
		}
	}
	return out
}

func (a *application) buildChartData() (chart.Data, chartSummary) {
	p := a.buildChartPayload(false)
	return p.Data, p.Summary
}

func (a *application) buildChartPayload(dark bool) chartPayload {
	metric := chartMetrics[a.chartMetric]
	rows, focused := a.chartSourceRows()
	summary := chartSummary{FocusedSelection: focused}
	if len(rows) == 0 {
		return chartPayload{Summary: summary}
	}

	stations := chartStations(rows)
	activeBase := a.chartBaseStation
	if len(stations) < 2 || !slices.Contains(stations, activeBase) {
		activeBase = ""
	}
	summary.BaseStation = activeBase

	baselines := buildBaselineMap(rows, activeBase)
	plotRows := rows
	if activeBase != "" {
		plotRows = make([]record, 0, len(rows))
		for _, r := range rows {
			if r.Category != activeBase {
				plotRows = append(plotRows, r)
			}
		}
	}

	if a.chartGroup == 0 {
		return buildCurrentCurveData(plotRows, metric, summary, activeBase, baselines, dark)
	}
	return buildEntryComparisonData(plotRows, metric, summary, activeBase, baselines, dark)
}

func buildCurrentCurveData(rows []record, metric metricDef, summary chartSummary, baseStation string, baselines map[baselineKey]record, dark bool) chartPayload {
	var currents []float64
	seenCur := make(map[float64]struct{})
	hasNilCurrent := false
	for _, r := range rows {
		if !isFiniteFloat(r.Current) {
			hasNilCurrent = true
			continue
		}
		if _, ok := seenCur[*r.Current]; !ok {
			seenCur[*r.Current] = struct{}{}
			currents = append(currents, *r.Current)
		}
	}
	sort.Float64s(currents)
	if len(currents) > maxChartCategories {
		currents = currents[:maxChartCategories]
	}

	labels := make([]string, len(currents))
	curIdx := make(map[float64]int, len(currents))
	for i, cur := range currents {
		labels[i] = formatCurrentLabel(cur)
		curIdx[cur] = i
	}
	nilIdx := -1
	if hasNilCurrent {
		nilIdx = len(labels)
		labels = append(labels, "峰值/默认")
	}

	type seriesKey struct{ entry, cat string }
	var order []seriesKey
	indexMap := make(map[seriesKey]int)
	var matrix [][]float64
	var metas [][]chartPointMeta
	for _, r := range rows {
		k := seriesKey{r.Entry, r.Category}
		idx, ok := indexMap[k]
		if !ok {
			summary.TotalSeries++
			if len(order) >= maxChartSeries {
				indexMap[k] = -1
				continue
			}
			idx = len(order)
			indexMap[k] = idx
			order = append(order, k)
			vals := make([]float64, len(labels))
			for j := range vals {
				vals[j] = math.NaN()
			}
			matrix = append(matrix, vals)
			metas = append(metas, make([]chartPointMeta, len(labels)))
		} else if idx < 0 {
			continue
		}
		col := nilIdx
		if isFiniteFloat(r.Current) {
			if ci, exists := curIdx[*r.Current]; exists {
				col = ci
			} else {
				continue
			}
		}
		if col < 0 {
			continue
		}
		if val, meta, valid := resolvePlotPoint(r, metric, baseStation, baselines); valid {
			matrix[idx][col] = val
			metas[idx][col] = meta
			summary.ValidPoints++
		}
	}

	series := make([]chart.Series, len(order))
	points := make(map[string][]chartPointMeta, len(order))
	for i, k := range order {
		id := k.entry + "\x00" + k.cat
		series[i] = chart.Series{
			ID:     id,
			Label:  k.entry + " · " + k.cat,
			Values: matrix[i],
			Color:  chartSeriesColor(dark, i),
		}
		points[id] = metas[i]
	}
	summary.ShownSeries = len(series)
	return chartPayload{Data: chart.Data{Labels: labels, Series: series}, Summary: summary, Points: points}
}

func buildEntryComparisonData(rows []record, metric metricDef, summary chartSummary, baseStation string, baselines map[baselineKey]record, dark bool) chartPayload {
	var labels []string
	entryIdx := make(map[string]int)
	for _, r := range rows {
		if _, ok := entryIdx[r.Entry]; ok {
			continue
		}
		if len(labels) >= maxChartCategories {
			break
		}
		entryIdx[r.Entry] = len(labels)
		labels = append(labels, r.Entry)
	}

	multiCurrent := distinctCurrentCount(rows) > 1
	seriesKeyLabel := func(r record) (string, string) {
		if multiCurrent && isFiniteFloat(r.Current) {
			lbl := r.Category + " @ " + formatCurrentLabel(*r.Current)
			return lbl, lbl
		}
		return r.Category, r.Category
	}

	var order []string
	seriesLabels := make(map[string]string)
	seriesIdx := make(map[string]int)
	var matrix [][]float64
	var metas [][]chartPointMeta
	for _, r := range rows {
		ei, inEntries := entryIdx[r.Entry]
		if !inEntries {
			continue
		}
		sid, slbl := seriesKeyLabel(r)
		si, ok := seriesIdx[sid]
		if !ok {
			summary.TotalSeries++
			if len(order) >= maxChartSeries {
				seriesIdx[sid] = -1
				continue
			}
			si = len(order)
			seriesIdx[sid] = si
			seriesLabels[sid] = slbl
			order = append(order, sid)
			vals := make([]float64, len(labels))
			for j := range vals {
				vals[j] = math.NaN()
			}
			matrix = append(matrix, vals)
			metas = append(metas, make([]chartPointMeta, len(labels)))
		} else if si < 0 {
			continue
		}
		if val, meta, valid := resolvePlotPoint(r, metric, baseStation, baselines); valid {
			matrix[si][ei] = val
			metas[si][ei] = meta
			summary.ValidPoints++
		}
	}

	series := make([]chart.Series, len(order))
	points := make(map[string][]chartPointMeta, len(order))
	for i, sid := range order {
		series[i] = chart.Series{
			ID:     sid,
			Label:  seriesLabels[sid],
			Values: matrix[i],
			Color:  chartSeriesColor(dark, i),
		}
		points[sid] = metas[i]
	}
	summary.ShownSeries = len(series)
	return chartPayload{Data: chart.Data{Labels: labels, Series: series}, Summary: summary, Points: points}
}

func siblingContextText(r *record, metric metricDef, base *record) string {
	if r == nil {
		return ""
	}
	if base != nil {
		cur, bv := metric.Extract(*r), metric.Extract(*base)
		if isFiniteFloat(cur) && isFiniteFloat(bv) {
			return fmt.Sprintf("现值 %s · 基准 %s",
				formatMetricNumber(*cur, metric.Unit, false),
				formatMetricNumber(*bv, metric.Unit, false),
			)
		}
		return ""
	}
	var parts []string
	appendMetric := func(id string, v *float64, unit string) {
		if metric.ID == id || !isFiniteFloat(v) || len(parts) >= 3 {
			return
		}
		parts = append(parts, formatMetricNumber(*v, unit, false))
	}
	appendMetric("power", r.Power, "W")
	appendMetric("voltage", r.Voltage, "V")
	appendMetric("efficiency", r.Efficiency, "%")
	appendMetric("lambda", r.Lambda, "nm")
	return strings.Join(parts, " · ")
}

func renderChartTooltipDetail(c *ui.Context, sel chart.Selection, metric metricDef, payload chartPayload) {
	t := c.Theme()
	header := sel.Label
	if payload.Summary.BaseStation != "" {
		header = fmt.Sprintf("%s · 较「%s」", sel.Label, payload.Summary.BaseStation)
	}
	components.Text(c, header).Bold()
	signed := payload.Summary.BaseStation != "" || metric.ID == "shift"
	for _, v := range sel.Values {
		ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(t.Space(1.5)).Children(func() {
				ui.Box(c).Size(8, 8).Shrink(0).Radius(3).Background(v.Color)
				components.Text(c, v.Label).TextColor(t.TextMuted).MinWidth(0).Shrink(1).NoWrap().Ellipsis("…")
				ui.Box(c).Grow(1)
				components.Text(c, formatMetricNumber(v.Value, metric.Unit, signed)).Bold().NoWrap()
			})
			if metas, ok := payload.Points[v.ID]; ok && sel.Index >= 0 && sel.Index < len(metas) {
				if sub := siblingContextText(metas[sel.Index].Record, metric, metas[sel.Index].Baseline); sub != "" {
					components.Text(c, sub).FontSize(max(10, t.FontSize-2)).TextColor(t.TextMuted).Padding(0, 0, 0, 14).SingleLine()
				}
			}
		})
	}
}

// adaptiveLineAreaPlot fits tight non-zero ranges (wavelength, voltage) dynamically
// so nanometer shifts are not flattened against a zero baseline.
func adaptiveLineAreaPlot(c *ui.Context, state *chart.State, data chart.Data, opts chart.Options, areaFill bool) {
	t := c.Theme()
	lo, hi := math.Inf(1), math.Inf(-1)
	var visibleSeries []int
	var values [][]float64
	valid := false
	for j, s := range data.Series {
		if state.Hidden[s.ID] {
			continue
		}
		row := make([]float64, len(data.Labels))
		for i := range row {
			if i < len(s.Values) && !math.IsNaN(s.Values[i]) && !math.IsInf(s.Values[i], 0) {
				v := s.Values[i]
				row[i] = v
				valid = true
				lo = min(lo, v)
				hi = max(hi, v)
			} else {
				row[i] = math.NaN()
			}
		}
		visibleSeries = append(visibleSeries, j)
		values = append(values, row)
	}

	if valid {
		if lo == hi {
			pad := max(0.5, math.Abs(lo)*0.001)
			lo -= pad
			hi += pad
		} else {
			pad := (hi - lo) * 0.12
			lo -= pad
			hi += pad
		}
	}
	yScale := base.NewLinear(lo, hi).Nice(4)
	if !valid || math.IsNaN(yScale.Lo) || math.IsNaN(yScale.Hi) || math.IsInf(yScale.Hi-yScale.Lo, 0) {
		valid = false
	}
	if !valid || state.Index >= len(data.Labels) {
		state.Index = -1
	}

	buildSelection := func(i int) chart.Selection {
		s := chart.Selection{Index: i}
		if i < 0 || i >= len(data.Labels) {
			return s
		}
		s.Label = data.Labels[i]
		for _, j := range visibleSeries {
			ser := data.Series[j]
			if i < len(ser.Values) && !math.IsNaN(ser.Values[i]) && !math.IsInf(ser.Values[i], 0) {
				s.Values = append(s.Values, chart.Value{
					ID:    ser.ID,
					Label: ser.Label,
					Value: ser.Values[i],
					Color: ser.Color,
				})
			}
		}
		return s
	}

	describeSelection := func(sel chart.Selection) string {
		parts := []string{sel.Label}
		for _, v := range sel.Values {
			valText := strconv.FormatFloat(v.Value, 'g', 5, 64)
			if opts.FormatValue != nil {
				valText = opts.FormatValue(v.Value)
			}
			parts = append(parts, v.Label+": "+valText)
		}
		return strings.Join(parts, "; ")
	}

	ui.Column(c).FillWidth().MinWidth(0).Gap(t.Space(3)).Children(func() {
		if !valid {
			ui.Column(c).FillWidth().Height(opts.PlotHeight).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				if opts.Empty != nil {
					opts.Empty(c)
				}
			})
		} else {
			xBand := base.NewBand(data.Labels...).Padding(.3, .3)
			xs := make([]float32, len(data.Labels))
			for i := range xs {
				xs[i] = xBand.At(i)
			}
			yScale.Format = opts.FormatValue
			holder := ui.Column(c).FillWidth().MinWidth(0).Height(opts.PlotHeight).Padding(8, 8, 0, 0)
			width := holder.LayoutBounds().W
			if width <= 0 {
				width = 240
			}
			axisSize := t.FontSize - 2
			tickWidth := t.Space(11)
			for _, label := range data.Labels {
				w, _ := c.MeasureText(0, ui.Span{Text: label, Size: axisSize})
				tickWidth = max(tickWidth, min(t.Space(21), w+t.Space(3)))
			}
			limit := max(1, int((width-48)/tickWidth))
			step := max(1, int(math.Ceil(float64(len(xs))/float64(limit))))
			var xTicks []base.Tick
			for i := 0; i < len(xs); i += step {
				xTicks = append(xTicks, base.Tick{At: xs[i], Label: data.Labels[i]})
			}
			axes := base.Axes{
				Left:   yScale.Ticks(4),
				Bottom: xTicks,
				Label: func(side base.Side, tick base.Tick) {
					lbl := components.Text(c, tick.Label).NoWrap().FontSize(axisSize).TextColor(t.TextMuted)
					if side == base.Bottom {
						lbl.MaxWidth(max(24, tickWidth-8)).Ellipsis("…")
					}
				},
			}
			holder.Children(func() {
				base.Plot(c, axes, func(area ui.Element) {
					area.Label(opts.Label).Description(opts.Description)
					if !area.IsDisabled() {
						positions := xs
						if bw, rw := area.Bounds().W, area.LayoutBounds().W; bw > 0 && rw > 0 && bw != rw {
							positions = make([]float32, len(xs))
							for i, px := range xs {
								positions[i] = px * rw / bw
							}
						}
						base.Select(c, area, &state.Index, positions, func(i int) string {
							return describeSelection(buildSelection(i))
						})
					}
					selected := state.Index
					area.Draw(func(painter *ui.Painter, r ui.Rect) {
						for _, tick := range axes.Left {
							painter.Line(r.X, base.Y(r, tick.At), r.X+r.W, base.Y(r, tick.At), 1, t.Border.Alpha(.45))
						}
						if selected >= 0 && selected < len(xs) {
							w := max(12, r.W/float32(len(xs))*.8)
							painter.Fill(ui.Rect{X: base.X(r, xs[selected]) - w/2, Y: r.Y, W: w, H: r.H}, t.Text.Alpha(.025), 6)
						}
					})
					for k, j := range visibleSeries {
						ser := data.Series[j]
						shade := ser.Color
						ys := make([]float32, len(values[k]))
						for i, v := range values[k] {
							ys[i] = yScale.At(v)
						}
						ui.Box(c).Key(ser.ID).Absolute().Left(0).Right(0).Top(0).Bottom(0).Children(func() {
							ui.Box(c).Absolute().Left(0).Right(0).Top(0).Bottom(0).Draw(func(p *ui.Painter, r ui.Rect) {
								if areaFill {
									p.FillPathGradient(base.Area(r, xs, nil, ys, base.CurveMonotone), ui.LinearGradient{
										From: shade.Alpha(.22), To: shade.Alpha(.02), Angle: 180,
									})
								}
								p.StrokePath(base.Line(r, xs, ys, base.CurveMonotone), 2, shade)
							})
							for i, v := range values[k] {
								if math.IsNaN(v) || math.IsInf(v, 0) || (len(xs) > 32 && i != selected) {
									continue
								}
								if i == selected {
									base.Point(c, xs[i], ys[i], 16, 16).Radius(8).Background(shade.Alpha(.16))
								}
								base.Point(c, xs[i], ys[i], 6, 6).Radius(3).Background(shade)
							}
						})
					}
					if selected >= 0 && selected < len(xs) {
						base.RuleX(c, xs[selected], 1).Background(t.TextMuted.Alpha(.3))
					}
					renderAdaptiveTip(c, area, opts, buildSelection(selected), func() ui.Element {
						return base.RuleX(c, xs[selected], 0)
					})
				}).Fill().MinWidth(0).Gap(10)
			})
		}

		// Keep legend outside the plot so hidden series can be re-enabled at any time.
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(t.Space(1)).Children(func() {
			for _, s := range data.Series {
				ui.Box(c).Key(s.ID).MaxWidthPercent(100).Children(func() {
					visible := !state.Hidden[s.ID]
					b := components.Button(c, "", components.ButtonGhost).Label(s.Label).Selected(visible).
						Padding(t.Space(1), t.Space(2)).Gap(t.Space(1.5)).MaxWidthPercent(100).
						TextColor(t.TextMuted).Background(ui.Color{})
					if visible {
						b.Background(t.Text.Alpha(.045)).TextColor(t.Text)
					}
					b.Children(func() {
						shade := s.Color
						if !visible {
							shade = t.Border
						}
						ui.Box(c).Size(8, 8).Shrink(0).Radius(3).Background(shade)
						components.Text(c, s.Label).FontSize(t.FontSize - 1)
					})
					if b.Clicked() {
						if state.Hidden == nil {
							state.Hidden = make(map[string]bool)
						}
						state.Hidden[s.ID] = visible
						c.Invalidate()
					}
				})
			}
		})
	})
}

func renderAdaptiveTip(c *ui.Context, area ui.Element, opts chart.Options, selection chart.Selection, mark func() ui.Element) {
	src := ui.Local(area, "onevra.chart.adaptive.tip", func() tipSource { return tipSource{index: -1} })
	x, y, over := area.PointerPosition()
	moved := over && (!src.over || x != src.x || y != src.y)
	switch {
	case moved:
		src.pointer = true
	case selection.Index != src.index:
		src.pointer = false
	}
	src.x, src.y, src.over, src.index = x, y, over, selection.Index
	if selection.Index < 0 || (src.pointer && !over) || (!src.pointer && !area.Focused()) {
		return
	}
	var anchor ui.Element
	if src.pointer {
		anchor = ui.Box(c).Absolute().Left(x).Top(y).Size(0, 0)
	} else {
		anchor = mark()
	}
	anchor.PassThrough()
	t := c.Theme()
	ui.Overlay(c, func() {
		ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).
			AttachTo(anchor, ui.AnchorEnd, ui.AnchorStart).MarginStart(t.Space(3)).
			MinWidth(t.Space(30)).MaxWidth(t.Space(80)).Padding(t.Space(2), t.Space(2.5)).Gap(t.Space(1)).
			Radius(t.Radius).Material(glass.Glass{Tint: t.Background.Alpha(.92)}).Border(1, t.Border).
			TextColor(t.Text).FontSize(t.FontSize-1).
			Shadow(0, t.Space(1), t.Space(4), 0, ui.RGBA(0, 0, 0, .14)).Children(func() {
			if opts.Detail != nil {
				opts.Detail(c, selection)
			}
		})
	})
}

func (a *application) chartTabView(c *ui.Context) {
	t := c.Theme()
	if a.chartMetric < 0 || a.chartMetric >= len(chartMetrics) {
		a.chartMetric = 0
	}
	metric := chartMetrics[a.chartMetric]
	payload := a.buildChartPayload(t.Dark)
	data, summary := payload.Data, payload.Summary
	if summary.ValidPoints == 0 {
		data = chart.Data{}
	}

	sourceRows, _ := a.chartSourceRows()
	stations := chartStations(sourceRows)

	_, availHeight := c.Size()
	plotHeight := float32(280)
	if availHeight > 0 {
		plotHeight = max(160, min(400, availHeight-395))
	}

	components.ScrollView(c.Key("chart-scroll"), components.ScrollViewOptions{}).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(12, 16).Gap(10).Children(func() {
			ui.Row(c).FillWidth().Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
				sel := components.Select(c.Key("chart-metric"), &a.chartMetric)
				sel.Trigger.Label("分析指标").Children(func() {
					components.Text(c, metric.Label).FontSize(13).SingleLine()
				})
				sel.Popup(func(panel ui.Element) {
					for i, m := range chartMetrics {
						item := sel.Item(i)
						item.Children(func() {
							components.Text(c, m.Label).FontSize(13).SingleLine()
						})
					}
				})

				if len(stations) >= 2 {
					if !slices.Contains(stations, a.chartBaseStation) {
						a.chartBaseStation = ""
					}
					baseLabel := "原始数值"
					if a.chartBaseStation != "" {
						baseLabel = "较 " + a.chartBaseStation + " (Δ)"
					}
					bs := components.Select(c.Key("chart-base"), &a.chartBaseStation)
					bs.Trigger.Label("对比基准").Children(func() {
						components.Text(c, baseLabel).FontSize(13).SingleLine()
					})
					bs.Popup(func(panel ui.Element) {
						bs.Item("").Children(func() {
							components.Text(c, "原始数值").FontSize(13).SingleLine()
						})
						for _, st := range stations {
							item := bs.Item(st)
							item.Children(func() {
								components.Text(c, "较 "+st+" (Δ)").FontSize(13).SingleLine()
							})
						}
					})
				}

				components.Tabs(c.Key("chart-group"), &a.chartGroup, components.TabsSegmented, "按电流曲线", "按编号对比").
					List.Label("图表维度")

				components.Tabs(c.Key("chart-kind"), &a.chartKind, components.TabsSegmented, "折线", "面积", "柱状").
					List.Label("图表类型")

				ui.Spacer(c)

				if summary.TotalSeries > 0 {
					caption := fmt.Sprintf("%d 组序列", summary.ShownSeries)
					if summary.FocusedSelection {
						caption = fmt.Sprintf("已聚焦选中 (%d 组)", summary.ShownSeries)
					} else if summary.TotalSeries > summary.ShownSeries {
						caption = fmt.Sprintf("显示前 %d / %d 组", summary.ShownSeries, summary.TotalSeries)
					}
					components.Badge(c, caption, components.BadgeOptions{Tone: components.BadgeNeutral})
				}
			})

			ui.Box(c).FillWidth().Radius(t.Radius).Border(1, t.Border).Background(t.Surface).Padding(10, 12).Children(func() {
				deltaMode := summary.BaseStation != ""
				signedAxis := deltaMode || metric.ID == "shift"
				opts := chart.Options{
					Label:       metric.Label + " 数据图表",
					Description: "按电流点或编号对比提取得到的光学与电学参数",
					PlotHeight:  plotHeight,
					FormatValue: func(v float64) string {
						return formatMetricNumber(v, metric.Unit, signedAxis)
					},
					Detail: func(dc *ui.Context, sel chart.Selection) {
						renderChartTooltipDetail(dc, sel, metric, payload)
					},
					Empty: func(ec *ui.Context) {
						title, detail := "暂无可绘制数据", "完成数据提取后，可在此对比各器件参数曲线"
						action := ""
						var onAction func()
						if len(a.rows) > 0 && summary.ValidPoints == 0 {
							title = fmt.Sprintf("当前结果不含「%s」数据", metric.Label)
							detail = "请切换上方的分析指标，或检查是否勾选了对应测试文件（LVI / 光谱）"
						} else if a.result == nil {
							action = "填写编号"
							onAction = func() { a.entriesField.Focus() }
						}
						emptyStateWithIcon(ec, "chart-empty", lucide.ChartLine(), title, detail, action, onAction)
					},
				}

				useAdaptive := len(data.Series) > 0 && !deltaMode && metric.TightScale && a.chartKind != 2
				switch {
				case useAdaptive:
					adaptiveLineAreaPlot(c, &a.chartState, data, opts, a.chartKind == 1)
				case a.chartKind == 1:
					chart.Area(c, &a.chartState, data, opts).FillWidth()
				case a.chartKind == 2:
					chart.Bar(c, &a.chartState, data, opts).FillWidth()
				default:
					chart.Line(c, &a.chartState, data, opts).FillWidth()
				}
			})
		})
	})
}
