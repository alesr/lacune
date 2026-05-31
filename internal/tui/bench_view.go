package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/alesr/lacune/pkg/gcbench"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type benchState int

const (
	benchReady benchState = iota
	benchRunning
	benchResults
	benchError

	defaultBenchCount = 3
	benchCountMin     = 1
	benchCountMax     = 50
)

type benchViewModel struct {
	viewport      viewport.Model
	open          bool
	state         benchState
	useDocker     bool
	count         int
	progress      string
	err           error
	showHelp      bool
	resultContent string
	width         int
	height        int
}

func newBenchViewModel() benchViewModel {
	return benchViewModel{
		viewport: viewport.Model{Width: 0, Height: 0},
		count:    defaultBenchCount,
	}
}

func (model benchViewModel) IsOpen() bool      { return model.open }
func (model benchViewModel) State() benchState { return model.state }
func (model benchViewModel) UseDocker() bool   { return model.useDocker }
func (model benchViewModel) Count() int        { return model.count }

func (model benchViewModel) IncCount() benchViewModel {
	newModel := model
	if newModel.count < benchCountMax {
		newModel.count++
	}
	return newModel
}

func (model benchViewModel) DecCount() benchViewModel {
	newModel := model
	if newModel.count > benchCountMin {
		newModel.count--
	}
	return newModel
}

func (model benchViewModel) Open() benchViewModel {
	newModel := model
	newModel.open = true
	newModel.state = benchReady
	newModel.progress = ""
	newModel.err = nil
	return newModel
}

func (model benchViewModel) Close() benchViewModel {
	newModel := model
	newModel.open = false
	return newModel
}

func (model benchViewModel) ToggleDocker() benchViewModel {
	newModel := model
	newModel.useDocker = !newModel.useDocker
	return newModel
}

func (model benchViewModel) SetRunning(stage string) benchViewModel {
	newModel := model
	newModel.state = benchRunning
	newModel.progress = stage
	newModel.err = nil
	return newModel
}

func (model benchViewModel) SetProgress(stage string) benchViewModel {
	newModel := model
	newModel.progress = stage
	return newModel
}

func (model benchViewModel) SetResult(content string) benchViewModel {
	newModel := model
	newModel.state = benchResults
	newModel.showHelp = false
	newModel.resultContent = content
	newModel.viewport.SetContent(content)
	newModel.viewport.GotoTop()
	return newModel
}

func (model benchViewModel) ToggleHelp() benchViewModel {
	newModel := model
	newModel.showHelp = !newModel.showHelp

	if newModel.showHelp {
		newModel.viewport.SetContent(helpContent())
	} else {
		newModel.viewport.SetContent(newModel.resultContent)
	}

	newModel.viewport.GotoTop()
	return newModel
}

func (model benchViewModel) SetError(err error) benchViewModel {
	newModel := model
	newModel.state = benchError
	newModel.err = err
	return newModel
}

func (model benchViewModel) SetSize(width, height int) benchViewModel {
	newModel := model
	newModel.viewport.Width = width
	newModel.viewport.Height = height
	newModel.width = width
	newModel.height = height
	return newModel
}

func (model benchViewModel) ScrollUp(lines int) benchViewModel {
	newModel := model
	newModel.viewport.ScrollUp(lines)
	return newModel
}

func (model benchViewModel) ScrollDown(lines int) benchViewModel {
	newModel := model
	newModel.viewport.ScrollDown(lines)
	return newModel
}

