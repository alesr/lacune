// Package tui implements the terminal user interface for Lacune.
package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alesr/lacune/internal/coverage"
	"github.com/alesr/lacune/pkg/gcbench"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	footer              = "[↑/↓] navigate  [tab] switch focus  [r] rerun  [d] details  [b] bench  [q] quit  [/] filter"
	footerDetails       = "[↑/↓] scroll  [d] close  [q] quit"
	footerBenchPicker   = "[+/-] iterations  [t] docker  [enter] run  [esc] close"
	footerBenchRunning  = "[esc] cancel"
	footerBenchResults  = "[↑/↓] scroll  [?] explain  [b/esc] close"
	moduleNameUnknown   = "unknown"
	fileListWidthRatio  = 0.3
	headerFooterHeight  = 6
	borderWidth         = 2
	loadingTickInterval = 120 * time.Millisecond

	LoadStageTest  loadStage = "test"
	LoadStageParse loadStage = "parse"
	LoadStageMin   loadStage = "min"
)

var loadingFrames = []string{"|", "/", "-", "\\"}

type (
	loadStage string

	LoadDiagnostics struct {
		Stdout string
		Stderr string
		Stage  loadStage
	}

	statusMsgMsg struct{ msg string }

	coverageUpdateMsg struct {
		files  []coverage.FileModel
		totals coverage.Totals
	}

	loadSuccessMsg struct {
		files  []coverage.FileModel
		totals coverage.Totals
	}

	loadErrorMsg struct {
		err         error
		diagnostics LoadDiagnostics
	}

	loadingTickMsg struct{}

	benchEvent struct {
		stage  string
		result *gcbench.Result
		err    error
	}

	benchEventMsg benchEvent

	benchTickMsg struct{}
)

// BenchRunner runs a GC comparison with the given iteration count, optionally
// reporting progress via onProgress. It is injected so the TUI stays decoupled
// from exec/docker concerns.
type BenchRunner func(ctx context.Context, useDocker bool, count int, onProgress func(string)) (gcbench.Result, error)

type Model struct {
	fileList    *fileListModel
	viewport    viewportModel
	header      HeaderModel
	keybindings keybindingModel
	focus       FocusModel
	packageView packageViewModel
	benchView   benchViewModel
	files       []coverage.FileModel
	currentFile int
	moduleName  string
	packageName string
	filterQuery string
	rerunFunc   func() ([]coverage.FileModel, coverage.Totals, error)
	benchFunc   BenchRunner
	benchChan   chan benchEvent
	benchCancel context.CancelFunc
	benchTick   int
	statusMsg   string
	loading     bool
	loader      func() ([]coverage.FileModel, coverage.Totals, LoadDiagnostics, error)
	loadErr     error
	loadDiag    LoadDiagnostics
	loadingTick int
	width       int
	height      int
}

func NewModel(files []coverage.FileModel, totals coverage.Totals, rerunFunc func() ([]coverage.FileModel, coverage.Totals, error), benchFunc BenchRunner) Model {
	moduleName, err := getModuleName()
	if err != nil {
		moduleName = moduleNameUnknown
	}
	packageName := moduleName
	if packageName == moduleNameUnknown || packageName == "" {
		packageName = extractPackageName(files)
	}

	keys := defaultKeyMap()
	fileList := newFileListModel(files, keys)
	viewport := newViewportModel()
	header := newHeaderModel(moduleName, totals)
	keybindings := newKeybindingModel(keys)
	focus := newFocusModel()
	packageView := newPackageViewModel()
	benchView := newBenchViewModel()

	if len(files) > 0 {
		fileList = fileList.Select(0)
		viewport = viewport.renderViewportContent(files[0], "")
	}
	return Model{
		fileList:    fileList,
		viewport:    viewport,
		header:      header,
		keybindings: keybindings,
		focus:       focus,
		packageView: packageView,
		benchView:   benchView,
		files:       files,
		currentFile: 0,
		moduleName:  moduleName,
		packageName: packageName,
		filterQuery: "",
		rerunFunc:   rerunFunc,
		benchFunc:   benchFunc,
		statusMsg:   "",
	}
}

