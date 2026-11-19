//go:build !windows

package main

import (
	"context"
	"os/exec"
)

func newEngineCommand(ctx context.Context, engine string) *exec.Cmd {
	return exec.CommandContext(ctx, engine)
}
