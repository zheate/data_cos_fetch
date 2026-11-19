package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/components/chart"
)

type taskOutcome struct {
	result  *extractResult
	apply   func()
	err     error
	message string
	receipt string
	elapsed time.Duration
}

type application struct {
	config              settings
	settingsPath        string
	engine              string
	engineError         error
	window              *mygo.Window
	setThemeSource      func(mygo.ThemeSource)
	wake                func()
	worker              func(context.Context, string, extractRequest) (*extractResult, error)
	completions         chan taskOutcome
	cancel              context.CancelFunc
	busy                bool
	status              string
	statusError         bool
	operationErrors     []string
	started             time.Time
	result              *extractResult
	rows                []record
	query               string
	resultTab           int
	table               components.TableVirtualizedState
	selection           ui.Selection[any]
	sort                ui.SortOrder
	chartState          chart.State
	chartKind           int
	chartGroup          int
	chartMetric         int
	chartBaseStation    string
	chartCompareCurrent string
	chartCompareWindow  int
	chartSortByValue    bool
	chartHoverSeries    string
	chartZoomPref       int // 0=auto, 1=force magnified Y, -1=force zero baseline
	chartZoomMetric     int
	rowsRev             uint64
	chartCache          chartCacheState
	entriesField        ui.Handle
	currentsField       ui.Handle
	searchField         ui.Handle
	currentMode         int
	customCurrents      string
	lastConfigCurrents  string
	sidebarWidth        float32
	zoom                float32
	settingsOpen        bool
	customOpen          bool
	customDraft         string
	dialogOpen          bool
	dialogTitle         string
	dialogText          string
	receipt             string
}

func newApplication() *application {
	a := &application{
		config:           defaultSettings(),
		sidebarWidth:     304,
		zoom:             defaultWorkbenchZoom,
		worker:           runEngine,
		completions:      make(chan taskOutcome, 1),
		chartState:       chart.State{Index: -1},
		chartSortByValue: true,
	}
	a.initTableState()
	a.syncCurrentMode()
	return a
}

const (
	defaultWorkbenchZoom = 1.3
	minWorkbenchZoom     = 0.7
	maxWorkbenchZoom     = 2.0
	workbenchZoomStep    = 0.1
)

func (a *application) toggleTheme(dark bool) {
	if dark {
		a.config.Theme = string(mygo.ThemeLight)
	} else {
		a.config.Theme = string(mygo.ThemeDark)
	}
	if a.setThemeSource != nil {
		a.setThemeSource(mygo.ThemeSource(a.config.Theme))
	}
	if a.settingsPath != "" {
		if err := saveSettings(a.settingsPath, a.config); err != nil {
			a.reportError("保存亮暗模式失败：" + err.Error())
		}
	}
}

func (a *application) setZoom(z float32) {
	clamped := max(float32(minWorkbenchZoom), min(float32(maxWorkbenchZoom), z))
	rounded := float32(math.Round(float64(clamped)*100) / 100)
	a.zoom = rounded
	a.statusError = false
	a.status = fmt.Sprintf("界面缩放 %d%%", int(math.Round(float64(rounded)*100)))
}

func (a *application) adjustZoom(delta float32) {
	if a.zoom <= 0 {
		a.zoom = defaultWorkbenchZoom
	}
	a.setZoom(a.zoom + delta)
}

func (a *application) syncCurrentMode() {
	trimmed := strings.TrimSpace(a.config.Currents)
	switch {
	case trimmed == "":
		a.currentMode = 0
	case strings.EqualFold(trimmed, "a"):
		a.currentMode = 1
	default:
		a.currentMode = 2
		a.customCurrents = a.config.Currents
	}
	a.lastConfigCurrents = a.config.Currents
}

func (a *application) initTableState() {
	a.table = components.TableVirtualizedState{
		TableState: components.TableState{
			Sort:   &a.sort,
			OnSort: a.sortRows,
		},
		List: components.ListState{
			Selection:  &a.selection,
			KeepOffset: true,
			Key: func(row int) any {
				if row < 0 || row >= len(a.rows) {
					return nil
				}
				return a.rows[row].identity()
			},
		},
	}
}