func (model Model) Init() tea.Cmd {
	return model.InitLoading()
}

func NewLoadingModel(
	loader func() ([]coverage.FileModel, coverage.Totals, LoadDiagnostics, error),
	rerunFunc func() ([]coverage.FileModel, coverage.Totals, error),
	benchFunc BenchRunner,
) Model {
	model := NewModel(nil, coverage.Totals{}, rerunFunc, benchFunc)
	model.loading = true
	model.loader = loader
	return model
}

func (model Model) InitLoading() tea.Cmd {
	if !model.loading || model.loader == nil {
		return nil
	}
	return tea.Batch(model.runLoader(), loadingTick())
}

func (model Model) rerunTests() tea.Cmd {
	return func() tea.Msg {
		files, totals, err := model.rerunFunc()
		if err != nil {
			return statusMsgMsg{fmt.Sprintf(statusMsgError, err)}
		}
		return coverageUpdateMsg{files, totals}
	}
}

func (model Model) runLoader() tea.Cmd {
	return func() tea.Msg {
		files, totals, diagnostics, err := model.loader()
		if err != nil {
			return loadErrorMsg{err: err, diagnostics: diagnostics}
		}
		return loadSuccessMsg{files: files, totals: totals}
	}
}

func loadingTick() tea.Cmd {
	return tea.Tick(loadingTickInterval, func(time.Time) tea.Msg {
		return loadingTickMsg{}
	})
}

func benchTick() tea.Cmd {
	return tea.Tick(loadingTickInterval, func(time.Time) tea.Msg {
		return benchTickMsg{}
	})
}

// startBench launches the selected benchmark in a goroutine, streaming progress
// and the final result/error through a channel consumed by waitForBench.
func (model Model) startBench() (Model, tea.Cmd) {
	useDocker := model.benchView.UseDocker()
	count := model.benchView.Count()
	benchFunc := model.benchFunc

	ch := make(chan benchEvent, 8)
	ctx, cancel := context.WithCancel(context.Background())

	model.benchChan = ch
	model.benchCancel = cancel
	model.benchTick = 0
	model.benchView = model.benchView.SetRunning("Starting benchmark...")

	go func() {
		res, err := benchFunc(ctx, useDocker, count, func(stage string) {
			ch <- benchEvent{stage: stage}
		})
		if err != nil {
			ch <- benchEvent{err: err}
		} else {
			ch <- benchEvent{result: &res}
		}
		close(ch)
	}()

	return model, tea.Batch(waitForBench(ch), benchTick())
}

func waitForBench(ch chan benchEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return benchEventMsg(ev)
	}
}

