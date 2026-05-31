package gcbench

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func runHost(ctx context.Context, dir string, env map[string]string, args []string) (string, time.Duration, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()

	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start)

	if err != nil {
		return buf.String(), dur, fmt.Errorf("%v: %s", err, lastLines(buf.String(), 20))
	}
	return buf.String(), dur, nil
}

// runDocker runs the suite in a pinned container so results are more consistent
// across machines. It caps the workload to a fixed CPU and memory budget
// regardless of how big the host actually is. (Host mode uses the whole
// machine, so docker and host numbers are not directly comparable.)
func runDocker(ctx context.Context, modulePath, dockerImage string, env map[string]string, testArgs []string) (string, time.Duration, error) {
	args := []string{
		"run", "--rm",
		"--cpuset-cpus=0,1",  // pin to 2 specific cores so the process isn't bounced around (less cache noise)
		"--memory=512m",      // fixed heap budget; also keeps GC pacing consistent run to run
		"-e", "GOMAXPROCS=2", // match the runtime's thread count to the 2 pinned cores
		"-v", fmt.Sprintf("%s:/app", modulePath),
		"-w", "/app",
	}

	for k, v := range env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, dockerImage, "go")
	args = append(args, testArgs...)

	cmd := exec.CommandContext(ctx, "docker", args...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start)

	if err != nil {
		return buf.String(), dur, fmt.Errorf("%v: %s", err, lastLines(buf.String(), 20))
	}
	return buf.String(), dur, nil
}

func checkPrerequisites(ctx context.Context, useDocker bool) error {
	if useDocker {
		if err := exec.CommandContext(ctx, "docker", "info").Run(); err != nil {
			return fmt.Errorf("docker daemon is unreachable: %w", err)
		}
	}
	return nil
}

// lastLines returns the final n lines of s, useful for surfacing the tail of a
// failed command's output without dumping everything.
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