func (a *application) launch(label string, task func(context.Context) taskOutcome) {
	if a.busy {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.busy = true
	a.started = time.Now()
	a.status = label
	a.statusError = false
	go func() {
		defer cancel()
		out := task(ctx)
		if ctx.Err() != nil {
			out = taskOutcome{err: ctx.Err()}
		}
		out.elapsed = time.Since(a.started)
		a.completions <- out
		if a.wake != nil {
			a.wake()
		}
	}()
}

func (a *application) finishTasks() {
	select {
	case out := <-a.completions:
		a.busy = false
		a.cancel = nil
		if out.err != nil {
			if errors.Is(out.err, context.Canceled) {
				a.status = "操作已取消，保留上次结果"
				return
			}
			a.reportError(out.err.Error())
			return
		}
		if out.apply != nil {
			out.apply()
		}
		if out.result != nil {
			sortRecordsByStation(out.result.Records, a.config)
			a.result = out.result
			a.operationErrors = nil
			a.query = ""
			a.resultTab = 0
			a.receipt = out.receipt
			a.selection.Clear()
			a.sort = ui.SortOrder{}
			a.initTableState()
			a.filter()
			a.chartState = chart.State{Index: -1}
			a.chartHoverSeries = ""
			a.chartCompareWindow = 0
			a.chartZoomPref = 0
			a.chartZoomMetric = 0
			a.chartCache = chartCacheState{}
			a.resetChartDefaults()
			if len(a.result.Records) == 0 && len(a.result.Errors)+len(a.result.Infos) > 0 {
				a.resultTab = 2
			}
			a.status = fmt.Sprintf("提取完成 · %d 条记录 · 耗时 %.1f 秒", len(a.result.Records), out.elapsed.Seconds())
			if len(a.result.Errors) > 0 {
				a.status = fmt.Sprintf("%d 条记录", len(a.result.Records))
			}
		} else if out.message != "" {
			a.status = out.message
		} else {
			a.status = "操作已取消"
		}
	default:
	}
}

func (a *application) run() {
	if a.busy {
		return
	}
	request, err := a.config.request()
	if err != nil {
		a.reportError(err.Error())
		return
	}
	if a.engineError != nil {
		a.reportError(a.engineError.Error())
		return
	}
	if a.settingsPath != "" {
		if err := saveSettings(a.settingsPath, a.config); err != nil {
			a.reportError("保存配置失败：" + err.Error())
			return
		}
	}
	worker, engine := a.worker, a.engine
	receipt := fmt.Sprintf("%s · %s · %d 个编号", time.Now().Format("15:04:05"), []string{"模块", "芯片"}[a.config.Mode], len(request.Entries))
	a.launch("正在读取测试文件", func(ctx context.Context) taskOutcome {
		result, err := worker(ctx, engine, request)
		return taskOutcome{result: result, err: err, receipt: receipt}
	})
}

func (a *application) filter() {
	a.rowsRev++
	a.rows = nil
	a.selection.Clear()
	if a.result == nil {
		return
	}
	q := strings.ToLower(strings.TrimSpace(a.query))
	for _, r := range a.result.Records {
		if q == "" || strings.Contains(strings.ToLower(r.Entry+" "+r.Category), q) {
			a.rows = append(a.rows, r)
		}
	}
	a.sortRows()
}

func (a *application) sortRows() {
	col := -1
	for i, title := range columnTitles {
		if a.sort.Column == title {
			col = i
			break
		}
	}
	if col < 0 {
		sortRecordsByStation(a.rows, a.config)
		return
	}
	a.rowsRev++
	sort.SliceStable(a.rows, func(i, j int) bool {
		x, y := a.rows[i].values()[col], a.rows[j].values()[col]
		if x == nil {
			return false
		}
		if y == nil {
			return true
		}
		less := false
		switch {
		case col == 1:
			ri, rj := a.config.stationRank(a.rows[i].Category), a.config.stationRank(a.rows[j].Category)
			if ri == rj {
				if a.rows[i].Category == a.rows[j].Category {
					return false
				}
				less = a.rows[i].Category < a.rows[j].Category
			} else {
				less = ri < rj
			}
		default:
			if xn, ok := x.(float64); ok {
				less = xn < y.(float64)
				if xn == y.(float64) {
					return false
				}
			} else {
				if x == y {
					return false
				}
				less = fmt.Sprint(x) < fmt.Sprint(y)
			}
		}
		if a.sort.Descending {
			return !less
		}
		return less
	})
}

func sortRecordsByStation(records []record, s settings) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Entry != records[j].Entry {
			return records[i].Entry < records[j].Entry
		}
		ri, rj := s.stationRank(records[i].Category), s.stationRank(records[j].Category)
		if ri != rj {
			return ri < rj
		}
		if records[i].Category != records[j].Category {
			return records[i].Category < records[j].Category
		}
		ci, cj := records[i].Current, records[j].Current
		if ci == nil {
			return false
		}
		if cj == nil {
			return true
		}
		return *ci < *cj
	})
}

