package main

import (
	"context"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func newEngineCommand(ctx context.Context, engine string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, engine)
	// The Rust worker keeps its console subsystem for command-line use. Suppress
	// console allocation when launching it from the GUI, while retaining pipes.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
		HideWindow:    true,
	}
	return cmd
}
