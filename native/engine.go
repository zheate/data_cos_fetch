package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func findEngine() (string, error) {
	if p := os.Getenv("DATA_COS_EXTRACT_BIN"); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
		return "", fmt.Errorf("找不到 DATA_COS_EXTRACT_BIN 指定的提取引擎：%s", p)
	}
	name := "data-cos-extract"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	exe, _ := os.Executable()
	candidates := []string{filepath.Join(filepath.Dir(exe), name), filepath.Join(filepath.Dir(exe), "..", "Resources", name), filepath.Join("..", "rust", "target", "release", name), filepath.Join("..", "rust", "target", "debug", name)}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return filepath.Abs(p)
		}
	}
	return "", fmt.Errorf("缺少提取引擎，请运行 npm run build:mygo")
}

func runEngine(ctx context.Context, engine string, request extractRequest) (*extractResult, error) {
	input, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	cmd := newEngineCommand(ctx, engine)
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("提取失败：%s（%w）", strings.TrimSpace(stderr.String()), err)
	}
	var result extractResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("提取引擎返回无效数据：%w", err)
	}
	if result.Total != len(result.Records) {
		return nil, fmt.Errorf("提取引擎返回的记录数量不一致")
	}
	return &result, nil
}
