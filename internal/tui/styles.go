package tui

import (
	"github.com/alesr/lacune/internal/coverage"
	"github.com/charmbracelet/lipgloss"
)

const (
	colorCovered        = "#06B6D4" // cyan accent
	colorCoveredBg      = "#0A3D45" // dark cyan tint for covered lines
	colorUncovered      = "252"
	colorUncoveredBg    = "#3D1515" // dark red tint (pairs with colorError)
	colorPartialBg      = "#3D3520" // dark amber for partial branch coverage
	colorError          = "9"
	colorHighlight      = "#FAFAFA"
	colorBorder         = "#06B6D4"
	colorBorderInactive = "240"
	colorNeutral        = "240" // gray: secondary/explanatory text

	statusMsgNone    = "No coverage data available"
	statusMsgRerun   = "Rerunning tests..."
	statusMsgSuccess = "Tests rerun successfully"
	statusMsgError   = "Error rerunning tests: %v"
)

type styles struct {
	lineCovered    lipgloss.Style
	lineUncovered  lipgloss.Style
	linePartial    lipgloss.Style
	linePlain      lipgloss.Style
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
	neutral        lipgloss.Style
}

func (s styles) forLineStatus(status coverage.LineStatus) lipgloss.Style {
	switch status {
	case coverage.Covered:
		return s.lineCovered
	case coverage.Uncovered:
		return s.lineUncovered
	case coverage.Partial:
		return s.linePartial
	default:
		return s.linePlain
	}
}

func defaultStyles() styles {
	return styles{
		lineCovered:    lipgloss.NewStyle().Background(lipgloss.Color(colorCoveredBg)).Foreground(lipgloss.Color(colorUncovered)),
		lineUncovered:  lipgloss.NewStyle().Background(lipgloss.Color(colorUncoveredBg)).Foreground(lipgloss.Color(colorUncovered)),
		linePartial:    lipgloss.NewStyle().Background(lipgloss.Color(colorPartialBg)).Foreground(lipgloss.Color(colorUncovered)),
		linePlain:      lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
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
		neutral:        lipgloss.NewStyle().Foreground(lipgloss.Color(colorNeutral)),
	}
}
