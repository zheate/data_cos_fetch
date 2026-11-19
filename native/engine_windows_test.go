package main

import (
	"context"
	"debug/pe"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestEngineLaunchSuppressesConsoleCreation(t *testing.T) {
	cmd := newEngineCommand(context.Background(), "data-cos-extract.exe")
	if cmd.SysProcAttr == nil || cmd.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatal("console worker launch must set CREATE_NO_WINDOW, including when the parent has no console")
	}
}

// Run a console-subsystem extraction worker without the UI runtime. This probes
// the actual runEngine launch path rather than just checking creation flags.
func TestEngineRunsWithoutConsoleWindow(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "console-worker.exe")
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go.exe")
	if output, err := exec.Command(goTool, "build", "-o", executable, "./testdata/console-worker").CombinedOutput(); err != nil {
		t.Fatalf("build console worker: %v\n%s", err, output)
	}
	image, err := pe.Open(executable)
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	if image.OptionalHeader.(*pe.OptionalHeader64).Subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_CUI {
		t.Fatal("test worker must use the console subsystem")
	}
	for range 3 {
		result, err := runEngine(context.Background(), executable, extractRequest{Entries: []string{"中文编号-01"}})
		if err != nil {
			t.Fatalf("worker must exchange JSON without a console window: %v", err)
		}
		if result.Total != 1 || result.Records[0].Entry != "中文编号-01" {
			t.Fatalf("hidden worker lost its JSON result: %+v", result)
		}
	}
	_, err = runEngine(context.Background(), executable, extractRequest{Mode: "failure"})
	if err == nil || !strings.Contains(err.Error(), "worker diagnostic") || !strings.Contains(err.Error(), "exit status 23") {
		t.Fatalf("hidden worker lost stderr or exit status: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err = runEngine(ctx, executable, extractRequest{Mode: "wait"})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 3*time.Second {
		t.Fatalf("hidden worker cancellation failed: %v after %v", err, time.Since(started))
	}
}