func (model Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if model.benchView.IsOpen() {
		return model.handleBenchKey(msg)
	}

	cmd, newFocus := model.keybindings.HandleKeyMsg(msg, model.focus.getFocus())
	if key.Matches(msg, model.keybindings.KeyMap().quit) {
		return model, tea.Quit
	}

	if model.packageView.IsOpen() {
		switch {
		case key.Matches(msg, model.keybindings.KeyMap().details):
			model.packageView = model.packageView.Close()
			return model, nil
		case key.Matches(msg, model.keybindings.KeyMap().up):
			model.packageView = model.packageView.ScrollUp(1)
			return model, nil
		case key.Matches(msg, model.keybindings.KeyMap().down):
			model.packageView = model.packageView.ScrollDown(1)
			return model, nil
		}
		return model, nil
	}

	model.focus = model.focus.setFocus(newFocus)
	model.statusMsg = ""

	if model.fileList.SettingFilter() {
		switch {
		case key.Matches(msg, model.keybindings.KeyMap().up):
			model.viewport = model.viewport.prevMatch()
			return model, nil
		case key.Matches(msg, model.keybindings.KeyMap().down):
			model.viewport = model.viewport.nextMatch()
			return model, nil
		case key.Matches(msg, model.keybindings.KeyMap().details):
			return model, nil
		}
	}

	// viewport scrolling
	switch {
	case key.Matches(msg, model.keybindings.KeyMap().up):
		if model.focus.getFocus() == focusViewport {
			model.viewport = model.viewport.scrollUp(3)
		}
	case key.Matches(msg, model.keybindings.KeyMap().down):
		if model.focus.getFocus() == focusViewport {
			model.viewport = model.viewport.scrollDown(3)
		}
	}

	if key.Matches(msg, model.keybindings.KeyMap().filter) {
		model.focus = model.focus.setFocus(focusFileList)
		return model, nil
	}

	if key.Matches(msg, model.keybindings.KeyMap().details) {
		content := buildPackageCoverageContent(model.files, model.moduleName, model.packageView.width)
		model.packageView = model.packageView.Open(content)
		return model, nil
	}

	if key.Matches(msg, model.keybindings.KeyMap().bench) {
		if model.benchFunc != nil {
			model.benchView = model.benchView.Open()
		}
		return model, nil
	}

	if key.Matches(msg, model.keybindings.KeyMap().rerun) {
		if model.rerunFunc != nil {
			model.statusMsg = statusMsgRerun
			return model, model.rerunTests()
		}
		return model, nil
	}
	return model, cmd
}

// handleBenchKey routes key input while the benchmark overlay is open. It fully
// owns input (consuming all keys) so the overlay behaves like a modal.
func (model Model) handleBenchKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	keys := model.keybindings.KeyMap()

	switch model.benchView.State() {
	case benchReady:
		switch {
		case key.Matches(msg, keys.incIters):
			model.benchView = model.benchView.IncCount()
		case key.Matches(msg, keys.decIters):
			model.benchView = model.benchView.DecCount()
		case key.Matches(msg, keys.dockerToggle):
			model.benchView = model.benchView.ToggleDocker()
		case key.Matches(msg, keys.confirm):
			if model.benchFunc != nil {
				return model.startBench()
			}
		case key.Matches(msg, keys.bench), key.Matches(msg, keys.quit):
			model.benchView = model.benchView.Close()
		}
		return model, nil
	case benchRunning:
		if key.Matches(msg, keys.quit) {
			if model.benchCancel != nil {
				model.benchCancel()
				model.benchCancel = nil
			}
			model.benchChan = nil
			model.benchView = model.benchView.Open()
		}
		return model, nil
	default: // benchResults, benchError
		switch {
		case key.Matches(msg, keys.help) && model.benchView.State() == benchResults:
			model.benchView = model.benchView.ToggleHelp()
		case key.Matches(msg, keys.up):
			model.benchView = model.benchView.ScrollUp(1)
		case key.Matches(msg, keys.down):
			model.benchView = model.benchView.ScrollDown(1)
		case key.Matches(msg, keys.bench), key.Matches(msg, keys.quit):
			model.benchView = model.benchView.Close()
		}
		return model, nil
	}
}

func (model Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	model.width = msg.Width
	model.height = msg.Height
	if model.loading {
		return model, nil
	}
	return model.resize(msg.Width, msg.Height), nil
}

func (model Model) handleCoverageUpdateMsg(msg coverageUpdateMsg) (Model, tea.Cmd) {
	model.files = msg.files
	model.header = model.header.SetTotals(msg.totals)
	model.statusMsg = statusMsgSuccess
	model.header = model.header.SetStatus(statusMsgSuccess)

	model.fileList = model.fileList.SetItems(msg.files)
	model.fileList = model.fileList.Select(0)
	model.currentFile = 0
	if len(msg.files) > 0 {
		model.viewport = model.viewport.renderViewportContent(msg.files[0], model.filterQuery)
	}
	return model, nil
}