func (model benchViewModel) View(spinner string) string {
	styles := defaultStyles()
	titleText := "GC Report"

	if model.state == benchResults && model.showHelp {
		titleText = "Metric guide"
	}
	title := styles.packageHeader.Render(titleText)

	var body string

	switch model.state {
	case benchReady:
		body = model.readyView(styles)
	case benchRunning:
		body = styles.normalDesc.Render(fmt.Sprintf("%s %s", spinner, model.progress))
	case benchResults:
		body = model.viewport.View()
	case benchError:
		body = styles.error.Render(fmt.Sprintf("Error: %v", model.err))
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func (model benchViewModel) readyView(styles styles) string {
	var content strings.Builder
	content.WriteString(styles.normalDesc.Render("Runs your test suite under your Go toolchain's default GC and reports"))
	content.WriteString("\n")
	content.WriteString(styles.normalDesc.Render("how your code behaves (timing + GC telemetry)."))
	content.WriteString("\n\n")
	dockerState := "off"

	if model.useDocker {
		dockerState = "on"
	}

	content.WriteString(styles.normalDesc.Render(fmt.Sprintf("iterations: %d    docker isolation: %s", model.count, dockerState)))
	return content.String()
}

// buildBenchContent renders the single GC run report plus raw gctrace.
func buildBenchContent(res gcbench.Result) string {
	styles := defaultStyles()
	var b strings.Builder

	header := fmt.Sprintf("Your test suite under %s", res.GCName)
	if res.GoVersion != "" {
		header += fmt.Sprintf(" (Go %s)", res.GoVersion)
	}

	b.WriteString(styles.packageHeader.Render(header))
	b.WriteString("\n\n")
	b.WriteString(styles.neutral.Render("Measured by running your tests, which approximates (not equals) your production workload."))
	b.WriteString("\n\n")

	s := res.Stats

	b.WriteString(fmt.Sprintf("  %-16s %s\n", "Metric", "Value"))
	b.WriteString("  " + strings.Repeat("-", 28) + "\n")
	writeBenchRow(&b, "Wall time", fmtDur(s.WallTime))
	writeBenchRow(&b, "GC cycles", fmt.Sprintf("%d", s.NumGC))
	writeBenchRow(&b, "Total GC pause", fmtMs(s.TotalPauseMs))
	writeBenchRow(&b, "GC CPU", fmtPct(s.GCCPUPercent))
	writeBenchRow(&b, "Peak heap", fmtMB(s.PeakHeapMB))

	b.WriteString("\n")

	if s.NumGC == 0 {
		b.WriteString(styles.neutral.Render("No GC activity was observed (the suite may be too small or have no tests)."))
		b.WriteString("\n")
	}

	b.WriteString("\n" + strings.Repeat("-", 60) + "\n")
	b.WriteString(styles.neutral.Render("Raw gctrace output"))
	b.WriteString("\n\n")
	b.WriteString(res.Raw)

	return b.String()
}

// helpContent returns a concise glossary explaining each metric in the report.
func helpContent() string {
	styles := defaultStyles()
	var b strings.Builder

	b.WriteString(styles.packageHeader.Render("What these metrics mean"))
	b.WriteString("\n\n")
	b.WriteString(styles.normalDesc.Render("Lower is better for every metric."))
	b.WriteString("\n\n")

	writeHelpEntry(&b, styles, "Wall time",
		"Total real time to run your whole test suite. The headline number, but the",
		"noisiest one: on short runs a single measurement swings easily, so raise the",
		"iteration count if a number looks suspicious.")
	writeHelpEntry(&b, styles, "GC cycles",
		"How many garbage-collection cycles the runtime ran. Fewer usually means less GC",
		"churn, but it is pacing-dependent.")
	writeHelpEntry(&b, styles, "Total GC pause",
		"Sum of stop-the-world pauses, i.e. the total time every goroutine was frozen for",
		"GC (concurrent marking is excluded). This is the latency-relevant number.")
	writeHelpEntry(&b, styles, "GC CPU",
		"Share of CPU spent in GC instead of your actual work, as reported by the runtime.",
		"A dash means it rounded to 0%.")
	writeHelpEntry(&b, styles, "Peak heap",
		"Largest heap size observed during a GC cycle. A proxy for memory footprint.")

	b.WriteString(styles.normalTitle.Render("  How it works"))
	b.WriteString("\n")
	writeHelpLines(&b, styles,
		"Lacune runs your test suite once under the GC that's the default for your Go",
		"toolchain (Green Tea on Go 1.26+, classic before that; the label is taken from",
		"your go.mod version). It warms the build cache first, so wall time reflects test",
		"execution, not compilation, and runs under GODEBUG=gctrace=1 so the runtime",
		"reports the GC numbers above.")
	b.WriteString("\n")
	writeHelpLines(&b, styles,
		"Iterations (set with +/- before running) maps to `go test -count`.",
		"A higher count re-runs every test that many times, which busts the test",
		"cache, lengthens the GC-active window, and averages out wall-time noise",
		"- at the cost of proportionally longer runs.")
	b.WriteString("\n")

	b.WriteString(styles.neutral.Render("Note: this measures your test suite, not your production workload, so treat the"))
	b.WriteString("\n")
	b.WriteString(styles.neutral.Render("numbers as an approximation of how your code exercises the garbage collector."))

	return b.String()
}

func writeHelpEntry(b *strings.Builder, styles styles, label string, lines ...string) {
	b.WriteString(styles.normalTitle.Render("  " + label))
	b.WriteString("\n")
	writeHelpLines(b, styles, lines...)
	b.WriteString("\n")
}

func writeHelpLines(b *strings.Builder, styles styles, lines ...string) {
	for _, line := range lines {
		b.WriteString(styles.normalDesc.Render("    " + line))
		b.WriteString("\n")
	}
}

func writeBenchRow(b *strings.Builder, label, value string) {
	b.WriteString(fmt.Sprintf("  %-16s %s\n", label, value))
}

func fmtDur(d time.Duration) string {
	if d == 0 {
		return "-"
	}

	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func fmtMs(ms float64) string {
	if ms == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1fms", ms)
}

func fmtPct(p float64) string {
	if p == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", p)
}

func fmtMB(mb int) string {
	if mb == 0 {
		return "-"
	}
	return fmt.Sprintf("%d MB", mb)
}
