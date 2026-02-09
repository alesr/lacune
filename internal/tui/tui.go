package tui

import (
	"fmt"
	"strings"

	"github.com/alesr/lacune/internal/coverage"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	focusFileList focusArea = iota
	focusViewport
)

type (
	focusArea         int
	statusMsgMsg      struct{ msg string }
	coverageUpdateMsg struct {
		files  []coverage.FileModel
		totals coverage.Totals
	}
)

func Run(files []coverage.FileModel, totals coverage.Totals, rerunFunc func() ([]coverage.FileModel, coverage.Totals, error)) error {
	m := NewModel(files, totals)
	m.rerunFunc = rerunFunc
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}

type Model struct {
	fileList    list.Model
	viewport    viewport.Model
	help        help.Model
	files       []coverage.FileModel
	currentFile int
	focus       focusArea
	keys        keyMap
	totals      coverage.Totals
	filterQuery string                                                // current search
	rerunFunc   func() ([]coverage.FileModel, coverage.Totals, error) // function to rerun tests
	statusMsg   string
}

func NewModel(files []coverage.FileModel, totals coverage.Totals) Model {
	fileItems := make([]list.Item, len(files))
	for i, file := range files {
		fileItems[i] = fileItem{
			file: file,
		}
	}

	delegate := list.NewDefaultDelegate()
	fileList := list.New(fileItems, delegate, 0, 0)
	fileList.Title = "Files"
	fileList.SetFilteringEnabled(true)
	fileList.SetShowHelp(false)

	viewport := viewport.Model{Width: 0, Height: 0}
	if len(files) == 0 {
		viewport.SetContent("No coverage data available. Run tests to generate coverage.")
	} else {
		viewport.SetContent("Select a file to view coverage details.")
	}
	return Model{
		fileList:    fileList,
		viewport:    viewport,
		help:        help.Model{},
		files:       files,
		currentFile: 0,
		focus:       focusFileList,
		keys:        defaultKeyMap(),
		totals:      totals,
		statusMsg:   "",
		rerunFunc:   nil,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// handle tab key for focus switching (always on)
		if key.Matches(msg, m.keys.Tab) {
			m.focus = toggleFocus(m.focus)
			m.statusMsg = ""
			return m, nil
		}

		// only intercept custom keys when filter is NOT active
		if !m.fileList.SettingFilter() {
			// handle viewport scrolling when viewport is focused
			if m.focus == focusViewport {
				switch msg.String() {
				case "up":
					m.viewport.ScrollUp(3)
				case "down":
					m.viewport.ScrollDown(3)
				}
			}
			switch {
			case msg.String() == "/":
				// let the list handle the / key for filtering
				m.fileList, cmd = m.fileList.Update(msg)
				cmds = append(cmds, cmd)

			case key.Matches(msg, m.keys.Rerun):
				if m.rerunFunc != nil {
					m.statusMsg = "Rerunning tests..."
					return m, m.rerunTests
				}
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case statusMsgMsg:
		m.statusMsg = msg.msg
		return m, nil
	case coverageUpdateMsg:
		m.files = msg.files
		m.totals = msg.totals
		m.statusMsg = "Tests rerun successfully"

		// update file list

		fileItems := make([]list.Item, len(msg.files))
		for i, file := range msg.files {
			fileItems[i] = fileItem{file: file}
		}

		m.fileList.SetItems(fileItems)

		// reset selection
		m.currentFile = 0
		m.fileList.Select(0)
		m.updateViewportContent()
		return m, nil
	}

	// update the focused component
	switch m.focus {
	case focusFileList:
		m.fileList, cmd = m.fileList.Update(msg)
		cmds = append(cmds, cmd)

		// update current file selection and viewport content
		if selected, ok := m.fileList.SelectedItem().(fileItem); ok {
			m.currentFile = findFileIndex(m.files, selected.file.FilePath)
			m.updateViewportContent()
		}

		// track filter query for highlighting
		m.filterQuery = m.fileList.FilterValue()
		m.updateViewportContent()

	case focusViewport:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) rerunTests() tea.Msg {
	files, totals, err := m.rerunFunc()
	if err != nil {
		return statusMsgMsg{fmt.Sprintf("Error rerunning tests: %v", err)}
	}
	return coverageUpdateMsg{files, totals}
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
	)

	for i := 0; i < len(lowerLine); i++ {
		if strings.HasPrefix(lowerLine[i:], lowerQuery) {
			// text before match
			result.WriteString(lineText[lastIndex:i])

			// matched text with highlight

			match := lineText[i : i+len(query)]

			result.WriteString(
				lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#FAFAFA")).
					Background(lipgloss.Color("#7D56F4")).
					Render(match),
			)

			i += len(query) - 1
			lastIndex = i + 1
		}
	}
	// remaining text
	result.WriteString(lineText[lastIndex:])

	return result.String()
}

func (m Model) View() string {
	if len(m.files) == 0 {
		return "No coverage data available."
	}

	header := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf(
			"Total Coverage: %.2f%% | File Coverage: %.2f%% | %s",
			m.totals.Percent, m.files[m.currentFile].Percent, m.statusMsg,
		),
	)
	footer := "[↑/↓] navigate  [tab] switch focus  [r] rerun  [q] quit  [/] filter"

	fileListView := m.fileList.View()
	viewportView := m.viewport.View()

	fileListStyle := lipgloss.NewStyle().Width(m.fileList.Width())
	viewportStyle := lipgloss.NewStyle()

	switch m.focus {
	case focusFileList:
		fileListStyle = makeBorder(fileListStyle)
	case focusViewport:
		viewportStyle = makeBorder(viewportStyle)
	}

	panes := lipgloss.JoinHorizontal(
		lipgloss.Top,
		fileListStyle.Render(fileListView),
		viewportStyle.Render(viewportView),
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		panes,
		footer,
	)
}

func (m *Model) updateViewportContent() {
	if len(m.files) == 0 {
		m.viewport.SetContent("No coverage data available.")
		return
	}
	if m.currentFile >= len(m.files) {
		m.viewport.SetContent("Select a file to view coverage details.")
		return
	}

	file := m.files[m.currentFile]
	if len(file.LineInfo) == 0 {
		m.viewport.SetContent("No coverage information for this file.")
		return
	}

	var content strings.Builder
	for _, line := range file.LineInfo {
		lineText := line.Text
		if m.filterQuery != "" {
			lineText = highlightLine(lineText, m.filterQuery)
		}
		content.WriteString(fmt.Sprintf("%4d %s %s\n", line.LineNo, statusSymbol(line.Status), lineText))
	}
	m.viewport.SetContent(content.String())
}

func (m *Model) resize(width, height int) {
	// 30% file list, 70% viewport
	fileListWidth := int(float64(width) * 0.3)
	viewportWidth := width - fileListWidth - 2 // account for borders

	m.fileList.SetSize(fileListWidth, height-4) // sub header and footer
	m.viewport.Width = viewportWidth
	m.viewport.Height = height - 4

	m.updateViewportContent()
}

var borderBG = "#7D56F4"

func makeBorder(style lipgloss.Style) lipgloss.Style {
	return style.Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(borderBG))
}

func toggleFocus(focus focusArea) focusArea {
	if focus == focusFileList {
		return focusViewport
	}
	return focusFileList
}

func findFileIndex(files []coverage.FileModel, path string) int {
	for i, file := range files {
		if file.FilePath == path {
			return i
		}
	}
	return 0
}

func statusSymbol(status coverage.LineStatus) string {
	switch status {
	case coverage.Covered:
		return "✓"
	case coverage.Uncovered:
		return "!"
	case coverage.Partial:
		return "~"
	default:
		return " "
	}
}
