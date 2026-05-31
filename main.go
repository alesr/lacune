package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alesr/lacune/internal/coverage"
	"github.com/alesr/lacune/internal/report"
	"github.com/alesr/lacune/internal/runner"
	"github.com/alesr/lacune/internal/tui"
	"github.com/alesr/lacune/pkg/gcbench"
	"golang.org/x/term"
)

const (
	defaultProfileDir  = ".coverage"
	defaultProfilePath = defaultProfileDir + "/cover.out"
	defaultTimeout     = 10 * time.Minute
)

type flags struct {
	dir      string
	coverpkg string
	profile  string
	noRun    bool
	min      float64
	timeout  time.Duration
	tags     string
	runFlag  string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flags := parseFlags()

	if term.IsTerminal(int(os.Stdout.Fd())) {
		if flags.noRun {
			fileModels, totals, err := parseProfileAndComputeTotals(flags.profile, flags.dir)
			if err != nil {
				return fmt.Errorf("failed to parse profile and compute totals: %w", err)
			}
			if flags.min > 0 && totals.Percent < flags.min {
				return fmt.Errorf("coverage %.2f%% is below minimum %.2f%%", totals.Percent, flags.min)
			}
			if err := tui.Run(fileModels, totals, rerunFunc(flags), benchFunc(flags)); err != nil {
				return fmt.Errorf("could not run TUI: %w", err)
			}
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(flags.profile), 0o755); err != nil {
			return fmt.Errorf("failed to create profile directory: %w", err)
		}

		loader := func() ([]coverage.FileModel, coverage.Totals, tui.LoadDiagnostics, error) {
			stdout, stderr, err := runTests(context.Background(), flags)
			if err != nil {
				return nil, coverage.Totals{}, tui.LoadDiagnostics{Stdout: stdout, Stderr: stderr, Stage: tui.LoadStageTest}, err
			}
			fileModels, totals, err := parseProfileAndComputeTotals(flags.profile, flags.dir)
			if err != nil {
				return nil, coverage.Totals{}, tui.LoadDiagnostics{Stdout: stdout, Stderr: stderr, Stage: tui.LoadStageParse}, err
			}
			if flags.min > 0 && totals.Percent < flags.min {
				return nil, coverage.Totals{}, tui.LoadDiagnostics{Stdout: stdout, Stderr: stderr, Stage: tui.LoadStageMin}, fmt.Errorf("coverage %.2f%% is below minimum %.2f%%", totals.Percent, flags.min)
			}
			return fileModels, totals, tui.LoadDiagnostics{Stdout: stdout, Stderr: stderr}, nil
		}

		if err := tui.RunWithLoader(loader, rerunFunc(flags), benchFunc(flags)); err != nil {
			if loadErr, ok := err.(tui.LoadError); ok {
				if loadErr.Diagnostics.Stage == tui.LoadStageTest {
					fmt.Fprintf(os.Stderr, "go test failed:\n%s\n%s\n", loadErr.Diagnostics.Stdout, loadErr.Diagnostics.Stderr)
				}
				return loadErr.Err
			}
			return fmt.Errorf("could not run TUI: %w", err)
		}
		return nil
	}

	if !flags.noRun {
		if err := os.MkdirAll(filepath.Dir(flags.profile), 0o755); err != nil {
			return fmt.Errorf("failed to create profile directory: %w", err)
		}

		stdout, stderr, err := runTests(context.Background(), flags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "go test failed:\n%s\n%s\n", stdout, stderr)
			return fmt.Errorf("go test failed: %w", err)
		}
	}

	fileModels, totals, err := parseProfileAndComputeTotals(flags.profile, flags.dir)
	if err != nil {
		return fmt.Errorf("failed to parse profile and compute totals: %w", err)
	}

	if flags.min > 0 && totals.Percent < flags.min {
		return fmt.Errorf("coverage %.2f%% is below minimum %.2f%%", totals.Percent, flags.min)
	}

	report.Print(os.Stdout, totals, fileModels, 10)
	return nil
}

func parseFlags() flags {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of lacune:\n")
		flag.PrintDefaults()
	}
	flags := flags{
		dir:      *flag.String("dir", ".", "directory to run in"),
		coverpkg: *flag.String("coverpkg", "", "passed to go test -coverpkg"),
		profile:  *flag.String("profile", defaultProfilePath, "path to coverage profile to read/write"),
		noRun:    *flag.Bool("no-run", false, "do not run tests"),
		min:      *flag.Float64("min", 0, "minimum total coverage percent; if total < min, exit code != 0"),
		timeout:  *flag.Duration("timeout", defaultTimeout, "go test timeout"),
		tags:     *flag.String("tags", "", "build tags"),
		runFlag:  *flag.String("run", "", "forwarded to go test -run"),
	}
	flag.Parse()
	return flags
}

func rerunFunc(f flags) func() ([]coverage.FileModel, coverage.Totals, error) {
	return func() ([]coverage.FileModel, coverage.Totals, error) {
		if _, _, err := runTests(context.Background(), f); err != nil {
			return nil, coverage.Totals{}, fmt.Errorf("could not run tests and compute totals: %w", err)
		}
		fileModels, totals, err := parseProfileAndComputeTotals(f.profile, f.dir)
		if err != nil {
			return nil, coverage.Totals{}, fmt.Errorf("could not parse profile and compute totals: %w", err)
		}
		return fileModels, totals, nil
	}
}

func benchFunc(f flags) tui.BenchRunner {
	return func(ctx context.Context, useDocker bool, count int, onProgress func(string)) (gcbench.Result, error) {
		return gcbench.Run(ctx, gcbench.Options{
			TargetModulePath: f.dir,
			UseDocker:        useDocker,
			Count:            count,
			OnProgress:       onProgress,
		})
	}
}

func runTests(ctx context.Context, f flags) (string, string, error) {
	r := runner.NewExecRunner()

	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	stdout, stderr, err := r.Run(ctx, f.dir, makeTestArgs(f))
	if err != nil {
		return "", "", fmt.Errorf("go test failed: %w", err)
	}
	return stdout, stderr, nil
}

func makeTestArgs(f flags) []string {
	args := []string{"test", "./...", "-coverprofile", f.profile}
	if f.coverpkg != "" {
		args = append(args, "-coverpkg="+f.coverpkg)
	}
	if f.timeout > 0 {
		args = append(args, "-timeout="+f.timeout.String())
	}
	if f.tags != "" {
		args = append(args, "-tags="+f.tags)
	}
	if f.runFlag != "" {
		args = append(args, "-run="+f.runFlag)
	}
	return args
}

func parseProfileAndComputeTotals(profile, dir string) ([]coverage.FileModel, coverage.Totals, error) {
	profiles, err := coverage.Load(profile)
	if err != nil {
		return nil, coverage.Totals{}, fmt.Errorf("could not load coverage profile: %w", err)
	}

	totals := coverage.ComputeTotals(profiles)

	fileModels, err := coverage.BuildFileModels(profiles, dir)
	if err != nil {
		return nil, coverage.Totals{}, fmt.Errorf("failed to build file models: %w", err)
	}
	return fileModels, totals, nil
}
