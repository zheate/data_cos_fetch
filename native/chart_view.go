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

type metricDirection uint8

const (
	dirHigherBest metricDirection = iota
	dirLowerBest
	dirZeroBest
	dirRangeOnly
)

type metricDef struct {
	ID         string
	Label      string
	Unit       string
	TightScale bool
	Direction  metricDirection
	Extract    func(record) *float64
}

var chartMetrics = []metricDef{
	{ID: "power", Label: "功率 (W)", Unit: "W", Direction: dirHigherBest, Extract: func(r record) *float64 { return r.Power }},
	{ID: "voltage", Label: "电压 (V)", Unit: "V", TightScale: true, Direction: dirLowerBest, Extract: func(r record) *float64 { return r.Voltage }},
	{ID: "efficiency", Label: "效率 (%)", Unit: "%", Direction: dirHigherBest, Extract: func(r record) *float64 { return r.Efficiency }},
	{ID: "lambda", Label: "波长 (nm)", Unit: "nm", TightScale: true, Direction: dirRangeOnly, Extract: func(r record) *float64 { return r.Lambda }},
	{ID: "shift", Label: "波长漂移 (nm)", Unit: "nm", Direction: dirZeroBest, Extract: func(r record) *float64 { return r.Shift }},
	{ID: "twoa", Label: "2A 波长 (nm)", Unit: "nm", TightScale: true, Direction: dirRangeOnly, Extract: func(r record) *float64 { return r.TwoA }},
	{ID: "cold", Label: "冷波长 (nm)", Unit: "nm", TightScale: true, Direction: dirRangeOnly, Extract: func(r record) *float64 { return r.Cold }},
}

type chartSummary struct {
	FocusedSelection bool
	ShownSeries      int
	TotalSeries      int
	ShownCategories  int
	TotalCategories  int
	SampledExtremes  bool
	ValidPoints      int
	BaseStation      string
	CompareCurrent   string
}

type chartPointMeta struct {
	Record   *record
	Baseline *record
}

type boxPoint struct {
	Entry string
	Value float64
}

type boxStats struct {
	Valid       bool
	Count       int
	Min         float64
	Q1          float64
	Median      float64
	Mean        float64
	Q3          float64
	Max         float64
	WhiskerLow  float64
	WhiskerHigh float64
	MinEntry    string
	MaxEntry    string
	Outliers    []boxPoint
}

type chartPayload struct {
	Data           chart.Data
	Summary        chartSummary
	Points         map[string][]chartPointMeta
	GlobalInsights chartInsights
	XValues        []float64
	HasLinearX     bool
	ActiveCompare  compareCurrentOption
	CompareOpts    []compareCurrentOption
	IsBox          bool
	Boxes          [][]boxStats
}

type baselineKey struct {
	entry      string
	hasCurrent bool
	current    float64
}

type compareCurrentOption struct {
	Key     string
	Label   string
	Current *float64
	IsAll   bool
}

type chartInsights struct {
	Count         int
	MinVal        float64
	MaxVal        float64
	Mean          float64
	Range         float64
	MinOwner      string
	MaxOwner      string
	BestAbsVal    float64
	BestAbsOwner  string
	WorstAbsVal   float64
	WorstAbsOwner string
	DeltaMeanPct  float64
	MaxAbsPct     float64
	HasMeanPct    bool
	HasMaxAbsPct  bool
}

type tipSource struct {
	x, y          float32
	over, pointer bool
	index         int
}

// Okabe-Ito inspired publication palette: colorblind-safe, high-contrast on paper & dark surfaces.
func chartSeriesColor(dark bool, index int) ui.Color {
	light := [maxChartSeries]ui.Color{
		ui.Hex("#1d242b"),
		ui.Hex("#0072b2"),
		ui.Hex("#d55e00"),
		ui.Hex("#009e73"),
		ui.Hex("#984ea3"),
		ui.Hex("#d98a00"),
		ui.Hex("#0099b4"),
		ui.Hex("#a65628"),
	}
	darkSet := [maxChartSeries]ui.Color{
		ui.Hex("#e5e7eb"),
		ui.Hex("#5ba8d6"),
		ui.Hex("#ee8133"),
		ui.Hex("#3fc49d"),
		ui.Hex("#c588cf"),
		ui.Hex("#f0b840"),
		ui.Hex("#5bc0da"),
		ui.Hex("#c9845b"),
	}
	if dark {
		return darkSet[index%maxChartSeries]
	}
	return light[index%maxChartSeries]
}

func drawSeriesMarker(p *ui.Painter, cx, cy, size float32, shape int, color, surface ui.Color) {
	r := size / 2
	var path ui.Path
	switch shape % 8 {
	case 0:
		path.Circle(cx, cy, r)
		p.FillPath(&path, color)
	case 1:
		p.Fill(ui.Rect{X: cx - r, Y: cy - r, W: size, H: size}, color, 0)
	case 2:
		path.MoveTo(cx, cy-r*1.15).LineTo(cx+r*1.1, cy+r*0.95).LineTo(cx-r*1.1, cy+r*0.95).Close()
		p.FillPath(&path, color)
	case 3:
		path.MoveTo(cx, cy-r*1.2).LineTo(cx+r*1.2, cy).LineTo(cx, cy+r*1.2).LineTo(cx-r*1.2, cy).Close()
		p.FillPath(&path, color)
	case 4:
		path.Circle(cx, cy, r)
		p.FillPath(&path, surface)
		p.StrokePath(&path, 1.4, color)
	case 5:
		rect := ui.Rect{X: cx - r, Y: cy - r, W: size, H: size}
		p.Fill(rect, surface, 0)
		p.Stroke(rect, color, 0, 1.4)
	case 6:
		path.MoveTo(cx, cy-r*1.15).LineTo(cx+r*1.1, cy+r*0.95).LineTo(cx-r*1.1, cy+r*0.95).Close()
		p.FillPath(&path, surface)
		p.StrokePath(&path, 1.4, color)
	case 7:
		path.MoveTo(cx-r*1.1, cy-r*0.95).LineTo(cx+r*1.1, cy-r*0.95).LineTo(cx, cy+r*1.15).Close()
		p.FillPath(&path, color)
	}
}

