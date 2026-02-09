package runner

import (
	"context"
	"os/exec"
	"strings"
)

type ExecRunner struct{}

func NewExecRunner() *ExecRunner {
	return &ExecRunner{}
}

func (r *ExecRunner) Run(ctx context.Context, dir string, args []string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir

	var stdoutBuilder, stderrBuilder strings.Builder
	cmd.Stdout = &stdoutBuilder
	cmd.Stderr = &stderrBuilder

	err = cmd.Run()
	stdout = stdoutBuilder.String()
	stderr = stderrBuilder.String()
	return
}
