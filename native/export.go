package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

var columnTitles = []string{"编号", "测试站别", "电流 / A", "功率 / W", "电压 / V", "效率 / %", "波长 / nm", "波长漂移 / nm", "2A 波长 / nm", "冷波长 / nm"}

func exportExcel(path string, result *extractResult) error {
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", "提取结果"); err != nil {
		return err
	}
	headers := make([]any, len(columnTitles))
	for i, v := range columnTitles {
		headers[i] = v
	}
	if err := f.SetSheetRow("提取结果", "A1", &headers); err != nil {
		return err
	}
	for i, r := range result.Records {
		values := r.values()
		if err := f.SetSheetRow("提取结果", fmt.Sprintf("A%d", i+2), &values); err != nil {
			return err
		}
	}
	if err := f.SetColWidth("提取结果", "A", "A", 32); err != nil {
		return err
	}
	if err := f.SetColWidth("提取结果", "B", "B", 22); err != nil {
		return err
	}
	if err := f.SetColWidth("提取结果", "C", "J", 18); err != nil {
		return err
	}
	if err := f.SetPanes("提取结果", &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	if len(result.Errors)+len(result.Infos) > 0 {
		if _, err := f.NewSheet("提取提示"); err != nil {
			return err
		}
		for i, msg := range append(append([]string{}, result.Errors...), result.Infos...) {
			if err := f.SetCellStr("提取提示", fmt.Sprintf("A%d", i+1), msg); err != nil {
				return err
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return err
	}
	return atomicWrite(path, buf.Bytes())
}

// Prefix formula-like text in CSV/TSV; XLSX string cells retain their type.
func safeText(s string) string {
	if strings.HasPrefix(s, "=") || strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") || strings.HasPrefix(s, "@") || strings.HasPrefix(s, "\t") || strings.HasPrefix(s, "\r") {
		return "'" + s
	}
	return s
}

func delimited(records []record, separator rune) ([]byte, error) {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Comma = separator
	w.UseCRLF = true
	if err := w.Write(columnTitles); err != nil {
		return nil, err
	}
	for _, r := range records {
		row := make([]string, len(columnTitles))
		for j, v := range r.values() {
			row[j] = plainValue(v)
			if j < 2 {
				row[j] = safeText(row[j])
			}
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return b.Bytes(), w.Error()
}