func (model Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd            tea.Cmd
		cmds           []tea.Cmd
		skipListUpdate bool
	)

	prevFilterQuery := model.filterQuery
	prevCurrentFile := model.currentFile

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if model.loading {
			if key.Matches(msg, model.keybindings.KeyMap().quit) {
				return model, tea.Quit
			}
			return model, nil
		}

		var keyCmd tea.Cmd
		model, keyCmd = model.handleKeyMsg(msg)

		if key.Matches(msg, model.keybindings.KeyMap().filter) {
			updatedFileList, cmd := model.fileList.Update(msg)
			model.fileList = updatedFileList.(*fileListModel)
			cmds = append(cmds, cmd)
			skipListUpdate = true
			model.focus = model.focus.setFocus(focusFileList)
		}

		if model.fileList.SettingFilter() && (key.Matches(msg, model.keybindings.KeyMap().up) || key.Matches(msg, model.keybindings.KeyMap().down)) {
			skipListUpdate = true
		}

		if key.Matches(msg, model.keybindings.KeyMap().quit) {
			return model, keyCmd
		}
		if keyCmd != nil {
			cmds = append(cmds, keyCmd)
		}
	case tea.WindowSizeMsg:
		return model.handleWindowSizeMsg(msg)
	case statusMsgMsg:
		model.statusMsg = msg.msg
		model.header = model.header.SetStatus(msg.msg)
		return model, nil
	case loadSuccessMsg:
		model.loading = false
		model.files = msg.files
		model.header = model.header.SetTotals(msg.totals)
		model.header = model.header.SetStatus("")
		model.statusMsg = ""
		model.fileList = model.fileList.SetItems(msg.files)
		model.fileList = model.fileList.Select(0)
		model.currentFile = 0

		if len(msg.files) > 0 {
			model.viewport = model.viewport.renderViewportContent(msg.files[0], model.filterQuery)
		}

		if model.width > 0 && model.height > 0 {
			model = model.resize(model.width, model.height)
		}
		return model, nil
	case loadErrorMsg:
		model.loadErr = msg.err
		model.loadDiag = msg.diagnostics
		return model, tea.Quit
	case loadingTickMsg:
		if model.loading {
			model.loadingTick = (model.loadingTick + 1) % len(loadingFrames)
			return model, loadingTick()
		}
		return model, nil
	case coverageUpdateMsg:
		model, cmd := model.handleCoverageUpdateMsg(msg)
		if model.packageView.IsOpen() {
			content := buildPackageCoverageContent(model.files, model.moduleName, model.packageView.width)
			model.packageView = model.packageView.SetContent(content)
		}
		return model, cmd
	case benchEventMsg:
		if model.benchView.State() != benchRunning {
			return model, nil
		}
		switch {
		case msg.err != nil:
			model.benchChan = nil
			model.benchCancel = nil
			model.benchView = model.benchView.SetError(msg.err)
			return model, nil
		case msg.result != nil:
			model.benchView = model.benchView.SetResult(buildBenchContent(*msg.result))
			model.benchChan = nil
			model.benchCancel = nil
			return model, nil
		default:
			model.benchView = model.benchView.SetProgress(msg.stage)
			return model, waitForBench(model.benchChan)
		}
	case benchTickMsg:
		if model.benchView.IsOpen() && model.benchView.State() == benchRunning {
			model.benchTick = (model.benchTick + 1) % len(loadingFrames)
			return model, benchTick()
		}
		return model, nil
	}

	if model.packageView.IsOpen() || model.benchView.IsOpen() {
		return model, tea.Batch(cmds...)
	}

	// update focused component
	switch model.focus.getFocus() {
	case focusFileList:
		if !skipListUpdate {
			updatedFileList, cmd := model.fileList.Update(msg)
			model.fileList = updatedFileList.(*fileListModel)
			cmds = append(cmds, cmd)
		}
	case focusViewport:
		if model.fileList.SettingFilter() {
			if !skipListUpdate {
				updatedFileList, cmd := model.fileList.Update(msg)
				model.fileList = updatedFileList.(*fileListModel)
				cmds = append(cmds, cmd)
			}
		} else {
			model.viewport, cmd = model.viewport.update(msg)
			cmds = append(cmds, cmd)
		}
	}

	newFilterQuery := model.fileList.FilterValue()
	selectedIndex := -1
	if selected, ok := model.fileList.SelectedFile(); ok {
		selectedIndex = findFileIndex(model.files, selected.FilePath)
	}
	if selectedIndex != -1 && selectedIndex < len(model.files) {
		if selectedIndex != model.currentFile {
			model.currentFile = selectedIndex
		}
		if selectedIndex != prevCurrentFile || newFilterQuery != prevFilterQuery {
			model.viewport = model.viewport.renderViewportContent(model.files[selectedIndex], newFilterQuery)
		}
	}

	model.filterQuery = newFilterQuery
	return model, tea.Batch(cmds...)
}