func (a *application) clearResult() {
	if a.busy {
		return
	}
	a.result = nil
	a.rows = nil
	a.rowsRev++
	a.chartCache = chartCacheState{}
	a.query = ""
	a.receipt = ""
	a.resultTab = 0
	a.selection.Clear()
	a.chartState = chart.State{Index: -1}
	a.chartBaseStation = ""
	a.chartCompareCurrent = ""
	a.chartCompareWindow = 0
	a.chartZoomPref = 0
	a.chartZoomMetric = 0
	a.chartHoverSeries = ""
	a.status = ""
	a.statusError = false
	a.operationErrors = nil
}

func (a *application) openCustomTests() {
	a.customDraft = a.config.CustomTests
	a.customOpen = true
}

func (a *application) deleteSelectedStations() {
	if a.busy {
		return
	}
	selected := a.config.selectedDeletableTests()
	if len(selected) == 0 {
		return
	}
	for _, name := range selected {
		a.config.deleteTest(name)
	}
	a.entriesField.Focus()
}

func (a *application) saveCustomTests() {
	names := lines(a.customDraft)
	a.config.CustomTests = strings.Join(names, "\n")
	a.config.UncheckedCustomTests = slices.DeleteFunc(a.config.UncheckedCustomTests, func(name string) bool {
		return !slices.Contains(names, name)
	})
	a.customOpen = false
}

func (a *application) tidyEntries() {
	a.config.Entries = strings.Join(lines(a.config.Entries), "\n")
	a.entriesField.Focus()
}

func (a *application) handleDroppedEntries(paths []string) {
	if a.busy || len(paths) == 0 {
		return
	}
	existing := lines(a.config.Entries)
	seen := make(map[string]bool, len(existing))
	for _, item := range existing {
		seen[item] = true
	}
	var added []string
	appendUnique := func(item string) {
		item = strings.TrimSpace(item)
		if item != "" && !seen[item] {
			seen[item] = true
			added = append(added, item)
		}
	}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		st, err := os.Stat(p)
		if err == nil && !st.IsDir() && strings.EqualFold(filepath.Ext(p), ".txt") {
			if st.Size() > 8*1024*1024 {
				a.reportError("清单超过 8 MiB，请分批导入")
				return
			}
			data, err := os.ReadFile(p)
			if err != nil {
				a.reportError("读取文件失败：" + err.Error())
				return
			}
			text := strings.TrimPrefix(string(data), "\ufeff")
			for _, item := range lines(text) {
				appendUnique(item)
			}
		} else {
			appendUnique(p)
		}
	}
	a.statusError = false
	if len(added) > 0 {
		merged := append(existing, added...)
		a.config.Entries = strings.Join(merged, "\n")
		a.status = fmt.Sprintf("已添加 %d 个编号或路径", len(added))
		a.entriesField.Focus()
	} else {
		a.status = "编号清单未新增条目"
	}
}

func (a *application) focusRowInChart(row int) {
	if row < 0 || row >= len(a.rows) {
		return
	}
	id := a.rows[row].identity()
	if !a.selection.Has(id) {
		a.selection.Clear()
		a.selection.Add(id)
	}
	a.resultTab = 1
	a.statusError = false
	a.status = fmt.Sprintf("已在图表中聚焦 %s", a.rows[row].Entry)
}

