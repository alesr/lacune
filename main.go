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
	"golang.org/x/term"
	"golang.org/x/tools/cover"
)

const (
	defaultProfileDir  = ".coverage"
	defaultProfilePath = defaultProfileDir + "/cover.out"
	defaultTimeout     = 10 * time.Minute
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of lacune:\n")
		flag.PrintDefaults()
	}

	dir := flag.String("dir", ".", "directory to run in")
	coverpkg := flag.String("coverpkg", "", "passed to go test -coverpkg")
	profile := flag.String("profile", defaultProfilePath, "path to coverage profile to read/write")
	noRun := flag.Bool("no-run", false, "do not run tests; only read -profile")
	min := flag.Float64("min", 0, "minimum total coverage percent; if total < min, exit code != 0")
	timeout := flag.Duration("timeout", defaultTimeout, "go test timeout")
	tags := flag.String("tags", "", "build tags")
	runFlag := flag.String("run", "", "forwarded to go test -run")
	flag.Parse()

	if !*noRun {
		if err := os.MkdirAll(filepath.Dir(*profile), 0o755); err != nil {
			return fmt.Errorf("failed to create profile directory: %w", err)
		}
	}

	var profiles []*cover.Profile
	if !*noRun {
		args := []string{"test", "./...", "-coverprofile", *profile}
		if *coverpkg != "" {
			args = append(args, "-coverpkg="+*coverpkg)
		}
		if *timeout > 0 {
			args = append(args, "-timeout="+timeout.String())
		}
		if *tags != "" {
			args = append(args, "-tags="+*tags)
		}
		if *runFlag != "" {
			args = append(args, "-run="+*runFlag)
		}

		r := runner.NewExecRunner()
		stdout, stderr, err := r.Run(context.Background(), *dir, args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "go test failed:\n%s\n%s\n", stdout, stderr)
			return fmt.Errorf("go test failed: %w", err)
		}
	}

	// parse coverage profile
	var err error
	profiles, err = coverage.Load(*profile)
	if err != nil {
		return fmt.Errorf("failed to load coverage profile: %w", err)
	}

	// compute totals
	totals := coverage.ComputeTotals(profiles)
	if *min > 0 && totals.Percent < *min {
		return fmt.Errorf("coverage %.2f%% is below minimum %.2f%%", totals.Percent, *min)
	}

	// build file models for TUI/text report
	fileModels, err := coverage.BuildFileModels(profiles, *dir)
	if err != nil {
		return fmt.Errorf("failed to build file models: %w", err)
	}

	// define rerun function
	rerunFunc := func() ([]coverage.FileModel, coverage.Totals, error) {
		// run tests
		args := []string{"test", "./...", "-coverprofile", *profile}
		if *coverpkg != "" {
			args = append(args, "-coverpkg="+*coverpkg)
		}
		if *timeout > 0 {
			args = append(args, "-timeout="+timeout.String())
		}
		if *tags != "" {
			args = append(args, "-tags="+*tags)
		}
		if *runFlag != "" {
			args = append(args, "-run="+*runFlag)
		}

		r := runner.NewExecRunner()
		stdout, stderr, err := r.Run(context.Background(), *dir, args)
		if err != nil {
			return nil, coverage.Totals{}, fmt.Errorf("go test failed: %w\n%s\n%s", err, stdout, stderr)
		}

		// parse coverage profile
		profiles, err := coverage.Load(*profile)
		if err != nil {
			return nil, coverage.Totals{}, fmt.Errorf("failed to load coverage profile: %w", err)
		}

		// compute totals
		totals := coverage.ComputeTotals(profiles)

		// build file models
		fileModels, err := coverage.BuildFileModels(profiles, *dir)
		if err != nil {
			return nil, coverage.Totals{}, fmt.Errorf("failed to build file models: %w", err)
		}
		return fileModels, totals, nil
	}

	// decide output mode (tui vs text)
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	text := flag.Bool("text", false, "force text output (no TUI)")
	if isTTY && !*text {
		if err := tui.Run(fileModels, totals, rerunFunc); err != nil {
			return fmt.Errorf("TUI failed: %w", err)
		}
	} else {
		report.Print(os.Stdout, totals, fileModels, 10)
	}
	return nil
}
