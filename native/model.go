package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var testCategories = []string{"耦合测试", "Pre测试", "低温储存后测试", "Post测试", "封盖测试", "高温测试"}
var measurements = []string{"LVI", "Rth", "lambd"}

type settings struct {
	Mode                 int      `json:"mode"`
	Entries              string   `json:"entries"`
	ModuleRoot           string   `json:"moduleRoot"`
	ChipRoots            string   `json:"chipRoots"`
	Currents             string   `json:"currents"`
	Measurements         [3]bool  `json:"measurements"`
	Tests                [6]bool  `json:"tests"`
	CustomTests          string   `json:"customTests"`
	UncheckedCustomTests []string `json:"uncheckedCustomTests,omitempty"`
	DeletedTests         []string `json:"deletedTests,omitempty"`
	SidebarWidth         float32  `json:"sidebarWidth"`
}

func defaultSettings() settings {
	return settings{ModuleRoot: "Z:/Ldtd/fcp/", ChipRoots: "Z:/Ldtd/\nZ:/Ldtd/Ldtd/", Measurements: [3]bool{true, true, true}, Tests: [6]bool{true, true, true, true, true, true}, SidebarWidth: 360}
}

func (s settings) customTestChecked(name string) bool {
	return !slices.Contains(s.UncheckedCustomTests, name)
}

func (s *settings) setCustomTestChecked(name string, checked bool) {
	s.UncheckedCustomTests = slices.DeleteFunc(s.UncheckedCustomTests, func(n string) bool { return n == name })
	if !checked {
		s.UncheckedCustomTests = append(s.UncheckedCustomTests, name)
	}
}

func (s settings) stations() []string {
	var names []string
	for _, name := range testCategories {
		if !slices.Contains(s.DeletedTests, name) {
			names = append(names, name)
		}
	}
	for _, name := range lines(s.CustomTests) {
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}

func (s settings) stationChecked(name string) bool {
	if i := slices.Index(testCategories, name); i >= 0 && !slices.Contains(s.DeletedTests, name) {
		return s.Tests[i]
	}
	return s.customTestChecked(name)
}

func (s *settings) setStationChecked(name string, checked bool) {
	if i := slices.Index(testCategories, name); i >= 0 && !slices.Contains(s.DeletedTests, name) {
		s.Tests[i] = checked
		return
	}
	s.setCustomTestChecked(name, checked)
}

func (s settings) selectedTests() []string {
	var selected []string
	for _, name := range s.stations() {
		if s.stationChecked(name) {
			selected = append(selected, name)
		}
	}
	return selected
}

func (s *settings) selectAllTests(checked bool) {
	for i := range s.Tests {
		s.Tests[i] = checked && !slices.Contains(s.DeletedTests, testCategories[i])
	}
	s.UncheckedCustomTests = nil
	if !checked {
		s.UncheckedCustomTests = lines(s.CustomTests)
	}
}

func (s *settings) deleteTest(name string) {
	if i := slices.Index(testCategories, name); i >= 0 {
		s.Tests[i] = false
		if !slices.Contains(s.DeletedTests, name) {
			s.DeletedTests = append(s.DeletedTests, name)
		}
	}
	names := slices.DeleteFunc(lines(s.CustomTests), func(n string) bool { return n == name })
	s.CustomTests = strings.Join(names, "\n")
	s.setCustomTestChecked(name, true)
}

func loadSettings(path string) (settings, error) {
	s := defaultSettings()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return defaultSettings(), fmt.Errorf("读取配置：%w", err)
	}
	if s.Mode != 0 && s.Mode != 1 {
		s.Mode = 0
	}
	return s, nil
}

func saveSettings(path string, s settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, data)
}

// Write beside the destination so a failed save leaves the existing file intact.
func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".data-cos-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

type extractRequest struct {
	Mode           string    `json:"mode"`
	Entries        []string  `json:"entries"`
	TestCategories []string  `json:"test_categories,omitempty"`
	Measurements   []string  `json:"measurements"`
	CurrentPoints  []float64 `json:"current_points"`
	ModuleRoot     string    `json:"module_default_root,omitempty"`
	ChipRoots      []string  `json:"chip_default_roots,omitempty"`
}

type record struct {
	Entry      string   `json:"entry_id"`
	Category   string   `json:"test_category"`
	Current    *float64 `json:"current_a"`
	Power      *float64 `json:"power_w"`
	Voltage    *float64 `json:"voltage_v"`
	Efficiency *float64 `json:"efficiency_pct"`
	Lambda     *float64 `json:"lambda_nm"`
	Shift      *float64 `json:"shift_nm"`
	TwoA       *float64 `json:"wavelength_2a_nm"`
	Cold       *float64 `json:"wavelength_cold_nm"`
}