func (model Model) View() string {
	if model.loading {
		return model.loadingView()
	}

	if model.benchView.IsOpen() {
		return model.benchOverlayView()
	}

	if len(model.files) == 0 {
		return statusMsgNone
	}

	var currentFile coverage.FileModel
	if model.currentFile >= 0 && model.currentFile < len(model.files) {
		currentFile = model.files[model.currentFile]
	} else {
		currentFile = coverage.FileModel{}
	}

	headerView := model.header.View(model.packageName, currentFile)

	fileListView := model.fileList.View()
	viewportView := model.viewport.view()
	if model.packageView.IsOpen() {
		packageView := model.packageView.View()
		packageBorderStyle := lipgloss.NewStyle().Width(model.packageView.width + borderWidth)
		packageBorderStyle = applyBorder(packageBorderStyle, true)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			headerView,
			packageBorderStyle.Render(packageView),
			footerDetails,
		)
	}

	fileListBorderStyle := lipgloss.NewStyle().Width(model.fileList.width + borderWidth)
	viewportBorderStyle := lipgloss.NewStyle().Width(model.viewport.width + borderWidth)

	fileListBorderStyle = applyBorder(fileListBorderStyle, model.focus.getFocus() == focusFileList)
	viewportBorderStyle = applyBorder(viewportBorderStyle, model.focus.getFocus() == focusViewport)

	panes := lipgloss.JoinHorizontal(
		lipgloss.Top,
		fileListBorderStyle.Render(fileListView),
		viewportBorderStyle.Render(viewportView),
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerView,
		panes,
		footer,
	)
}

func (model Model) benchOverlayView() string {
	var currentFile coverage.FileModel
	if model.currentFile >= 0 && model.currentFile < len(model.files) {
		currentFile = model.files[model.currentFile]
	}
	headerView := model.header.View(model.packageName, currentFile)

	frame := loadingFrames[model.benchTick%len(loadingFrames)]
	benchView := model.benchView.View(frame)

	benchBorderStyle := lipgloss.NewStyle().Width(model.benchView.width + borderWidth)
	benchBorderStyle = applyBorder(benchBorderStyle, true)

	foot := footerBenchPicker
	switch model.benchView.State() {
	case benchRunning:
		foot = footerBenchRunning
	case benchResults, benchError:
		foot = footerBenchResults
	case benchReady:
		foot = footerBenchPicker
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerView,
		benchBorderStyle.Render(benchView),
		foot,
	)
}