func (a *application) selectedRows() []record {
	if a.selection.Len() == 0 {
		return nil
	}
	var rows []record
	for _, row := range a.rows {
		if a.selection.Has(row.identity()) {
			rows = append(rows, row)
		}
	}
	return rows
}

func (a *application) chooseRoot() {
	mode, parent := a.config.Mode, a.window
	a.launch("选择数据目录", func(ctx context.Context) taskOutcome {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: parent, Title: "选择数据根目录", Directory: true, Multiple: mode == 1})
		if err != nil || len(paths) == 0 {
			return taskOutcome{err: err}
		}
		return taskOutcome{message: "数据根目录已更新", apply: func() {
			if mode == 0 {
				a.config.ModuleRoot = paths[0]
			} else {
				a.config.ChipRoots = strings.Join(paths, "\n")
			}
		}}
	})
}

func (a *application) importEntries() {
	parent := a.window
	a.launch("导入编号清单", func(ctx context.Context) taskOutcome {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: parent, Title: "导入编号清单", Filters: []mygo.FileFilter{{Name: "文本清单", Extensions: []string{"txt"}}}})
		if err != nil || len(paths) == 0 {
			return taskOutcome{err: err}
		}
		st, err := os.Stat(paths[0])
		if err != nil {
			return taskOutcome{err: err}
		}
		if st.Size() > 8*1024*1024 {
			return taskOutcome{err: fmt.Errorf("清单超过 8 MiB，请分批导入")}
		}
		data, err := os.ReadFile(paths[0])
		if err != nil {
			return taskOutcome{err: err}
		}
		text := strings.TrimPrefix(string(data), "\ufeff")
		return taskOutcome{message: fmt.Sprintf("导入 %d 个编号", len(lines(text))), apply: func() { a.config.Entries = text }}
	})
}

func (a *application) export(kind string) {
	if a.busy || a.result == nil || len(a.result.Records) == 0 {
		return
	}
	result, parent := a.result, a.window
	a.launch("正在导出全部记录", func(ctx context.Context) taskOutcome {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: parent, Title: "导出全部提取记录", DefaultPath: "Data_Fetch_Output." + kind, Filters: []mygo.FileFilter{{Name: strings.ToUpper(kind), Extensions: []string{kind}}}})
		if err != nil || path == "" {
			return taskOutcome{err: err}
		}
		if filepath.Ext(path) == "" {
			path += "." + kind
		}
		if kind == "xlsx" {
			err = exportExcel(path, result)
		} else {
			var data []byte
			data, err = delimited(result.Records, ',')
			if err == nil {
				err = atomicWrite(path, append([]byte{0xef, 0xbb, 0xbf}, data...))
			}
		}
		return taskOutcome{err: err, message: fmt.Sprintf("导出 %d 条记录至 %s", len(result.Records), path)}
	})
}

func (a *application) diagnostics() {
	roots := []string{strings.TrimSpace(a.config.ModuleRoot)}
	if a.config.Mode == 1 {
		roots = lines(a.config.ChipRoots)
	}
	engine, engineErr := a.engine, a.engineError
	a.launch("正在检查读取环境", func(ctx context.Context) taskOutcome {
		var report strings.Builder
		if engineErr != nil {
			fmt.Fprintf(&report, "失败 · %s\n\n", engineErr)
		} else {
			fmt.Fprintf(&report, "提取引擎\n%s\n\n", engine)
		}
		if len(roots) == 0 {
			report.WriteString("未设置根目录；完整路径仍可以直接提取\n")
		}
		for _, root := range roots {
			if root == "" {
				continue
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				fmt.Fprintf(&report, "读取失败 · %s\n%s\n\n", root, err)
			} else {
				fmt.Fprintf(&report, "可以读取 · %s\n%d 个直属文件或目录\n\n", root, len(entries))
			}
		}
		return taskOutcome{message: "读取环境检测完成", apply: func() { a.dialogTitle = "读取环境检测"; a.dialogText = report.String(); a.dialogOpen = true }}
	})
}