func (a *application) resetChartDefaults() {
	if a.chartMetric < 0 || a.chartMetric >= len(chartMetrics) {
		a.chartMetric = 0
	}
	a.chartHoverSeries = ""
	if len(a.rows) == 0 {
		a.chartBaseStation = ""
		a.chartCompareCurrent = ""
		return
	}
	stations := chartStations(a.rows)
	if len(stations) < 2 || !slices.Contains(stations, a.chartBaseStation) {
		a.chartBaseStation = ""
	}
	if a.chartCompareCurrent != "" && a.chartCompareCurrent != "all" {
		opts := chartCompareOptions(a.rows)
		found := false
		for _, opt := range opts {
			if opt.Key == a.chartCompareCurrent {
				found = true
				break
			}
		}
		if !found {
			a.chartCompareCurrent = ""
		}
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

func formatCurrentKey(v float64) string {
	return "cur:" + strconv.FormatFloat(v, 'g', -1, 64)
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

const (
	baseMeanSentinel     = "\x00mean"
	tightSpreadAutoRatio = 0.15
)

func baseStationDisplayName(baseStation string) string {
	if baseStation == baseMeanSentinel {
		return "全量均值"
	}
	return baseStation
}

func baseStationOptionLabel(baseStation string) string {
	if baseStation == "" {
		return "原始数值"
	}
	return "较 " + baseStationDisplayName(baseStation) + " (Δ)"
}

func distinctEntryCount(rows []record) int {
	seen := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		seen[r.Entry] = struct{}{}
	}
	return len(seen)
}

func makeCurrentKey(r record) baselineKey {
	if isFiniteFloat(r.Current) {
		return baselineKey{hasCurrent: true, current: *r.Current}
	}
	return baselineKey{}
}

func assignMetricValue(r *record, metricID string, val float64) {
	v := val
	switch metricID {
	case "power":
		r.Power = &v
	case "voltage":
		r.Voltage = &v
	case "efficiency":
		r.Efficiency = &v
	case "lambda":
		r.Lambda = &v
	case "shift":
		r.Shift = &v
	case "twoa":
		r.TwoA = &v
	case "cold":
		r.Cold = &v
	}
}

func buildMeanBaselineMap(rows []record, metric metricDef) map[baselineKey]record {
	type sumCount struct {
		sum   float64
		count int
	}
	acc := make(map[baselineKey]sumCount)
	for _, r := range rows {
		v := metric.Extract(r)
		if !isFiniteFloat(v) {
			continue
		}
		k := makeCurrentKey(r)
		sc := acc[k]
		sc.sum += *v
		sc.count++
		acc[k] = sc
	}
	out := make(map[baselineKey]record, len(acc))
	for k, sc := range acc {
		if sc.count == 0 {
			continue
		}
		var rec record
		assignMetricValue(&rec, metric.ID, sc.sum/float64(sc.count))
		out[k] = rec
	}
	return out
}

func isValidBaseStation(baseStation string, stations []string, rows []record) bool {
	if baseStation == baseMeanSentinel {
		return distinctEntryCount(rows) >= 2
	}
	return baseStation != "" && len(stations) >= 2 && slices.Contains(stations, baseStation)
}

func shouldAutoMagnifyDiff(metric metricDef, ins chartInsights) bool {
	if metric.TightScale {
		return true
	}
	if metric.ID == "shift" || ins.Count < 2 || ins.Range <= 0 || ins.Mean == 0 {
		return false
	}
	// Auto-zoom only when all values share one sign and sit inside a narrow band relative to mean.
	if ins.MinVal*ins.MaxVal <= 0 {
		return false
	}
	return ins.Range/math.Abs(ins.Mean) <= tightSpreadAutoRatio
}

func shouldAutoMagnifySelection(rows []record, metric metricDef, active compareCurrentOption) bool {
	if metric.TightScale {
		return true
	}
	if metric.ID == "shift" {
		return false
	}
	filtered := filterCompareRows(rows, active)
	var ins chartInsights
	sum := 0.0
	for _, r := range filtered {
		v := metric.Extract(r)
		if !isFiniteFloat(v) {
			continue
		}
		if ins.Count == 0 || *v < ins.MinVal {
			ins.MinVal = *v
		}
		if ins.Count == 0 || *v > ins.MaxVal {
			ins.MaxVal = *v
		}
		sum += *v
		ins.Count++
	}
	if ins.Count < 2 {
		return false
	}
	ins.Mean = sum / float64(ins.Count)
	ins.Range = ins.MaxVal - ins.MinVal
	return shouldAutoMagnifyDiff(metric, ins)
}

func (a *application) resolveZoomDiff(metric metricDef, ins chartInsights, deltaMode bool) bool {
	if deltaMode || (a.chartGroup != 1 && a.chartKind != 3) {
		return false
	}
	if a.chartZoomMetric == a.chartMetric && a.chartZoomPref != 0 {
		return a.chartZoomPref > 0
	}
	return shouldAutoMagnifyDiff(metric, ins)
}

func formatRangeWithSpread(ins chartInsights, metric metricDef) string {
	text := formatMetricNumber(ins.Range, metric.Unit, false)
	if ins.Count >= 2 && ins.Range > 0 && ins.Mean != 0 && ins.MinVal*ins.MaxVal > 0 && metric.Unit != "%" && metric.Unit != "nm" {
		pct := ins.Range / math.Abs(ins.Mean) * 100
		if pct >= 0.05 && pct < 100 {
			text += fmt.Sprintf(" (%.1f %%)", pct)
		}
	}
	return text
}

func makeBaselineKey(r record) baselineKey {
	if isFiniteFloat(r.Current) {
		return baselineKey{entry: r.Entry, hasCurrent: true, current: *r.Current}
	}
	return baselineKey{entry: r.Entry}
}

func chartCompareOptions(rows []record) []compareCurrentOption {
	var currents []float64
	seen := make(map[float64]struct{})
	hasNil := false
	for _, r := range rows {
		if !isFiniteFloat(r.Current) {
			hasNil = true
			continue
		}
		if _, ok := seen[*r.Current]; !ok {
			seen[*r.Current] = struct{}{}
			currents = append(currents, *r.Current)
		}
	}
	sort.Float64s(currents)
	total := len(currents)
	if hasNil {
		total++
	}
	if total <= 1 {
		return nil
	}
	opts := make([]compareCurrentOption, 0, total+1)
	for _, cur := range currents {
		v := cur
		opts = append(opts, compareCurrentOption{
			Key:     formatCurrentKey(cur),
			Label:   formatCurrentLabel(cur),
			Current: &v,
		})
	}
	if hasNil {
		opts = append(opts, compareCurrentOption{Key: "nil", Label: "峰值/默认"})
	}
	opts = append(opts, compareCurrentOption{Key: "all", Label: "全部电流", IsAll: true})
	return opts
}

func resolveCompareOption(rows []record, selected string) (compareCurrentOption, []compareCurrentOption) {
	opts := chartCompareOptions(rows)
	if len(opts) == 0 {
		return compareCurrentOption{IsAll: true}, nil
	}
	if selected != "" {
		for _, opt := range opts {
			if opt.Key == selected {
				return opt, opts
			}
		}
	}
	// Default to the largest numeric current so entry comparisons do not collapse into grouped sub-bars.
	var fallback compareCurrentOption
	for _, opt := range opts {
		if !opt.IsAll && opt.Current != nil {
			fallback = opt
		}
	}
	if fallback.Key == "" {
		fallback = opts[0]
	}
	return fallback, opts
}

func filterCompareRows(rows []record, active compareCurrentOption) []record {
	if active.IsAll {
		return rows
	}
	out := make([]record, 0, len(rows))
	for _, r := range rows {
		if active.Current == nil {
			if !isFiniteFloat(r.Current) {
				out = append(out, r)
			}
		} else if isFiniteFloat(r.Current) && *r.Current == *active.Current {
			out = append(out, r)
		}
	}
	return out
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
	key := makeBaselineKey(r)
	if baseStation == baseMeanSentinel {
		key.entry = ""
	}
	baseRec, ok := baselines[key]
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

type chartSourceKey struct {
	rowsRev uint64
	rowsLen int
	selHash uint64
	group   int
}

type chartCacheKey struct {
	source         chartSourceKey
	dark           bool
	metric         int
	baseStation    string
	compareCurrent string
	compareWindow  int
	sortByValue    bool
	boxMode        bool
}

type lttbSeriesCache struct {
	xs   []float32
	vals []float64
}

type chartCacheState struct {
	sourceValid bool
	sourceKey   chartSourceKey
	sourceRows  []record
	focused     bool
	stations    []string
	hasMeanBase bool

	valid      bool
	key        chartCacheKey
	payload    chartPayload
	rebuilds   int
	lttbBudget int
	lttbBounds [2]float32
	lttbSeries map[string]lttbSeriesCache

	autoZoomValid  bool
	autoZoomSrc    chartSourceKey
	autoZoomMetric int
	autoZoomComp   string
	autoZoomVal    bool
}

func (a *application) chartSelectionHash() uint64 {
	n := a.selection.Len()
	if n == 0 {
		return 0
	}
	var sum, xor uint64
	for k := range a.selection.All() {
		var h uint64 = 14695981039346656037
		if s, ok := k.(string); ok {
			for i := 0; i < len(s); i++ {
				h ^= uint64(s[i])
				h *= 1099511628211
			}
		} else {
			h ^= uint64(n)
		}
		sum += h
		xor ^= h * 0x9e3779b97f4a7c15
	}
	return sum ^ xor ^ uint64(n)
}

func (a *application) cachedSourceSnapshot() ([]record, bool, []string, bool) {
	key := chartSourceKey{
		rowsRev: a.rowsRev,
		rowsLen: len(a.rows),
		selHash: a.chartSelectionHash(),
		group:   a.chartGroup,
	}
	if a.chartCache.sourceValid && a.chartCache.sourceKey == key {
		return a.chartCache.sourceRows, a.chartCache.focused, a.chartCache.stations, a.chartCache.hasMeanBase
	}
	rows, focused := a.chartSourceRows()
	stations := chartStations(rows)
	hasMean := distinctEntryCount(rows) >= 2
	a.chartCache.sourceValid = true
	a.chartCache.sourceKey = key
	a.chartCache.sourceRows = rows
	a.chartCache.focused = focused
	a.chartCache.stations = stations
	a.chartCache.hasMeanBase = hasMean
	a.chartCache.autoZoomValid = false
	return rows, focused, stations, hasMean
}

func (a *application) cachedAutoZoom(metricIdx int, compareKey string) bool {
	rows, _, _, _ := a.cachedSourceSnapshot()
	srcKey := a.chartCache.sourceKey
	if a.chartCache.autoZoomValid &&
		a.chartCache.autoZoomSrc == srcKey &&
		a.chartCache.autoZoomMetric == metricIdx &&
		a.chartCache.autoZoomComp == compareKey {
		return a.chartCache.autoZoomVal
	}
	curCompare, _ := resolveCompareOption(rows, compareKey)
	val := shouldAutoMagnifySelection(rows, chartMetrics[metricIdx], curCompare)
	a.chartCache.autoZoomValid = true
	a.chartCache.autoZoomSrc = srcKey
	a.chartCache.autoZoomMetric = metricIdx
	a.chartCache.autoZoomComp = compareKey
	a.chartCache.autoZoomVal = val
	return val
}

func (a *application) cachedLTTBSeries(seriesID string, xs []float32, vals []float64, budget int) ([]float32, []float64) {
	if budget < 3 || len(xs) <= budget {
		return xs, vals
	}
	bounds := [2]float32{xs[0], xs[len(xs)-1]}
	if a.chartCache.lttbBudget != budget || a.chartCache.lttbBounds != bounds {
		a.chartCache.lttbBudget = budget
		a.chartCache.lttbBounds = bounds
		a.chartCache.lttbSeries = nil
	}
	if entry, ok := a.chartCache.lttbSeries[seriesID]; ok {
		return entry.xs, entry.vals
	}
	dxs, dvals := downsampleSeriesLTTB(xs, vals, budget)
	if a.chartCache.lttbSeries == nil {
		a.chartCache.lttbSeries = make(map[string]lttbSeriesCache)
	}
	a.chartCache.lttbSeries[seriesID] = lttbSeriesCache{xs: dxs, vals: dvals}
	return dxs, dvals
}

func (a *application) buildChartData() (chart.Data, chartSummary) {
	p := a.buildChartPayload(false)
	return p.Data, p.Summary
}

func (a *application) buildChartPayload(dark bool) chartPayload {
	rows, focused, stations, _ := a.cachedSourceSnapshot()
	boxMode := a.chartKind == 3
	key := chartCacheKey{
		source:         a.chartCache.sourceKey,
		dark:           dark,
		metric:         a.chartMetric,
		baseStation:    a.chartBaseStation,
		compareCurrent: a.chartCompareCurrent,
		compareWindow:  a.chartCompareWindow,
		sortByValue:    a.chartSortByValue,
		boxMode:        boxMode,
	}
	if a.chartCache.valid {
		ck := a.chartCache.key
		compareSame := ck.compareCurrent == key.compareCurrent ||
			(ck.compareCurrent == "" && key.compareCurrent != "" && key.compareCurrent == a.chartCache.payload.ActiveCompare.Key)
		if ck.source == key.source &&
			ck.dark == key.dark &&
			ck.metric == key.metric &&
			ck.baseStation == key.baseStation &&
			compareSame &&
			ck.compareWindow == key.compareWindow &&
			ck.sortByValue == key.sortByValue &&
			ck.boxMode == key.boxMode {
			return a.chartCache.payload
		}
	}

	metric := chartMetrics[a.chartMetric]
	summary := chartSummary{FocusedSelection: focused}
	var payload chartPayload
	if len(rows) == 0 {
		payload = chartPayload{Summary: summary, IsBox: boxMode}
	} else {
		activeBase := a.chartBaseStation
		if activeBase != "" && !isValidBaseStation(activeBase, stations, rows) {
			activeBase = ""
		}
		summary.BaseStation = activeBase

		var baselines map[baselineKey]record
		if activeBase == baseMeanSentinel {
			baselines = buildMeanBaselineMap(rows, metric)
		} else {
			baselines = buildBaselineMap(rows, activeBase)
		}
		plotRows := rows
		if activeBase != "" && activeBase != baseMeanSentinel {
			plotRows = make([]record, 0, len(rows))
			for _, r := range rows {
				if r.Category != activeBase {
					plotRows = append(plotRows, r)
				}
			}
		}

		if boxMode {
			if a.chartGroup == 0 {
				payload = buildBoxPlotData(plotRows, metric, summary, activeBase, baselines, dark, 0, compareCurrentOption{})
			} else {
				activeCompare, compareOpts := resolveCompareOption(plotRows, a.chartCompareCurrent)
				if !activeCompare.IsAll && activeCompare.Label != "" {
					summary.CompareCurrent = activeCompare.Label
				}
				payload = buildBoxPlotData(plotRows, metric, summary, activeBase, baselines, dark, 1, activeCompare)
				payload.ActiveCompare = activeCompare
				payload.CompareOpts = compareOpts
			}
		} else if a.chartGroup == 0 {
			payload = buildCurrentCurveData(plotRows, metric, summary, activeBase, baselines, dark)
		} else {
			activeCompare, compareOpts := resolveCompareOption(plotRows, a.chartCompareCurrent)
			if !activeCompare.IsAll && activeCompare.Label != "" {
				summary.CompareCurrent = activeCompare.Label
			}
			payload = buildEntryComparisonViewWindow(plotRows, metric, summary, activeBase, baselines, dark, activeCompare, a.chartSortByValue, a.chartCompareWindow)
			payload.ActiveCompare = activeCompare
			payload.CompareOpts = compareOpts
		}
	}

	a.chartCache.valid = true
	a.chartCache.key = key
	a.chartCache.payload = payload
	a.chartCache.rebuilds++
	a.chartCache.lttbBudget = 0
	a.chartCache.lttbSeries = nil
	return payload
}

func metricRankBetter(va, vb float64, metric metricDef, deltaMode bool) bool {
	if deltaMode {
		switch metric.Direction {
		case dirLowerBest:
			return va < vb
		case dirZeroBest, dirRangeOnly:
			return math.Abs(va) < math.Abs(vb)
		default:
			return va > vb
		}
	}
	switch metric.Direction {
	case dirLowerBest:
		return va < vb
	case dirZeroBest:
		return math.Abs(va) < math.Abs(vb)
	default:
		return va > vb
	}
}

func pickWindowIndices(n, limit, window int) []int {
	if n <= 0 || limit <= 0 {
		return nil
	}
	if n <= limit {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	}
	switch window {
	case 1:
		start := n - limit
		out := make([]int, limit)
		for i := range out {
			out[i] = start + i
		}
		return out
	case 2:
		head := limit / 2
		tail := limit - head
		out := make([]int, 0, limit)
		for i := 0; i < head; i++ {
			out = append(out, i)
		}
		for i := n - tail; i < n; i++ {
			out = append(out, i)
		}
		return out
	default:
		out := make([]int, limit)
		for i := range out {
			out[i] = i
		}
		return out
	}
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
	if (hasNilCurrent || len(currents) < 2) && len(currents) > maxChartCategories {
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
	var sums []float64
	var counts []int
	for _, r := range rows {
		k := seriesKey{r.Entry, r.Category}
		idx, ok := indexMap[k]
		if !ok {
			idx = len(order)
			indexMap[k] = idx
			order = append(order, k)
			vals := make([]float64, len(labels))
			for j := range vals {
				vals[j] = math.NaN()
			}
			matrix = append(matrix, vals)
			metas = append(metas, make([]chartPointMeta, len(labels)))
			sums = append(sums, 0)
			counts = append(counts, 0)
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
			sums[idx] += val
			counts[idx]++
			summary.ValidPoints++
		}
	}

	summary.TotalSeries = len(order)
	allSeries := make([]chart.Series, len(order))
	allPoints := make(map[string][]chartPointMeta, len(order))
	allVis := make([]int, len(order))
	for i, k := range order {
		id := k.entry + "\x00" + k.cat
		allSeries[i] = chart.Series{
			ID:     id,
			Label:  k.entry + " · " + k.cat,
			Values: matrix[i],
		}
		allPoints[id] = metas[i]
		allVis[i] = i
	}
	globalInsights := computeChartInsightsFromSeries(chart.Data{Labels: labels, Series: allSeries}, allPoints, allVis, metric, true)

	chosen := make([]int, 0, min(len(order), maxChartSeries))
	if len(order) <= maxChartSeries {
		for i := range order {
			chosen = append(chosen, i)
		}
	} else {
		summary.SampledExtremes = true
		ranked := make([]int, 0, len(order))
		for i := range order {
			if counts[i] > 0 {
				ranked = append(ranked, i)
			}
		}
		if len(ranked) == 0 {
			for i := range order {
				ranked = append(ranked, i)
			}
		} else {
			deltaMode := baseStation != ""
			sort.SliceStable(ranked, func(i, j int) bool {
				ai, bi := ranked[i], ranked[j]
				return metricRankBetter(sums[ai]/float64(counts[ai]), sums[bi]/float64(counts[bi]), metric, deltaMode)
			})
		}
		for _, pos := range pickWindowIndices(len(ranked), maxChartSeries, 2) {
			chosen = append(chosen, ranked[pos])
		}
	}

	series := make([]chart.Series, len(chosen))
	points := make(map[string][]chartPointMeta, len(chosen))
	for i, origIdx := range chosen {
		s := allSeries[origIdx]
		s.Color = chartSeriesColor(dark, i)
		series[i] = s
		points[s.ID] = allPoints[s.ID]
	}
	summary.ShownSeries = len(series)
	payload := chartPayload{
		Data:           chart.Data{Labels: labels, Series: series},
		Summary:        summary,
		Points:         points,
		GlobalInsights: globalInsights,
	}
	if !hasNilCurrent && len(currents) >= 2 {
		payload.XValues = currents
		payload.HasLinearX = true
	}
	return payload
}

func buildEntryComparisonData(rows []record, metric metricDef, summary chartSummary, baseStation string, baselines map[baselineKey]record, dark bool) chartPayload {
	var allLabels []string
	seen := make(map[string]struct{})
	for _, r := range rows {
		if _, ok := seen[r.Entry]; ok {
			continue
		}
		seen[r.Entry] = struct{}{}
		allLabels = append(allLabels, r.Entry)
	}
	summary.TotalCategories = len(allLabels)
	limit := min(len(allLabels), maxChartCategories)
	labels := allLabels[:limit]
	summary.ShownCategories = len(labels)
	plotted := buildEntryComparisonMatrix(rows, labels, metric, summary, baseStation, baselines, dark)
	if len(allLabels) > maxChartCategories {
		full := buildEntryComparisonMatrix(rows, allLabels, metric, summary, baseStation, baselines, dark)
		vis := make([]int, len(full.Data.Series))
		for i := range vis {
			vis[i] = i
		}
		plotted.GlobalInsights = computeChartInsightsFromSeries(full.Data, full.Points, vis, metric, false)
	} else {
		vis := make([]int, len(plotted.Data.Series))
		for i := range vis {
			vis[i] = i
		}
		plotted.GlobalInsights = computeChartInsightsFromSeries(plotted.Data, plotted.Points, vis, metric, false)
	}
	return plotted
}

func buildEntryComparisonView(rows []record, metric metricDef, summary chartSummary, baseStation string, baselines map[baselineKey]record, dark bool, activeCompare compareCurrentOption, sortByValue bool) chartPayload {
	return buildEntryComparisonViewWindow(rows, metric, summary, baseStation, baselines, dark, activeCompare, sortByValue, 0)
}

func buildEntryComparisonViewWindow(rows []record, metric metricDef, summary chartSummary, baseStation string, baselines map[baselineKey]record, dark bool, activeCompare compareCurrentOption, sortByValue bool, compareWindow int) chartPayload {
	filtered := filterCompareRows(rows, activeCompare)
	if len(filtered) == 0 {
		filtered = rows
	}
	type entryStat struct {
		entry string
		order int
		sum   float64
		count int
	}
	stats := make([]entryStat, 0)
	idxByEntry := make(map[string]int)
	for _, r := range filtered {
		idx, ok := idxByEntry[r.Entry]
		if !ok {
			idx = len(stats)
			idxByEntry[r.Entry] = idx
			stats = append(stats, entryStat{entry: r.Entry, order: idx})
		}
		if val, _, valid := resolvePlotPoint(r, metric, baseStation, baselines); valid {
			stats[idx].sum += val
			stats[idx].count++
		}
	}
	validStats := make([]entryStat, 0, len(stats))
	for _, st := range stats {
		if st.count > 0 {
			validStats = append(validStats, st)
		}
	}
	if len(validStats) == 0 {
		validStats = stats
	}
	summary.TotalCategories = len(validStats)
	if sortByValue && len(validStats) > 1 {
		deltaMode := baseStation != ""
		sort.SliceStable(validStats, func(i, j int) bool {
			a, b := validStats[i], validStats[j]
			if a.count == 0 || b.count == 0 {
				return a.count > b.count
			}
			return metricRankBetter(a.sum/float64(a.count), b.sum/float64(b.count), metric, deltaMode)
		})
	}
	chosen := pickWindowIndices(len(validStats), maxChartCategories, compareWindow)
	labels := make([]string, len(chosen))
	for i, pos := range chosen {
		labels[i] = validStats[pos].entry
	}
	summary.ShownCategories = len(labels)
	plotted := buildEntryComparisonMatrix(filtered, labels, metric, summary, baseStation, baselines, dark)
	if len(validStats) > maxChartCategories {
		allLabels := make([]string, len(validStats))
		for i, st := range validStats {
			allLabels[i] = st.entry
		}
		full := buildEntryComparisonMatrix(filtered, allLabels, metric, summary, baseStation, baselines, dark)
		vis := make([]int, len(full.Data.Series))
		for i := range vis {
			vis[i] = i
		}
		plotted.GlobalInsights = computeChartInsightsFromSeries(full.Data, full.Points, vis, metric, false)
	} else {
		vis := make([]int, len(plotted.Data.Series))
		for i := range vis {
			vis[i] = i
		}
		plotted.GlobalInsights = computeChartInsightsFromSeries(plotted.Data, plotted.Points, vis, metric, false)
	}
	return plotted
}

func buildEntryComparisonMatrix(rows []record, labels []string, metric metricDef, summary chartSummary, baseStation string, baselines map[baselineKey]record, dark bool) chartPayload {
	entryIdx := make(map[string]int, len(labels))
	for i, lbl := range labels {
		entryIdx[lbl] = i
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

func sortedPercentile(sorted []boxPoint, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return math.NaN()
	}
	if n == 1 {
		return sorted[0].Value
	}
	pos := p * float64(n-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo].Value
	}
	frac := pos - float64(lo)
	return sorted[lo].Value + frac*(sorted[hi].Value-sorted[lo].Value)
}

func computeBoxStats(pts []boxPoint) boxStats {
	if len(pts) == 0 {
		return boxStats{}
	}
	sorted := slices.Clone(pts)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Value != sorted[j].Value {
			return sorted[i].Value < sorted[j].Value
		}
		return sorted[i].Entry < sorted[j].Entry
	})
	var sum float64
	for _, p := range sorted {
		sum += p.Value
	}
	n := len(sorted)
	q1 := sortedPercentile(sorted, 0.25)
	med := sortedPercentile(sorted, 0.50)
	q3 := sortedPercentile(sorted, 0.75)
	iqr := q3 - q1
	lowFence := q1 - 1.5*iqr
	highFence := q3 + 1.5*iqr
	whiskerLow := sorted[0].Value
	for _, p := range sorted {
		if p.Value >= lowFence {
			whiskerLow = p.Value
			break
		}
	}
	whiskerHigh := sorted[n-1].Value
	for i := n - 1; i >= 0; i-- {
		if sorted[i].Value <= highFence {
			whiskerHigh = sorted[i].Value
			break
		}
	}
	var outliers []boxPoint
	for _, p := range sorted {
		if p.Value < lowFence || p.Value > highFence {
			outliers = append(outliers, p)
		}
	}
	return boxStats{
		Valid:       true,
		Count:       n,
		Min:         sorted[0].Value,
		Q1:          q1,
		Median:      med,
		Mean:        sum / float64(n),
		Q3:          q3,
		Max:         sorted[n-1].Value,
		WhiskerLow:  whiskerLow,
		WhiskerHigh: whiskerHigh,
		MinEntry:    sorted[0].Entry,
		MaxEntry:    sorted[n-1].Entry,
		Outliers:    outliers,
	}
}

func computeBoxPlotInsights(rows []record, metric metricDef, baseStation string, baselines map[baselineKey]record, visibleSeries map[string]bool, groupMode int) chartInsights {
	ins := chartInsights{
		MinVal: math.Inf(1),
		MaxVal: math.Inf(-1),
	}
	bestAbsMag := math.Inf(1)
	worstAbsMag := -1.0
	var sum, baseSum float64
	var baseCount int
	for _, r := range rows {
		val, meta, valid := resolvePlotPoint(r, metric, baseStation, baselines)
		if !valid {
			continue
		}
		if visibleSeries != nil {
			var sid string
			if groupMode == 0 {
				sid = r.Category
				if sid == "" {
					sid = "全部"
				}
			} else if isFiniteFloat(r.Current) {
				sid = formatCurrentKey(*r.Current)
			} else {
				sid = "cur:nil"
			}
			if !visibleSeries[sid] && !visibleSeries["all"] {
				continue
			}
		}
		catLabel := r.Category
		if catLabel == "" {
			catLabel = "全部"
		}
		curLabel := ""
		if isFiniteFloat(r.Current) {
			curLabel = formatCurrentLabel(*r.Current)
		}
		owner := r.Entry
		if groupMode == 0 && curLabel != "" {
			owner = r.Entry + " · " + curLabel
		} else if catLabel != "" {
			owner = r.Entry + " · " + catLabel
		}
		ins.Count++
		sum += val
		if val < ins.MinVal {
			ins.MinVal = val
			ins.MinOwner = owner
		}
		if val > ins.MaxVal {
			ins.MaxVal = val
			ins.MaxOwner = owner
		}
		absV := math.Abs(val)
		if absV < bestAbsMag {
			bestAbsMag = absV
			ins.BestAbsVal = val
			ins.BestAbsOwner = owner
		}
		if absV > worstAbsMag {
			worstAbsMag = absV
			ins.WorstAbsVal = val
			ins.WorstAbsOwner = owner
			ins.HasMaxAbsPct = false
			if meta.Baseline != nil {
				if bv := metric.Extract(*meta.Baseline); isFiniteFloat(bv) && *bv != 0 {
					ins.MaxAbsPct = val / math.Abs(*bv) * 100
					ins.HasMaxAbsPct = true
				}
			}
		}
		if meta.Baseline != nil {
			if bv := metric.Extract(*meta.Baseline); isFiniteFloat(bv) {
				baseSum += *bv
				baseCount++
			}
		}
	}
	if ins.Count == 0 {
		return chartInsights{}
	}
	ins.Mean = sum / float64(ins.Count)
	ins.Range = ins.MaxVal - ins.MinVal
	if baseCount > 0 && baseSum != 0 {
		ins.DeltaMeanPct = sum / math.Abs(baseSum) * 100
		ins.HasMeanPct = true
	}
	return ins
}

func buildBoxPlotData(rows []record, metric metricDef, summary chartSummary, baseStation string, baselines map[baselineKey]record, dark bool, groupMode int, activeCompare compareCurrentOption) chartPayload {
	workRows := rows
	if groupMode == 1 {
		filtered := filterCompareRows(rows, activeCompare)
		if len(filtered) > 0 {
			workRows = filtered
		}
	}

	var xLabels []string
	xIdxMap := make(map[string]int)
	var seriesIDs []string
	seriesLabels := make(map[string]string)
	seriesIdxMap := make(map[string]int)

	if groupMode == 0 {
		var currents []float64
		seenCur := make(map[float64]struct{})
		hasNil := false
		for _, r := range workRows {
			if !isFiniteFloat(r.Current) {
				hasNil = true
				continue
			}
			if _, ok := seenCur[*r.Current]; !ok {
				seenCur[*r.Current] = struct{}{}
				currents = append(currents, *r.Current)
			}
		}
		sort.Float64s(currents)
		for _, c := range currents {
			lbl := formatCurrentLabel(c)
			xIdxMap[formatCurrentKey(c)] = len(xLabels)
			xLabels = append(xLabels, lbl)
		}
		if hasNil {
			xIdxMap["cur:nil"] = len(xLabels)
			xLabels = append(xLabels, "峰值/默认")
		}
		for _, r := range workRows {
			cat := r.Category
			if cat == "" {
				cat = "全部"
			}
			if _, ok := seriesIdxMap[cat]; !ok {
				if len(seriesIDs) >= maxChartSeries {
					continue
				}
				seriesIdxMap[cat] = len(seriesIDs)
				seriesIDs = append(seriesIDs, cat)
				seriesLabels[cat] = cat
			}
		}
	} else {
		for _, r := range workRows {
			cat := r.Category
			if cat == "" {
				cat = "全部"
			}
			if _, ok := xIdxMap[cat]; !ok {
				xIdxMap[cat] = len(xLabels)
				xLabels = append(xLabels, cat)
			}
		}
		if !activeCompare.IsAll {
			lbl := activeCompare.Label
			if lbl == "" {
				lbl = "全部样本"
			}
			seriesIdxMap["all"] = 0
			seriesIDs = append(seriesIDs, "all")
			seriesLabels["all"] = lbl
		} else {
			var currents []float64
			seenCur := make(map[float64]struct{})
			hasNil := false
			for _, r := range workRows {
				if !isFiniteFloat(r.Current) {
					hasNil = true
					continue
				}
				if _, ok := seenCur[*r.Current]; !ok {
					seenCur[*r.Current] = struct{}{}
					currents = append(currents, *r.Current)
				}
			}
			sort.Float64s(currents)
			for _, c := range currents {
				if len(seriesIDs) >= maxChartSeries {
					break
				}
				sid := formatCurrentKey(c)
				seriesIdxMap[sid] = len(seriesIDs)
				seriesIDs = append(seriesIDs, sid)
				seriesLabels[sid] = formatCurrentLabel(c)
			}
			if hasNil && len(seriesIDs) < maxChartSeries {
				seriesIdxMap["cur:nil"] = len(seriesIDs)
				seriesIDs = append(seriesIDs, "cur:nil")
				seriesLabels["cur:nil"] = "峰值/默认"
			}
		}
	}

	buckets := make([][][]boxPoint, len(seriesIDs))
	for s := range buckets {
		buckets[s] = make([][]boxPoint, len(xLabels))
	}

	validRows := make([]record, 0, len(workRows))
	for _, r := range workRows {
		val, _, valid := resolvePlotPoint(r, metric, baseStation, baselines)
		if !valid {
			continue
		}
		var xKey, sKey string
		if groupMode == 0 {
			if isFiniteFloat(r.Current) {
				xKey = formatCurrentKey(*r.Current)
			} else {
				xKey = "cur:nil"
			}
			sKey = r.Category
			if sKey == "" {
				sKey = "全部"
			}
		} else {
			xKey = r.Category
			if xKey == "" {
				xKey = "全部"
			}
			if !activeCompare.IsAll {
				sKey = "all"
			} else if isFiniteFloat(r.Current) {
				sKey = formatCurrentKey(*r.Current)
			} else {
				sKey = "cur:nil"
			}
		}
		xi, xOK := xIdxMap[xKey]
		si, sOK := seriesIdxMap[sKey]
		if !xOK || !sOK {
			continue
		}
		buckets[si][xi] = append(buckets[si][xi], boxPoint{Entry: r.Entry, Value: val})
		validRows = append(validRows, r)
		summary.ValidPoints++
	}

	series := make([]chart.Series, len(seriesIDs))
	boxes := make([][]boxStats, len(seriesIDs))
	for si, sid := range seriesIDs {
		medians := make([]float64, len(xLabels))
		rowBoxes := make([]boxStats, len(xLabels))
		for xi := range xLabels {
			b := computeBoxStats(buckets[si][xi])
			rowBoxes[xi] = b
			if b.Valid {
				medians[xi] = b.Median
			} else {
				medians[xi] = math.NaN()
			}
		}
		boxes[si] = rowBoxes
		series[si] = chart.Series{
			ID:     sid,
			Label:  seriesLabels[sid],
			Values: medians,
			Color:  chartSeriesColor(dark, si),
		}
	}

	summary.TotalSeries = len(series)
	summary.ShownSeries = len(series)
	summary.TotalCategories = distinctEntryCount(validRows)
	summary.ShownCategories = summary.TotalCategories

	return chartPayload{
		Data:           chart.Data{Labels: xLabels, Series: series},
		Summary:        summary,
		GlobalInsights: computeBoxPlotInsights(validRows, metric, baseStation, baselines, nil, groupMode),
		IsBox:          true,
		Boxes:          boxes,
	}
}

func shortSeriesLabels(data chart.Data, visibleSeries []int) map[string]string {
	out := make(map[string]string, len(visibleSeries))
	if len(visibleSeries) == 0 {
		return out
	}
	var prefixes, suffixes []string
	allSplit := true
	for _, idx := range visibleSeries {
		lbl := data.Series[idx].Label
		parts := strings.SplitN(lbl, " · ", 2)
		if len(parts) != 2 {
			allSplit = false
			break
		}
		prefixes = append(prefixes, parts[0])
		suffixes = append(suffixes, parts[1])
	}
	sameSuffix := allSplit && len(suffixes) > 0
	samePrefix := allSplit && len(prefixes) > 0
	if allSplit {
		for i := 1; i < len(visibleSeries); i++ {
			if suffixes[i] != suffixes[0] {
				sameSuffix = false
			}
			if prefixes[i] != prefixes[0] {
				samePrefix = false
			}
		}
	}
	for k, idx := range visibleSeries {
		s := data.Series[idx]
		switch {
		case allSplit && sameSuffix && !samePrefix:
			out[s.ID] = prefixes[k]
		case allSplit && samePrefix && !sameSuffix:
			out[s.ID] = suffixes[k]
		default:
			out[s.ID] = s.Label
		}
	}
	return out
}

func formatPointOwner(shortLabel, categoryLabel string, seriesCount int, curveMode bool) string {
	if curveMode {
		if seriesCount <= 1 {
			return categoryLabel
		}
		return shortLabel + " · " + categoryLabel
	}
	if seriesCount <= 1 {
		return categoryLabel
	}
	return categoryLabel + " · " + shortLabel
}

func computeChartInsights(payload chartPayload, state chart.State, metric metricDef, curveMode bool) chartInsights {
	var vis []int
	for j, s := range payload.Data.Series {
		if !state.Hidden[s.ID] {
			vis = append(vis, j)
		}
	}
	return computeChartInsightsFromSeries(payload.Data, payload.Points, vis, metric, curveMode)
}

func computeChartInsightsFromSeries(data chart.Data, points map[string][]chartPointMeta, vis []int, metric metricDef, curveMode bool) chartInsights {
	short := shortSeriesLabels(data, vis)
	ins := chartInsights{
		MinVal: math.Inf(1),
		MaxVal: math.Inf(-1),
	}
	bestAbsMag := math.Inf(1)
	worstAbsMag := -1.0
	var sum, baseSum float64
	var baseCount int
	for _, j := range vis {
		s := data.Series[j]
		metas := points[s.ID]
		sShort := short[s.ID]
		if sShort == "" {
			sShort = s.Label
		}
		for i, cat := range data.Labels {
			if i >= len(s.Values) {
				break
			}
			v := s.Values[i]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			owner := formatPointOwner(sShort, cat, len(vis), curveMode)
			ins.Count++
			sum += v
			if v < ins.MinVal {
				ins.MinVal = v
				ins.MinOwner = owner
			}
			if v > ins.MaxVal {
				ins.MaxVal = v
				ins.MaxOwner = owner
			}
			absV := math.Abs(v)
			if absV < bestAbsMag {
				bestAbsMag = absV
				ins.BestAbsVal = v
				ins.BestAbsOwner = owner
			}
			if absV > worstAbsMag {
				worstAbsMag = absV
				ins.WorstAbsVal = v
				ins.WorstAbsOwner = owner
				ins.HasMaxAbsPct = false
				if i < len(metas) && metas[i].Baseline != nil {
					if bv := metric.Extract(*metas[i].Baseline); isFiniteFloat(bv) && *bv != 0 {
						ins.MaxAbsPct = v / math.Abs(*bv) * 100
						ins.HasMaxAbsPct = true
					}
				}
			}
			if i < len(metas) && metas[i].Baseline != nil {
				if bv := metric.Extract(*metas[i].Baseline); isFiniteFloat(bv) {
					baseSum += *bv
					baseCount++
				}
			}
		}
	}
	if ins.Count == 0 {
		return chartInsights{}
	}
	ins.Mean = sum / float64(ins.Count)
	ins.Range = ins.MaxVal - ins.MinVal
	if baseCount > 0 && baseSum != 0 {
		ins.DeltaMeanPct = sum / math.Abs(baseSum) * 100
		ins.HasMeanPct = true
	}
	return ins
}

func siblingContextText(r *record, metric metricDef, base *record) string {
	if r == nil {
		return ""
	}
	if base != nil {
		cur, bv := metric.Extract(*r), metric.Extract(*base)
		if isFiniteFloat(cur) && isFiniteFloat(bv) {
			curStr := formatMetricNumber(*cur, metric.Unit, false)
			baseStr := formatMetricNumber(*bv, metric.Unit, false)
			if *bv != 0 {
				pct := (*cur - *bv) / math.Abs(*bv) * 100
				return fmt.Sprintf("现值 %s · 基准 %s (%s)", curStr, baseStr, formatMetricNumber(pct, "%", true))
			}
			return fmt.Sprintf("现值 %s · 基准 %s", curStr, baseStr)
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
	if payload.IsBox {
		renderBoxTooltipDetail(c, sel, metric, payload)
		return
	}
	t := c.Theme()
	header := sel.Label
	if payload.Summary.BaseStation != "" {
		header = fmt.Sprintf("%s · 较「%s」", sel.Label, baseStationDisplayName(payload.Summary.BaseStation))
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

func renderBoxTooltipDetail(c *ui.Context, sel chart.Selection, metric metricDef, payload chartPayload) {
	t := c.Theme()
	header := sel.Label
	if payload.Summary.BaseStation != "" {
		header = fmt.Sprintf("%s · 较「%s」", sel.Label, baseStationDisplayName(payload.Summary.BaseStation))
	}
	components.Text(c, header).Bold()
	signed := payload.Summary.BaseStation != "" || metric.ID == "shift"
	seriesByID := make(map[string]int, len(payload.Data.Series))
	for i, s := range payload.Data.Series {
		seriesByID[s.ID] = i
	}
	for _, v := range sel.Values {
		si, ok := seriesByID[v.ID]
		if !ok || si >= len(payload.Boxes) || sel.Index < 0 || sel.Index >= len(payload.Boxes[si]) {
			continue
		}
		b := payload.Boxes[si][sel.Index]
		if !b.Valid {
			continue
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(3).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(t.Space(1.5)).Children(func() {
				ui.Box(c).Size(8, 8).Shrink(0).Radius(3).Background(v.Color)
				components.Text(c, v.Label).TextColor(t.TextMuted).MinWidth(0).Shrink(1).NoWrap().Ellipsis("…")
				ui.Box(c).Grow(1)
				components.Text(c, fmt.Sprintf("n=%d · 中位 %s", b.Count, formatMetricNumber(b.Median, metric.Unit, signed))).Bold().FontFeatures("tnum").NoWrap()
			})
			sub := fmt.Sprintf("Q1 %s · Q3 %s · 均值 %s",
				formatMetricNumber(b.Q1, "", signed),
				formatMetricNumber(b.Q3, "", signed),
				formatMetricNumber(b.Mean, metric.Unit, signed))
			components.Text(c, sub).FontSize(max(10, t.FontSize-2)).FontFeatures("tnum").TextColor(t.TextMuted).Padding(0, 0, 0, 14).SingleLine()
			ext := fmt.Sprintf("最大 %s (%s) · 最小 %s (%s)",
				formatMetricNumber(b.Max, "", signed), b.MaxEntry,
				formatMetricNumber(b.Min, metric.Unit, signed), b.MinEntry)
			components.Text(c, ext).FontSize(max(10, t.FontSize-2)).FontFeatures("tnum").TextColor(t.TextMuted).Padding(0, 0, 0, 14).SingleLine()
			if len(b.Outliers) > 0 {
				var outParts []string
				limit := min(3, len(b.Outliers))
				for i := 0; i < limit; i++ {
					op := b.Outliers[i]
					outParts = append(outParts, op.Entry+" "+formatMetricNumber(op.Value, metric.Unit, signed))
				}
				outText := fmt.Sprintf("异常点(%d): %s", len(b.Outliers), strings.Join(outParts, "、"))
				if len(b.Outliers) > limit {
					outText += fmt.Sprintf(" 等 %d 个", len(b.Outliers))
				}
				components.Text(c, outText).FontSize(max(10, t.FontSize-2)).FontFeatures("tnum").TextColor(t.Warning).Padding(0, 0, 0, 14).SingleLine()
			}
		})
	}
}

func drawDashedHLine(p *ui.Painter, r ui.Rect, yFrac float32, color ui.Color) {
	if yFrac < 0 || yFrac > 1 {
		return
	}
	y := base.Y(r, yFrac)
	const dash, gap = float32(5), float32(4)
	for x := r.X; x < r.X+r.W; x += dash + gap {
		x2 := min(r.X+r.W, x+dash)
		p.Line(x, y, x2, y, 1, color)
	}
}

func toggleSoloSeries(state *chart.State, series []chart.Series, targetID string) {
	if state.Hidden == nil {
		state.Hidden = make(map[string]bool)
	}
	otherVisible := false
	for _, s := range series {
		if s.ID != targetID && !state.Hidden[s.ID] {
			otherVisible = true
			break
		}
	}
	if otherVisible {
		for _, s := range series {
			state.Hidden[s.ID] = s.ID != targetID
		}
	} else {
		clear(state.Hidden)
	}
}

type plotPrep struct {
	visibleSeries []int
	values        [][]float64
	lo, hi, mean  float64
	validCount    int
	valid         bool
}

func preparePlotSeries(data chart.Data, hidden map[string]bool) plotPrep {
	p := plotPrep{lo: math.Inf(1), hi: math.Inf(-1)}
	var sum float64
	for j, s := range data.Series {
		if hidden[s.ID] {
			continue
		}
		row := make([]float64, len(data.Labels))
		for i := range row {
			if i < len(s.Values) && !math.IsNaN(s.Values[i]) && !math.IsInf(s.Values[i], 0) {
				v := s.Values[i]
				row[i] = v
				p.lo = min(p.lo, v)
				p.hi = max(p.hi, v)
				sum += v
				p.validCount++
			} else {
				row[i] = math.NaN()
			}
		}
		p.visibleSeries = append(p.visibleSeries, j)
		p.values = append(p.values, row)
	}
	if p.validCount > 0 {
		p.mean = sum / float64(p.validCount)
		p.valid = true
	}
	return p
}

func preparePlotBoxes(payload chartPayload, hidden map[string]bool) plotPrep {
	p := plotPrep{lo: math.Inf(1), hi: math.Inf(-1)}
	var sum float64
	for j, s := range payload.Data.Series {
		if hidden[s.ID] {
			continue
		}
		row := make([]float64, len(payload.Data.Labels))
		for i := range row {
			if j < len(payload.Boxes) && i < len(payload.Boxes[j]) && payload.Boxes[j][i].Valid {
				b := payload.Boxes[j][i]
				row[i] = b.Median
				p.lo = min(p.lo, b.Min)
				p.hi = max(p.hi, b.Max)
				sum += b.Mean * float64(b.Count)
				p.validCount += b.Count
			} else {
				row[i] = math.NaN()
			}
		}
		p.visibleSeries = append(p.visibleSeries, j)
		p.values = append(p.values, row)
	}
	if p.validCount > 0 {
		p.mean = sum / float64(p.validCount)
		p.valid = true
	}
	return p
}

func computeBoxInsightsFromPrep(payload chartPayload, hidden map[string]bool) chartInsights {
	ins := chartInsights{MinVal: math.Inf(1), MaxVal: math.Inf(-1)}
	bestAbsMag := math.Inf(1)
	worstAbsMag := -1.0
	var sum float64
	for j, s := range payload.Data.Series {
		if hidden[s.ID] || j >= len(payload.Boxes) {
			continue
		}
		for i, cat := range payload.Data.Labels {
			if i >= len(payload.Boxes[j]) || !payload.Boxes[j][i].Valid {
				continue
			}
			b := payload.Boxes[j][i]
			ins.Count += b.Count
			sum += b.Mean * float64(b.Count)
			if b.Min < ins.MinVal {
				ins.MinVal = b.Min
				ins.MinOwner = b.MinEntry + " · " + cat
			}
			if b.Max > ins.MaxVal {
				ins.MaxVal = b.Max
				ins.MaxOwner = b.MaxEntry + " · " + cat
			}
			for _, candidate := range []struct {
				val   float64
				entry string
			}{{b.Min, b.MinEntry}, {b.Max, b.MaxEntry}} {
				absV := math.Abs(candidate.val)
				if absV < bestAbsMag {
					bestAbsMag = absV
					ins.BestAbsVal = candidate.val
					ins.BestAbsOwner = candidate.entry + " · " + cat
				}
				if absV > worstAbsMag {
					worstAbsMag = absV
					ins.WorstAbsVal = candidate.val
					ins.WorstAbsOwner = candidate.entry + " · " + cat
				}
			}
		}
	}
	if ins.Count == 0 {
		return chartInsights{}
	}
	ins.Mean = sum / float64(ins.Count)
	ins.Range = ins.MaxVal - ins.MinVal
	return ins
}

func drawBoxSeries(p *ui.Painter, r ui.Rect, boxes []boxStats, k, seriesCount int, xBand base.Band, yScale base.Linear, shade, textColor, surface ui.Color, isFocused, isDimmed bool) {
	if seriesCount <= 0 {
		return
	}
	borderColor := shade.Mix(textColor, 0.38)
	fillAlpha := float32(0.28)
	stemColor := borderColor.Alpha(0.9)
	if isFocused {
		fillAlpha = 0.40
		stemColor = borderColor
	} else if isDimmed {
		fillAlpha = 0.08
		stemColor = borderColor.Alpha(0.3)
		borderColor = borderColor.Alpha(0.35)
	}
	for i, b := range boxes {
		if !b.Valid {
			continue
		}
		band := xBand.Band(i)
		slotW := (band.Hi - band.Lo) / float32(seriesCount)
		slotGap := float32(0)
		if seriesCount > 1 {
			slotGap = slotW * 0.16
		}
		slotLo := band.Lo + float32(k)*slotW + slotGap/2
		slotHi := band.Lo + float32(k+1)*slotW - slotGap/2
		pxLo := base.X(r, slotLo)
		pxHi := base.X(r, slotHi)
		boxW := min(float32(64), max(float32(12), (pxHi-pxLo)*0.72))
		cx := (pxLo + pxHi) / 2
		bx0 := cx - boxW/2
		bx1 := cx + boxW/2
		capW := max(float32(6), boxW*0.44)

		yMinW := base.Y(r, yScale.At(b.WhiskerLow))
		yQ1 := base.Y(r, yScale.At(b.Q1))
		yMed := base.Y(r, yScale.At(b.Median))
		yQ3 := base.Y(r, yScale.At(b.Q3))
		yMaxW := base.Y(r, yScale.At(b.WhiskerHigh))
		yMean := base.Y(r, yScale.At(b.Mean))

		if yQ3 > yMaxW {
			p.Line(cx, yMaxW, cx, yQ3, 1.2, stemColor)
		}
		p.Line(cx-capW/2, yMaxW, cx+capW/2, yMaxW, 1.2, stemColor)
		if yMinW > yQ1 {
			p.Line(cx, yQ1, cx, yMinW, 1.2, stemColor)
		}
		p.Line(cx-capW/2, yMinW, cx+capW/2, yMinW, 1.2, stemColor)

		boxH := yQ1 - yQ3
		if boxH < 4 {
			midRect := ui.Rect{X: bx0, Y: yMed - 1.5, W: boxW, H: 3}
			p.Fill(midRect, shade.Alpha(max(fillAlpha, 0.4)), 0)
			p.Stroke(midRect, borderColor, 0, 1.2)
		} else {
			boxRect := ui.Rect{X: bx0, Y: yQ3, W: boxW, H: boxH}
			p.Fill(boxRect, shade.Alpha(fillAlpha), 0)
			p.Stroke(boxRect, borderColor, 0, 1.2)
			p.Line(bx0+0.5, yMed, bx1-0.5, yMed, 2.0, borderColor)
			if boxH >= 12 {
				const cross = float32(2.5)
				p.Line(cx-cross, yMean, cx+cross, yMean, 1.2, borderColor)
				p.Line(cx, yMean-cross, cx, yMean+cross, 1.2, borderColor)
			}
		}

		// Keep outliers centered on the whisker stem; jitter horizontally only when vertical pixels collide.
		var placedY []float32
		for _, op := range b.Outliers {
			oy := base.Y(r, yScale.At(op.Value))
			nearCount := 0
			for _, py := range placedY {
				if math.Abs(float64(oy-py)) < 5.5 {
					nearCount++
				}
			}
			placedY = append(placedY, oy)
			ox := cx
			if nearCount > 0 {
				sign := float32(1)
				if nearCount%2 == 1 {
					sign = -1
				}
				ox += sign * float32((nearCount+1)/2) * 4.5
			}
			var oPath ui.Path
			oPath.Circle(ox, oy, 2.6)
			p.FillPath(&oPath, surface)
			p.StrokePath(&oPath, 1.2, borderColor)
		}
	}
}

func computeBoxYScale(p plotPrep, deltaMode, showMeanRule bool, refMean float64) (base.Linear, bool) {
	if !p.valid {
		return base.Linear{}, false
	}
	yLo, yHi := p.lo, p.hi
	if deltaMode {
		yLo = min(yLo, 0)
		yHi = max(yHi, 0)
	}
	if showMeanRule {
		yLo = min(yLo, refMean)
		yHi = max(yHi, refMean)
	}
	span := yHi - yLo
	if span <= 0 {
		pad := max(0.5, math.Abs(yLo)*0.05)
		yLo -= pad
		yHi += pad
	} else {
		yLo -= span * 0.06
		yHi += span * 0.06
	}
	yScale := base.NewLinear(yLo, yHi).Nice(5)
	if math.IsNaN(yScale.Lo) || math.IsNaN(yScale.Hi) || math.IsInf(yScale.Hi-yScale.Lo, 0) {
		return base.Linear{}, false
	}
	return yScale, true
}

func computePlotYScale(p plotPrep, metric metricDef, isBar, deltaMode, useMeanBaseline, includeMean bool, refMean float64) (base.Linear, bool) {
	if !p.valid {
		return base.Linear{}, false
	}
	yLo, yHi := p.lo, p.hi
	switch {
	case useMeanBaseline:
		yLo = min(yLo, refMean)
		yHi = max(yHi, refMean)
		pad := max(0.5, math.Abs(yLo)*0.001)
		if yLo != yHi {
			pad = (yHi - yLo) * 0.16
		}
		yLo -= pad
		yHi += pad
	case !isBar && metric.TightScale && !deltaMode:
		pad := max(0.5, math.Abs(yLo)*0.001)
		if yLo != yHi {
			pad = (yHi - yLo) * 0.12
		}
		yLo -= pad
		yHi += pad
	default:
		yLo = min(yLo, 0)
		yHi = max(yHi, 0)
		if includeMean {
			yLo = min(yLo, refMean)
			yHi = max(yHi, refMean)
		}
		if yLo == yHi {
			yHi = yLo + 1
		} else {
			span := yHi - yLo
			if yHi > 0 {
				yHi += span * 0.10
			}
			if yLo < 0 {
				yLo -= span * 0.10
			}
		}
	}
	yScale := base.NewLinear(yLo, yHi).Nice(4)
	if math.IsNaN(yScale.Lo) || math.IsNaN(yScale.Hi) || math.IsInf(yScale.Hi-yScale.Lo, 0) {
		return base.Linear{}, false
	}
	return yScale, true
}

func drawBarSeries(c *ui.Context, ser chart.Series, vals []float64, k, seriesCount int, xBand base.Band, yScale base.Linear, barBaseline float64, shade ui.Color, isFocused, isDimmed, showBarLabels, signedAxis bool) {
	t := c.Theme()
	edgeColor := shade.Mix(t.Text, 0.35)
	fillColor := shade.Alpha(0.84)
	if isFocused {
		fillColor = shade.Alpha(0.95)
	} else if isDimmed {
		fillColor = shade.Alpha(0.2)
		edgeColor = edgeColor.Alpha(0.25)
	}
	ui.Box(c).Key(ser.ID).Absolute().Left(0).Right(0).Top(0).Bottom(0).Children(func() {
		for i, v := range vals {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			band := xBand.Band(i)
			w := (band.Hi - band.Lo) / float32(seriesCount)
			gap := float32(0)
			if seriesCount > 1 {
				gap = w * 0.14
			}
			barBand := base.Span{Lo: band.Lo + float32(k)*w + gap/2, Hi: band.Lo + float32(k+1)*w - gap/2}
			if math.Abs(float64(yScale.At(v)-yScale.At(barBaseline))) < 0.004 {
				ui.Box(c).Absolute().
					LeftPercent(100*barBand.Lo).WidthPercent(100*(barBand.Hi-barBand.Lo)).
					TopPercent(100*(1-yScale.At(barBaseline))).Height(2).Margin(-1, 0, 0, 0).
					Background(edgeColor)
			} else {
				span := yScale.Span(barBaseline, v)
				base.Rect(c, barBand, span).Background(fillColor).Border(1, edgeColor)
			}
			if showBarLabels && !isDimmed {
				vAlign := ui.End
				margin := [4]float32{0, 0, 3, 0}
				if v < barBaseline {
					vAlign = ui.Start
					margin = [4]float32{3, 0, 0, 0}
				}
				valLabel := formatMetricNumber(v, "", signedAxis)
				base.Anchor(c, barBand.Mid(), yScale.At(v), ui.Center, vAlign, func() {
					lbl := components.Text(c, valLabel).
						FontSize(max(10, t.FontSize-3)).FontFeatures("tnum").
						TextColor(t.Text.Alpha(0.85)).NoWrap().
						Margin(margin[0], margin[1], margin[2], margin[3])
					if i == 0 || isFocused {
						lbl.TextColor(t.Text).FontWeight(600)
					}
				})
			}
		}
	})
}

func lttbPointBudget(plotW float32) int {
	budget := int(plotW / 3)
	if budget < 80 {
		return 80
	}
	if budget > 320 {
		return 320
	}
	return budget
}

func lttbSegment(xs []float32, vals []float64, threshold int) ([]float32, []float64) {
	n := len(xs)
	if threshold <= 0 || n <= threshold {
		return xs, vals
	}
	if threshold <= 2 {
		if threshold == 1 {
			return []float32{xs[0]}, []float64{vals[0]}
		}
		return []float32{xs[0], xs[n-1]}, []float64{vals[0], vals[n-1]}
	}

	outX := make([]float32, 0, threshold)
	outY := make([]float64, 0, threshold)
	outX = append(outX, xs[0])
	outY = append(outY, vals[0])

	every := float64(n-2) / float64(threshold-2)
	a := 0
	for i := 0; i < threshold-2; i++ {
		avgStart := int(math.Floor(float64(i+1)*every)) + 1
		avgEnd := int(math.Floor(float64(i+2)*every)) + 1
		if avgStart >= n {
			avgStart = n - 1
		}
		if avgEnd > n {
			avgEnd = n
		}
		avgX, avgY := 0.0, 0.0
		avgCount := avgEnd - avgStart
		if avgCount > 0 {
			for j := avgStart; j < avgEnd; j++ {
				avgX += float64(xs[j])
				avgY += vals[j]
			}
			avgX /= float64(avgCount)
			avgY /= float64(avgCount)
		} else {
			avgX = float64(xs[n-1])
			avgY = vals[n-1]
		}

		rangeStart := int(math.Floor(float64(i)*every)) + 1
		rangeEnd := min(n-1, avgStart)
		if rangeStart >= rangeEnd {
			rangeStart = max(1, rangeEnd-1)
		}
		ax := float64(xs[a])
		ay := vals[a]
		maxArea := -1.0
		maxIdx := rangeStart
		for j := rangeStart; j < rangeEnd; j++ {
			area := math.Abs((ax-avgX)*(vals[j]-ay) - (ax-float64(xs[j]))*(avgY-ay))
			if area > maxArea {
				maxArea = area
				maxIdx = j
			}
		}
		outX = append(outX, xs[maxIdx])
		outY = append(outY, vals[maxIdx])
		a = maxIdx
	}

	outX = append(outX, xs[n-1])
	outY = append(outY, vals[n-1])
	return outX, outY
}

func downsampleSeriesLTTB(xs []float32, vals []float64, threshold int) ([]float32, []float64) {
	n := min(len(xs), len(vals))
	if threshold < 3 || n <= threshold {
		return xs[:n], vals[:n]
	}
	type span struct{ start, end int }
	var runs []span
	totalFinite := 0
	inRun := false
	start := 0
	for i := 0; i < n; i++ {
		finite := !math.IsNaN(vals[i]) && !math.IsInf(vals[i], 0)
		if finite && !inRun {
			start = i
			inRun = true
		} else if !finite && inRun {
			runs = append(runs, span{start: start, end: i})
			totalFinite += i - start
			inRun = false
		}
	}
	if inRun {
		runs = append(runs, span{start: start, end: n})
		totalFinite += n - start
	}
	if totalFinite <= threshold || len(runs) == 0 {
		return xs[:n], vals[:n]
	}
	if len(runs) == 1 && runs[0].start == 0 && runs[0].end == n {
		return lttbSegment(xs[:n], vals[:n], threshold)
	}
	outX := make([]float32, 0, min(n, threshold+len(runs)))
	outY := make([]float64, 0, min(n, threshold+len(runs)))
	for idx, seg := range runs {
		if idx > 0 {
			gapIdx := max(0, seg.start-1)
			outX = append(outX, xs[gapIdx])
			outY = append(outY, math.NaN())
		}
		segLen := seg.end - seg.start
		segBudget := max(min(segLen, 3), int(math.Round(float64(threshold)*float64(segLen)/float64(totalFinite))))
		sx, sy := lttbSegment(xs[seg.start:seg.end], vals[seg.start:seg.end], segBudget)
		outX = append(outX, sx...)
		outY = append(outY, sy...)
	}
	return outX, outY
}

func pickEvenIndices(n, limit int) []int {
	if n <= limit || limit <= 0 {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	}
	if limit == 1 {
		return []int{0}
	}
	out := make([]int, limit)
	for i := 0; i < limit; i++ {
		out[i] = i * (n - 1) / (limit - 1)
	}
	return out
}

func drawLineAreaSeries(c *ui.Context, ser chart.Series, shape int, vals []float64, xs []float32, plotXs []float32, plotVals []float64, yScale base.Linear, shade, surface ui.Color, areaFill, deltaMode, isFocused, isDimmed bool, selected int) {
	ys := make([]float32, len(plotVals))
	baseline := make([]float32, len(plotVals))
	baseValFrac := yScale.At(yScale.Lo)
	if deltaMode && yScale.Lo <= 0 && yScale.Hi >= 0 {
		baseValFrac = yScale.At(0)
	}
	for i, v := range plotVals {
		ys[i] = yScale.At(v)
		baseline[i] = baseValFrac
	}
	strokeW := float32(1.8)
	if isFocused {
		strokeW = 2.6
	} else if isDimmed {
		strokeW = 1.2
	}
	ptSize := float32(6.5)
	if isFocused {
		ptSize = 7.5
	}
	ui.Box(c).Key(ser.ID).Absolute().Left(0).Right(0).Top(0).Bottom(0).Draw(func(p *ui.Painter, r ui.Rect) {
		if areaFill {
			fillA := float32(.14)
			if isFocused {
				fillA = .22
			} else if isDimmed {
				fillA = .04
			}
			p.FillPath(base.Area(r, plotXs, baseline, ys, base.CurveLinear), ser.Color.Alpha(fillA))
		}
		p.StrokePath(base.Line(r, plotXs, ys, base.CurveLinear), strokeW, shade)
		if len(xs) > 32 {
			if selected >= 0 && selected < len(vals) && selected < len(xs) {
				v := vals[selected]
				if !math.IsNaN(v) && !math.IsInf(v, 0) {
					px := base.X(r, xs[selected])
					py := base.Y(r, yScale.At(v))
					if !isDimmed {
						var halo ui.Path
						halo.Circle(px, py, 7)
						p.FillPath(&halo, ser.Color.Alpha(.16))
					}
					drawSeriesMarker(p, px, py, ptSize, shape, shade, surface)
				}
			}
			return
		}
		for i, v := range vals {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			px := base.X(r, xs[i])
			py := base.Y(r, yScale.At(v))
			if i == selected && !isDimmed {
				var halo ui.Path
				halo.Circle(px, py, 7)
				p.FillPath(&halo, ser.Color.Alpha(.16))
			}
			drawSeriesMarker(p, px, py, ptSize, shape, shade, surface)
		}
	})
}

func (a *application) renderUnifiedChart(c *ui.Context, metric metricDef, payload chartPayload, opts chart.Options) {
	t := c.Theme()
	data := payload.Data
	if payload.IsBox && len(data.Series) == 1 {
		data.Series = slices.Clone(data.Series)
		data.Series[0].Color = chartSeriesColor(t.Dark, 1)
	}
	state := &a.chartState

	isBox := payload.IsBox
	var prep plotPrep
	if isBox {
		prep = preparePlotBoxes(payload, state.Hidden)
	} else {
		prep = preparePlotSeries(data, state.Hidden)
	}
	isBar := !isBox && a.chartKind == 2
	areaFill := !isBox && a.chartKind == 1
	deltaMode := payload.Summary.BaseStation != ""
	signed := signedAxis(deltaMode, metric)
	zoomIns := payload.GlobalInsights
	if isBox {
		if len(state.Hidden) > 0 || zoomIns.Count < 2 {
			zoomIns = computeBoxInsightsFromPrep(payload, state.Hidden)
		}
	} else if len(state.Hidden) > 0 || zoomIns.Count < 2 {
		zoomIns = computeChartInsightsFromSeries(data, payload.Points, prep.visibleSeries, metric, a.chartGroup == 0)
	}
	zoomActive := a.resolveZoomDiff(metric, zoomIns, deltaMode)
	useMeanBaseline := isBar && !deltaMode && prep.valid && (metric.TightScale && a.chartGroup != 1 || a.chartGroup == 1 && zoomActive)
	scaleMetric := metric
	if (a.chartGroup == 1 || isBox) && !deltaMode {
		scaleMetric.TightScale = zoomActive
	}
	showMeanRule := a.chartGroup == 1 && prep.validCount >= 2 && !deltaMode && (!isBox || len(data.Labels) >= 2)
	refMean := prep.mean
	meanPrefix := "均值 "
	if showMeanRule && len(state.Hidden) == 0 && (payload.Summary.TotalCategories > payload.Summary.ShownCategories || payload.Summary.TotalSeries > payload.Summary.ShownSeries) && payload.GlobalInsights.Count > 0 {
		refMean = payload.GlobalInsights.Mean
		meanPrefix = "全量均值 "
	}
	barBaseline := 0.0
	if useMeanBaseline {
		barBaseline = refMean
	}

	var yScale base.Linear
	var valid bool
	if isBox {
		yScale, valid = computeBoxYScale(prep, deltaMode, showMeanRule, refMean)
	} else {
		yScale, valid = computePlotYScale(prep, scaleMetric, isBar, deltaMode, useMeanBaseline, showMeanRule, refMean)
	}
	if !valid || state.Index >= len(data.Labels) {
		state.Index = -1
	}

	activeHover := ""
	if a.chartHoverSeries != "" && !state.Hidden[a.chartHoverSeries] {
		for _, j := range prep.visibleSeries {
			if data.Series[j].ID == a.chartHoverSeries {
				activeHover = a.chartHoverSeries
				break
			}
		}
	}

	buildSelection := func(i int) chart.Selection {
		s := chart.Selection{Index: i}
		if i < 0 || i >= len(data.Labels) {
			return s
		}
		s.Label = data.Labels[i]
		for _, j := range prep.visibleSeries {
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

	insights := payload.GlobalInsights
	if isBox {
		if len(state.Hidden) > 0 || insights.Count == 0 {
			insights = computeBoxInsightsFromPrep(payload, state.Hidden)
		}
	} else if len(state.Hidden) > 0 || insights.Count == 0 {
		insights = computeChartInsights(payload, *state, metric, a.chartGroup == 0)
	}

	contentWidth, _ := c.Size()
	contentWidth -= a.sidebarWidth + 48
	sidebarWidth := float32(224)
	if contentWidth < 800 {
		sidebarWidth = 188
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
		if len(data.Series) > 0 {
			stats := chartStats(insights, metric, payload.Summary)
			if isBox {
				stats = chartBoxStats(payload, state, metric, a.chartGroup)
			}
			if !valid {
				// The first click of a double-click can hide the last visible
				// series. Keep the legend's position stable until its second press.
				stats = chartStats(payload.GlobalInsights, metric, payload.Summary)
				if isBox {
					stats = chartBoxStats(payload, &chart.State{}, metric, a.chartGroup)
				}
				for i := range stats {
					stats[i].value, stats[i].detail = "—", "全部序列已隐藏"
				}
			}
			renderChartStats(c, stats, contentWidth)
			ui.Box(c).FillWidth().Height(1).Background(t.Border)
		}
		components.Text(c, chartTitle(metric, payload, a.chartGroup)).FontSize(15).FontWeight(600)
		ui.Row(c).FillWidth().MinWidth(0).Gap(16).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
				if !valid {
					ui.Column(c).FillWidth().Height(opts.PlotHeight).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
						if opts.Empty != nil {
							opts.Empty(c)
						}
					})
				} else {
					yAxisTitle := metric.Label
					if deltaMode {
						yAxisTitle = "Δ " + metric.Label
					}
					xAxisTitle := "电流 (A)"
					if isBox && a.chartGroup == 1 {
						xAxisTitle = "测试站别"
					} else if a.chartGroup == 1 {
						xAxisTitle = "编号"
					}

					chartRow := ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Stretch).Gap(2)
					width := chartRow.LayoutBounds().W
					if width <= 0 {
						width = 480
					}
					plotW := max(120, width-68)

					barIndices := []int(nil)
					barLabels := data.Labels
					if isBar && len(data.Labels) > maxChartCategories {
						barIndices = pickEvenIndices(len(data.Labels), maxChartCategories)
						barLabels = make([]string, len(barIndices))
						for i, rawIdx := range barIndices {
							barLabels[i] = data.Labels[rawIdx]
						}
					}

					xBand := base.NewBand(barLabels...).Padding(.3, .3)
					xs := make([]float32, len(data.Labels))
					if !isBar && !isBox && payload.HasLinearX && len(payload.XValues) == len(data.Labels) && len(data.Labels) >= 2 {
						xMin, xMax := chartCurrentScale(payload.XValues)
						span := xMax - xMin
						leftFrac, rightFrac := float32(0), float32(0.96)
						for i, xv := range payload.XValues {
							xs[i] = leftFrac + float32((xv-xMin)/span)*(rightFrac-leftFrac)
						}
					} else if isBar && len(barIndices) > 0 {
						xs = make([]float32, len(barLabels))
						for i := range xs {
							xs[i] = xBand.At(i)
						}
					} else {
						for i := range xs {
							xs[i] = xBand.At(i)
						}
					}

					activeLabels := data.Labels
					if isBar && len(barIndices) > 0 {
						activeLabels = barLabels
					}
					tickLabels := make([]string, len(activeLabels))
					for i, lbl := range activeLabels {
						if a.chartGroup == 0 {
							tickLabels[i] = strings.TrimSuffix(lbl, " A")
						} else {
							tickLabels[i] = lbl
						}
					}

					yScale.Format = func(v float64) string {
						return formatMetricNumber(v, "", signed)
					}
					axisSize := max(10.5, t.FontSize-2)
					spineColor := t.Text.Alpha(0.65)
					if t.Dark {
						spineColor = t.Text.Alpha(0.55)
					}

					chartRow.Children(func() {
						ui.Box(c).Width(20).Height(opts.PlotHeight).Label(yAxisTitle).Draw(func(p *ui.Painter, r ui.Rect) {
							plotMidY := r.Y + max(24, (r.H-24)/2)
							cx := r.X + r.W/2
							span := ui.Span{Text: yAxisTitle, Size: 11.5, Weight: 600, Color: t.Text.Alpha(0.9)}
							w, h := p.MeasureText(0, span)
							p.Transform(ui.Rotation(-90), cx, plotMidY, func() {
								p.RichText(cx-w/2, plotMidY-h/2, w+6, span)
							})
						})

						ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
							holder := ui.Column(c).FillWidth().MinWidth(0).Height(opts.PlotHeight).Padding(6, 8, 0, 2)

							tickWidth := t.Space(11)
							sampleIdx := pickEvenIndices(len(tickLabels), 16)
							longestIdx, longestLen := 0, -1
							for i, label := range tickLabels {
								if len(label) > longestLen {
									longestLen = len(label)
									longestIdx = i
								}
							}
							for _, idx := range append(sampleIdx, longestIdx) {
								if idx < 0 || idx >= len(tickLabels) {
									continue
								}
								w, _ := c.MeasureText(0, ui.Span{Text: tickLabels[idx], Size: axisSize})
								tickWidth = max(tickWidth, min(t.Space(21), w+t.Space(3)))
							}
							var xTicks []base.Tick
							lastTickPx := float32(-1e9)
							for i := range xs {
								px := xs[i] * plotW
								if i == 0 || px-lastTickPx >= tickWidth {
									xTicks = append(xTicks, base.Tick{At: xs[i], Label: tickLabels[i]})
									lastTickPx = px
								}
							}
							if !isBar && !isBox && payload.HasLinearX && len(payload.XValues) >= 2 {
								lo, hi := chartCurrentScale(payload.XValues)
								xTicks = base.NewLinear(lo, hi).Ticks(5)
								for i := range xTicks {
									xTicks[i].At *= 0.96
								}
							}
							axes := base.Axes{
								Left:   yScale.Ticks(5),
								Bottom: xTicks,
								Label: func(side base.Side, tick base.Tick) {
									lbl := components.Text(c, tick.Label).NoWrap().FontSize(axisSize).FontFeatures("tnum").TextColor(t.Text.Alpha(0.85))
									if side == base.Bottom {
										lbl.MaxWidth(max(24, tickWidth-8)).Ellipsis("…")
									}
								},
							}

							holder.Children(func() {
								base.Plot(c, axes, func(area ui.Element) {
									area.Label(opts.Label).Description(opts.Description)
									if !area.IsDisabled() {
										area.Cursor(ui.CursorCrosshair)
										positions := xs
										if bw, rw := area.Bounds().W, area.LayoutBounds().W; bw > 0 && rw > 0 && bw != rw {
											positions = make([]float32, len(xs))
											for i, px := range xs {
												positions[i] = px * rw / bw
											}
										}
										base.Select(c, area, &state.Index, positions, func(i int) string {
											rawIdx := i
											if len(barIndices) > 0 && i >= 0 && i < len(barIndices) {
												rawIdx = barIndices[i]
											}
											return describeSelection(buildSelection(rawIdx))
										})
									}

									areaW := area.LayoutBounds().W
									if areaW <= 0 {
										areaW = plotW
									}

									selected := state.Index
									if selected >= len(xs) {
										selected = -1
									}
									rawSelected := selected
									if len(barIndices) > 0 && selected >= 0 && selected < len(barIndices) {
										rawSelected = barIndices[selected]
									}
									curveBudget := lttbPointBudget(areaW)
									showZeroRule := (deltaMode || metric.ID == "shift") && yScale.Lo <= 0 && yScale.Hi >= 0

									area.Draw(func(painter *ui.Painter, r ui.Rect) {
										for _, tick := range axes.Left {
											y := base.Y(r, tick.At)
											if y > r.Y+1 && y < r.Y+r.H-1 {
												painter.Line(r.X, y, r.X+r.W, y, 0.8, t.Border.Alpha(.30))
											}
										}
										if showZeroRule {
											drawDashedHLine(painter, r, yScale.At(0), t.TextMuted.Alpha(.6))
										}
										if showMeanRule {
											alpha := float32(.55)
											if useMeanBaseline {
												alpha = .75
											}
											drawDashedHLine(painter, r, yScale.At(refMean), t.TextMuted.Alpha(alpha))
										}
										if selected >= 0 && selected < len(xs) {
											w := max(12, r.W/float32(max(1, len(xs)))*.8)
											painter.Fill(ui.Rect{X: base.X(r, xs[selected]) - w/2, Y: r.Y, W: w, H: r.H}, t.Text.Alpha(.025), 0)
										}
										if isBox {
											boxOrder := make([]int, 0, len(prep.visibleSeries))
											hoveredBox := -1
											for k, j := range prep.visibleSeries {
												if activeHover != "" && data.Series[j].ID == activeHover {
													hoveredBox = k
													continue
												}
												boxOrder = append(boxOrder, k)
											}
											if hoveredBox >= 0 {
												boxOrder = append(boxOrder, hoveredBox)
											}
											for _, k := range boxOrder {
												origIdx := prep.visibleSeries[k]
												if origIdx >= len(payload.Boxes) {
													continue
												}
												ser := data.Series[origIdx]
												isFocused := activeHover != "" && ser.ID == activeHover
												isDimmed := activeHover != "" && !isFocused
												shade := ser.Color
												if isDimmed {
													shade = shade.Alpha(.18)
												}
												drawBoxSeries(painter, r, payload.Boxes[origIdx], k, len(prep.visibleSeries), xBand, yScale, shade, t.Text, t.Surface, isFocused, isDimmed)
											}
										}
									})

									area.DrawOver(func(painter *ui.Painter, r ui.Rect) {
										// Quiet axes leave the plot open above and to the right.
										painter.Line(r.X, r.Y, r.X, r.Y+r.H, 1, spineColor)
										painter.Line(r.X, r.Y+r.H, r.X+r.W, r.Y+r.H, 1, spineColor)
										const tickLen = float32(4.5)
										for _, tick := range axes.Left {
											y := base.Y(r, tick.At)
											if y >= r.Y-0.5 && y <= r.Y+r.H+0.5 {
												painter.Line(r.X, y, r.X+tickLen, y, 1.0, spineColor)
											}
										}
										for _, tick := range axes.Bottom {
											x := base.X(r, tick.At)
											if x >= r.X-0.5 && x <= r.X+r.W+0.5 {
												painter.Line(x, r.Y+r.H-tickLen, x, r.Y+r.H, 1.0, spineColor)
											}
										}
									})

									// Draw non-hovered series first so the focused series always paints on top.
									if !isBox {
										drawOrder := make([]int, 0, len(prep.visibleSeries))
										hoveredOrder := -1
										for k, j := range prep.visibleSeries {
											if activeHover != "" && data.Series[j].ID == activeHover {
												hoveredOrder = k
												continue
											}
											drawOrder = append(drawOrder, k)
										}
										if hoveredOrder >= 0 {
											drawOrder = append(drawOrder, hoveredOrder)
										}

										showBarLabels := isBar && len(barLabels)*len(prep.visibleSeries) <= 12
										for _, k := range drawOrder {
											origIdx := prep.visibleSeries[k]
											ser := data.Series[origIdx]
											isFocused := activeHover != "" && ser.ID == activeHover
											isDimmed := activeHover != "" && !isFocused
											shade := ser.Color
											if isDimmed {
												shade = shade.Alpha(.18)
											}
											if isBar {
												barVals := prep.values[k]
												if len(barIndices) > 0 {
													barVals = make([]float64, len(barIndices))
													for bi, rawIdx := range barIndices {
														barVals[bi] = prep.values[k][rawIdx]
													}
												}
												drawBarSeries(c, ser, barVals, k, len(prep.visibleSeries), xBand, yScale, barBaseline, shade, isFocused, isDimmed, showBarLabels, signed)
											} else {
												plotXs, plotVals := a.cachedLTTBSeries(ser.ID, xs, prep.values[k], curveBudget)
												drawLineAreaSeries(c, ser, origIdx, prep.values[k], xs, plotXs, plotVals, yScale, shade, t.Surface, areaFill, deltaMode, isFocused, isDimmed, selected)
											}
										}
									}

									if showMeanRule {
										meanText := meanPrefix + formatMetricNumber(refMean, metric.Unit, false)
										base.Anchor(c, 0.985, yScale.At(refMean), ui.End, ui.End, func() {
											components.Text(c, meanText).
												FontSize(max(10, t.FontSize-3)).FontFeatures("tnum").
												TextColor(t.TextMuted).Background(t.Surface.Alpha(.85)).
												Padding(1, 4).Radius(2).Margin(0, 0, 2, 0).NoWrap()
										})
									} else if showZeroRule && deltaMode {
										base.Anchor(c, 0.985, yScale.At(0), ui.End, ui.End, func() {
											components.Text(c, "基准 0").
												FontSize(max(10, t.FontSize-3)).
												TextColor(t.TextMuted).Background(t.Surface.Alpha(.85)).
												Padding(1, 4).Radius(2).Margin(0, 0, 2, 0).NoWrap()
										})
									}

									if isBox {
										renderBoxAnnotations(c, payload, prep.visibleSeries, xBand, yScale, selected, metric)
									}
									if selected >= 0 && selected < len(xs) {
										base.RuleX(c, xs[selected], 1).Background(t.TextMuted.Alpha(.3))
									}
									renderAdaptiveTip(c, area, opts, buildSelection(rawSelected), func() ui.Element {
										return base.RuleX(c, xs[selected], 0)
									})
								}).Fill().MinWidth(0).Gap(6)
							})
							ui.Row(c).FillWidth().Justify(ui.Center).Padding(0, 8, 2, 26).Children(func() {
								components.Text(c, xAxisTitle).FontSize(11.5).FontWeight(600).TextColor(t.Text.Alpha(0.9)).NoWrap()
							})
						})
					})
				}

			})
			a.chartSidebar(c, data, payload, metric, opts.PlotHeight+40, sidebarWidth)
		})
		if isBox && valid {
			components.Text(c, "局部纵轴 "+formatMetricNumber(yScale.Lo, "", signed)+"–"+formatMetricNumber(yScale.Hi, metric.Unit, signed)).FontSize(11).TextColor(t.TextMuted)
		}
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
				return
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
	summary := payload.Summary
	if summary.ValidPoints == 0 {
		payload.Data = chart.Data{}
	}

	sourceRows, _, stations, hasMeanBaseline := a.cachedSourceSnapshot()
	activeCompare, compareOpts := payload.ActiveCompare, payload.CompareOpts
	if len(compareOpts) > 0 && a.chartCompareCurrent != activeCompare.Key {
		a.chartCompareCurrent = activeCompare.Key
	}

	availWidth, availHeight := c.Size()
	rightWidth := availWidth - a.sidebarWidth - 48
	reserve := float32(440)
	if rightWidth < 760 {
		reserve += 36
	}
	if rightWidth < 560 {
		reserve += 88
	}
	plotHeight := max(float32(220), availHeight-reserve)

	components.ScrollView(c.Key("chart-scroll"), components.ScrollViewOptions{}).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(12).Gap(12).Children(func() {
			ui.Row(c).FillWidth().Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
				components.Text(c, "指标").FontSize(12).TextColor(t.TextMuted)
				sel := components.Select(c.Key("chart-metric"), &a.chartMetric)
				sel.Trigger.Label("分析指标").Cursor(ui.CursorPointer).Children(func() {
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

				if a.chartBaseStation != "" && !isValidBaseStation(a.chartBaseStation, stations, sourceRows) {
					a.chartBaseStation = ""
				}
				if len(stations) >= 2 || hasMeanBaseline {
					components.Text(c, "对比").FontSize(12).TextColor(t.TextMuted)
					baseLabel := baseStationOptionLabel(a.chartBaseStation)
					bs := components.Select(c.Key("chart-base"), &a.chartBaseStation)
					bs.Trigger.Label("对比基准").Cursor(ui.CursorPointer).Children(func() {
						components.Text(c, baseLabel).FontSize(13).SingleLine()
					})
					bs.Popup(func(panel ui.Element) {
						bs.Item("").Children(func() {
							components.Text(c, baseStationOptionLabel("")).FontSize(13).SingleLine()
						})
						if hasMeanBaseline {
							bs.Item(baseMeanSentinel).Children(func() {
								components.Text(c, baseStationOptionLabel(baseMeanSentinel)).FontSize(13).SingleLine()
							})
						}
						if len(stations) >= 2 {
							for _, st := range stations {
								item := bs.Item(st)
								item.Children(func() {
									components.Text(c, baseStationOptionLabel(st)).FontSize(13).SingleLine()
								})
							}
						}
					})
				}

				ui.Spacer(c)
				components.Text(c, "图表").FontSize(12).TextColor(t.TextMuted)
				components.Tabs(c.Key("chart-kind"), &a.chartKind, components.TabsHighlight, "折线", "面积", "柱状", "箱线").List.Label("图表类型")
			})
			ui.Box(c).FillWidth().Height(1).Background(t.Border)
			ui.Row(c).FillWidth().Wrap().Gap(8).AlignItems(ui.Center).Children(func() {

				dimLabels := [2]string{"按电流曲线", "按编号对比"}
				if a.chartKind == 3 {
					dimLabels = [2]string{"按电流分布", "按站别分布"}
				}
				components.Tabs(c.Key("chart-group"), &a.chartGroup, components.TabsSegmented, dimLabels[0], dimLabels[1]).
					List.Label("图表维度")

				if a.chartGroup == 1 {
					if len(compareOpts) > 0 {
						cs := components.Select(c.Key("chart-compare-current"), &a.chartCompareCurrent)
						cs.Trigger.Label("对比电流").Cursor(ui.CursorPointer).Children(func() {
							components.Text(c, "对比: "+activeCompare.Label).FontSize(13).SingleLine()
						})
						cs.Popup(func(panel ui.Element) {
							for _, opt := range compareOpts {
								item := cs.Item(opt.Key)
								item.Children(func() {
									components.Text(c, opt.Label).FontSize(13).SingleLine()
								})
							}
						})
					}
					if a.chartKind != 3 && summary.TotalCategories > maxChartCategories {
						wLabels := [3]string{"最优前 24", "最差后 24", "两头各 12"}
						if !a.chartSortByValue {
							wLabels = [3]string{"前 24 项", "后 24 项", "首尾各 12"}
						}
						wIdx := a.chartCompareWindow
						if wIdx < 0 || wIdx >= len(wLabels) {
							wIdx = 0
						}
						ws := components.Select(c.Key("chart-compare-window"), &a.chartCompareWindow)
						ws.Trigger.Label("对比范围").Cursor(ui.CursorPointer).Children(func() {
							components.Text(c, wLabels[wIdx]).FontSize(13).SingleLine()
						})
						ws.Popup(func(panel ui.Element) {
							for idx, lbl := range wLabels {
								item := ws.Item(idx)
								item.Children(func() {
									components.Text(c, lbl).FontSize(13).SingleLine()
								})
							}
						})
					}
					if a.chartKind != 3 {
						sortBtn := components.ToggleButton(c.Key("chart-sort"), &a.chartSortByValue, "按值排序").
							Height(30).Padding(4, 10).FontSize(12).Cursor(ui.CursorPointer)
						components.Tooltip(c, sortBtn, "按当前指标优劣从左到右排序")
					}
				}
				if a.chartKind != 3 && a.chartGroup == 1 && a.chartBaseStation == "" {
					if a.chartZoomMetric != a.chartMetric {
						a.chartZoomPref = 0
						a.chartZoomMetric = a.chartMetric
					}
					zoomed := a.cachedAutoZoom(a.chartMetric, a.chartCompareCurrent)
					if a.chartZoomPref != 0 {
						zoomed = a.chartZoomPref > 0
					}
					prevZoomed := zoomed
					zoomBtn := components.ToggleButton(c.Key("chart-zoom"), &zoomed, "放大差异").
						Height(30).Padding(4, 10).FontSize(12).Cursor(ui.CursorPointer)
					components.Tooltip(c, zoomBtn, "按均值附近局部缩放 Y 轴，突出微小差异")
					if zoomed != prevZoomed {
						a.chartZoomMetric = a.chartMetric
						if zoomed {
							a.chartZoomPref = 1
						} else {
							a.chartZoomPref = -1
						}
					}
				}

				ui.Spacer(c)

				if summary.TotalSeries > 0 || summary.TotalCategories > 0 {
					components.Text(c, a.chartCaption(summary, payload.IsBox)).FontSize(12).TextColor(t.TextMuted).SingleLine()
					if summary.FocusedSelection {
						quietButton(c, "chart-clear-focus", "取消聚焦", false, func() {
							a.selection.Clear()
						})
					}
				}
			})

			ui.Box(c).FillWidth().MinWidth(0).Children(func() {
				opts := chart.Options{
					Label:       metric.Label + " 数据图表",
					Description: "按电流点或编号对比提取得到的光学与电学参数",
					PlotHeight:  plotHeight,
					FormatValue: func(v float64) string {
						return formatMetricNumber(v, metric.Unit, signedAxis(summary.BaseStation != "", metric))
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

				a.renderUnifiedChart(c, metric, payload, opts)
			})
		})
	})
}

func signedAxis(deltaMode bool, metric metricDef) bool {
	return deltaMode || metric.ID == "shift"
}