func (model Model) loadingView() string {
	styles := defaultStyles()
	frame := loadingFrames[model.loadingTick%len(loadingFrames)]
	title := styles.packageHeader.Render("Lacune")
	body := styles.normalDesc.Render(fmt.Sprintf("Running tests... %s", frame))
	content := lipgloss.JoinVertical(lipgloss.Center, title, body)
	if model.width > 0 && model.height > 0 {
		return lipgloss.Place(model.width, model.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

// resize updates the model dimensions
func (model Model) resize(width, height int) Model {
	fileListWidth := int(float64(width) * fileListWidthRatio)
	viewportWidth := width - fileListWidth
	contentHeight := max(height-headerFooterHeight-borderWidth, 0)

	fileListContentWidth := max(fileListWidth-borderWidth, 0)
	viewportContentWidth := max(viewportWidth-borderWidth, 0)

	newModel := model
	newModel.fileList = newModel.fileList.SetSize(fileListContentWidth, contentHeight)
	newModel.viewport = newModel.viewport.setSize(viewportContentWidth, contentHeight)
	newModel.packageView = newModel.packageView.SetSize(width-borderWidth, contentHeight)
	newModel.benchView = newModel.benchView.SetSize(width-borderWidth, contentHeight)
	return newModel
}

func Run(files []coverage.FileModel, totals coverage.Totals, rerunFunc func() ([]coverage.FileModel, coverage.Totals, error), benchFunc BenchRunner) error {
	model := NewModel(files, totals, rerunFunc, benchFunc)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("could not run TUI: %w", err)
	}
	return nil
}

func RunWithLoader(loader func() ([]coverage.FileModel, coverage.Totals, LoadDiagnostics, error), rerunFunc func() ([]coverage.FileModel, coverage.Totals, error), benchFunc BenchRunner) error {
	model := NewLoadingModel(loader, rerunFunc, benchFunc)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("could not run TUI: %w", err)
	}
	if m, ok := finalModel.(Model); ok {
		if m.loadErr != nil {
			return LoadError{Err: m.loadErr, Diagnostics: m.loadDiag}
		}
	}
	return nil
}

// getModuleName reads the module name from go.mod
func getModuleName() (string, error) {
	modFile, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	modFileContent := string(modFile)
	lines := strings.SplitSeq(modFileContent, "\n")

	for line := range lines {
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", fmt.Errorf("module name not found in go.mod")
}

// replaces the common prefix (module name) with "..."
func formatTruncatedPath(path string) string {
	moduleName, err := getModuleName()
	if err != nil {
		return path // Fallback to full path if module name cannot be read
	}

	// ensure module name ends with a slash for prefix matching
	prefix := moduleName + "/"
	if strings.HasPrefix(path, prefix) {
		return "..." + path[len(prefix)-1:] // keep trailing slash
	}
	return path
}

// extracts the package name ("github.com/alesr/lacune") from the file paths
func extractPackageName(files []coverage.FileModel) string {
	if len(files) == 0 {
		return ""
	}

	// assume all files share the same prefix (e.g., "github.com/alesr/lacune/...")
	firstFile := files[0].FilePath
	lastSlash := strings.LastIndex(firstFile, "/")
	if lastSlash == -1 {
		return firstFile
	}
	return firstFile[:lastSlash]
}

func highlightLine(lineText, query string) string {
	if query == "" {
		return lineText
	}

	lowerLine := strings.ToLower(lineText)
	lowerQuery := strings.ToLower(query)

	var (
		result    strings.Builder
		lastIndex int
		styles    = defaultStyles()
	)

	for i := 0; i < len(lowerLine); i++ {
		if strings.HasPrefix(lowerLine[i:], lowerQuery) {
			result.WriteString(lineText[lastIndex:i]) // text before match
			match := lineText[i : i+len(query)]       // matched text with highlight

			result.WriteString(styles.highlight.Render(match))

			i += len(query) - 1
			lastIndex = i + 1
		}
	}
	result.WriteString(lineText[lastIndex:]) // remaining text
	return result.String()
}

// findFileIndex finds the index of a file by path.
func findFileIndex(files []coverage.FileModel, path string) int {
	for i, file := range files {
		if file.FilePath == path {
			return i
		}
	}
	return -1
}

func applyBorder(style lipgloss.Style, active bool) lipgloss.Style {
	styles := defaultStyles()
	if active {
		return styles.border.Inherit(style)
	}
	return styles.borderInactive.Inherit(style)
}