// The selection follows a record when sorting changes its displayed index.
type rowIdentity struct {
	Entry      string
	Category   string
	Current    float64
	HasCurrent bool
}

func (r record) identity() rowIdentity {
	key := rowIdentity{Entry: r.Entry, Category: r.Category}
	if r.Current != nil {
		key.Current = *r.Current
		key.HasCurrent = true
	}
	return key
}

type extractResult struct {
	Total   int      `json:"total"`
	Records []record `json:"records"`
	Errors  []string `json:"errors"`
	Infos   []string `json:"infos"`
}

func lines(raw string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' || r == '，' }) {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			out = append(out, s)
			seen[s] = true
		}
	}
	return out
}

const numberPattern = `[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?`

var currentRange = regexp.MustCompile(`^(` + numberPattern + `)\s*[~-]\s*(` + numberPattern + `)$`)
var rangeSpaces = regexp.MustCompile(`(` + numberPattern + `)\s*([~-])\s*(` + numberPattern + `)`)

func parseCurrents(raw string) ([]float64, error) {
	raw = strings.TrimSpace(raw)
	if strings.EqualFold(raw, "a") {
		return nil, nil
	}
	points := []float64{}
	seen := map[float64]bool{}
	add := func(v float64) error {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("电流必须为有限数值")
		}
		if !seen[v] {
			points = append(points, v)
			seen[v] = true
		}
		if len(points) > 10000 {
			return fmt.Errorf("电流点不得超过 10000 个")
		}
		return nil
	}
	raw = strings.NewReplacer("，", ",", "～", "~").Replace(raw)
	raw = rangeSpaces.ReplaceAllString(raw, "$1$2$3")
	for _, piece := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r' }) {
		if v, err := strconv.ParseFloat(piece, 64); err == nil {
			if err = add(v); err != nil {
				return nil, err
			}
			continue
		}
		match := currentRange.FindStringSubmatch(piece)
		if match == nil {
			return nil, fmt.Errorf("无法读取电流：%s", piece)
		}
		start, _ := strconv.ParseFloat(match[1], 64)
		end, _ := strconv.ParseFloat(match[2], 64)
		if math.IsInf(start, 0) || math.IsInf(end, 0) {
			return nil, fmt.Errorf("电流超出数值范围")
		}
		if math.Trunc(start) == start && math.Trunc(end) == end {
			if math.Abs(end-start) > 9999 {
				return nil, fmt.Errorf("电流范围不得超过 10000 个点")
			}
			step := 1.0
			if end < start {
				step = -1
			}
			for i := 0; i <= int(math.Abs(end-start)); i++ {
				if err := add(start + float64(i)*step); err != nil {
					return nil, err
				}
			}
		} else {
			if err := add(start); err != nil {
				return nil, err
			}
			if err := add(end); err != nil {
				return nil, err
			}
		}
	}
	return points, nil
}

func (s settings) request() (extractRequest, error) {
	r := extractRequest{Mode: "module", Entries: lines(s.Entries), Measurements: []string{}, ModuleRoot: strings.TrimSpace(s.ModuleRoot)}
	if len(r.Entries) == 0 {
		return r, fmt.Errorf("请输入至少一个编号或路径")
	}
	for i, checked := range s.Measurements {
		if checked {
			r.Measurements = append(r.Measurements, measurements[i])
		}
	}
	if len(r.Measurements) == 0 {
		return r, fmt.Errorf("请选择至少一种测试文件")
	}
	var err error
	r.CurrentPoints, err = parseCurrents(s.Currents)
	if err != nil {
		return r, err
	}
	if s.Mode == 1 {
		r.Mode = "chip"
		r.ChipRoots = lines(s.ChipRoots)
		return r, nil
	}
	r.TestCategories = s.selectedTests()
	if len(r.TestCategories) == 0 {
		return r, fmt.Errorf("请选择至少一个测试站别")
	}
	return r, nil
}

func (r record) values() []any {
	values := []any{r.Entry, r.Category}
	for _, p := range []*float64{r.Current, r.Power, r.Voltage, r.Efficiency, r.Lambda, r.Shift, r.TwoA, r.Cold} {
		if p == nil {
			values = append(values, nil)
		} else {
			values = append(values, *p)
		}
	}
	return values
}

func formatValue(v any) string {
	if v == nil {
		return "—"
	}
	if n, ok := v.(float64); ok {
		return strconv.FormatFloat(n, 'f', 3, 64)
	}
	return fmt.Sprint(v)
}

// Clipboard and export use the source value, independently of display precision.
func plainValue(v any) string {
	if v == nil {
		return ""
	}
	if n, ok := v.(float64); ok {
		return strconv.FormatFloat(n, 'g', -1, 64)
	}
	return fmt.Sprint(v)
}
