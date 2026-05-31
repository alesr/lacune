// Package gcbench runs a project's own test suite under the garbage collector
// that is the default for its Go toolchain (Green Tea on Go 1.26+, the classic
// GC before that) and reports how the program behaves. It measures wall-clock
// time and parses the runtime GC trace (GODEBUG=gctrace=1) to surface cycles,
// pause time, GC CPU usage, and peak heap.
package gcbench

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"
)

// Options configures a GC benchmark run.
type Options struct {
	TargetModulePath string             // path to the user's Go module (defaults to ".")
	Count            int                // go test -count value (defaults to 1)
	UseDocker        bool               // run inside a pinned golang container for isolation
	OnProgress       func(stage string) // optional progress reporter (nil-safe)
}

// GCStats holds the metrics captured for the run.
type GCStats struct {
	WallTime     time.Duration // wall-clock time of the test run
	NumGC        int           // number of GC cycles observed in the trace
	TotalPauseMs float64       // sum of stop-the-world pause time (ms)
	GCCPUPercent float64       // cumulative GC CPU fraction reported by the runtime
	PeakHeapMB   int           // largest heap size observed during a GC cycle (MB)
}

// Result holds the structured output of a run.
type Result struct {
	UseDocker bool
	GCName    string  // active GC inferred from go.mod ("Green Tea GC" / "Classic GC")
	GoVersion string  // the go.mod "go" directive value
	Stats     GCStats // measured metrics
	Raw       string  // raw test + gctrace output
}

func progress(opts Options, stage string) {
	if opts.OnProgress != nil {
		opts.OnProgress(stage)
	}
}

// Run measures the target module's test suite under its default GC.
func Run(ctx context.Context, opts Options) (Result, error) {
	if opts.Count <= 0 {
		opts.Count = 1
	}

	if opts.TargetModulePath == "" {
		opts.TargetModulePath = "."
	}

	result := Result{UseDocker: opts.UseDocker}

	absPath, err := filepath.Abs(opts.TargetModulePath)
	if err != nil {
		return result, fmt.Errorf("could not resolve module path: %w", err)
	}

	if err := checkPrerequisites(ctx, opts.UseDocker); err != nil {
		return result, fmt.Errorf("env validation failed: %w", err)
	}

	goVersion, greenTeaDefault, err := parseGoModVersion(absPath)
	if err != nil {
		return result, fmt.Errorf("could not parse go mod version: %w", err)
	}

	result.GoVersion = goVersion
	result.GCName = "Classic GC"

	if greenTeaDefault {
		result.GCName = "Green Tea GC"
	}

	dockerImage := fmt.Sprintf("golang:%s", goVersion)

	progress(opts, fmt.Sprintf("Running your tests under %s...", result.GCName))

	stats, raw, err := runMode(ctx, opts, absPath, dockerImage)
	if err != nil {
		return result, fmt.Errorf("benchmark run failed: %w", err)
	}

	result.Stats = stats
	result.Raw = raw
	return result, nil
}

// runMode warms the build cache (so the timed run excludes compilation) and
// then runs the test suite with gctrace enabled, parsing the result.
func runMode(ctx context.Context, opts Options, absPath, dockerImage string) (GCStats, string, error) {
	count := strconv.Itoa(opts.Count)

	progress(opts, "Compiling tests...")

	// Warm the build cache: compile every test binary but run no tests.
	if out, _, err := runTest(ctx, opts, absPath, dockerImage, false, "^$", count); err != nil {
		return GCStats{}, out, err
	}

	progress(opts, "Measuring...")

	out, dur, err := runTest(ctx, opts, absPath, dockerImage, true, ".", count)
	if err != nil {
		return GCStats{}, out, err
	}

	stats := parseGCTrace(out)
	stats.WallTime = dur
	return stats, out, nil
}

// runTest executes `go test` for the whole module. Uses the toolchain default
// GC (no GOEXPERIMENT). When trace is true, GODEBUG=gctrace=1 is set so the
// runtime emits GC telemetry to stderr.
func runTest(ctx context.Context, opts Options, absPath, dockerImage string, trace bool, runPattern, count string) (string, time.Duration, error) {
	env := map[string]string{}
	if trace {
		env["GODEBUG"] = "gctrace=1"
	}

	// -vet=off keeps the run focused on test execution and avoids polluting the
	// trace with the vet tool's own GC activity. -count busts the test cache.
	args := []string{"test", "-run=" + runPattern, "-count=" + count, "-vet=off", "./..."}

	if opts.UseDocker {
		return runDocker(ctx, absPath, dockerImage, env, args)
	}
	return runHost(ctx, absPath, env, args)
}
