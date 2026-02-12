package tui

import "github.com/charmbracelet/lipgloss"

const (
	colorCovered        = "#7D56F4"
	colorUncovered      = "252"
	colorError          = "9"
	colorHighlight      = "#FAFAFA"
	colorBorder         = "#7D56F4"
	colorBorderInactive = "240"

	statusMsgNone    = "No coverage data available"
	statusMsgRerun   = "Rerunning tests..."
	statusMsgSuccess = "Tests rerun successfully"
	statusMsgError   = "Error rerunning tests: %v"
)

type styles struct {
	covered        lipgloss.Style
	uncovered      lipgloss.Style
	error          lipgloss.Style
	highlight      lipgloss.Style
	border         lipgloss.Style
	borderInactive lipgloss.Style
	packageHeader  lipgloss.Style
	header         lipgloss.Style
	normalTitle    lipgloss.Style
	normalDesc     lipgloss.Style
	selectedTitle  lipgloss.Style
	selectedDesc   lipgloss.Style
}

func defaultStyles() styles {
	return styles{
		covered:        lipgloss.NewStyle().Foreground(lipgloss.Color(colorCovered)),
		uncovered:      lipgloss.NewStyle().Foreground(lipgloss.Color(colorUncovered)),
		error:          lipgloss.NewStyle().Foreground(lipgloss.Color(colorError)),
		highlight:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorHighlight)).Background(lipgloss.Color(colorCovered)),
		border:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(colorBorder)),
		borderInactive: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(colorBorderInactive)),
		packageHeader:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorCovered)),
		header:         lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		normalTitle:    lipgloss.NewStyle().Foreground(lipgloss.Color(colorUncovered)),
		normalDesc:     lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		selectedTitle:  lipgloss.NewStyle().Foreground(lipgloss.Color(colorCovered)).Bold(true),
		selectedDesc:   lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
	}
}
