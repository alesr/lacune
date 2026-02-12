package tui

import (
	"fmt"
	"strings"

	"github.com/alesr/lacune/internal/coverage"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type lineStatus struct {
	lineNo int
	status coverage.LineStatus
	text   string
}
type viewportModel struct {
	viewport        viewport.Model
	viewportContent string
	scrollPos       int
	lineStatus      []lineStatus
	matchLines      []int
	matchIndex      int
	width           int
	height          int
	filterQuery     string
}

func newViewportModel() viewportModel {
	return viewportModel{
		viewport:   viewport.Model{Width: 0, Height: 0},
		matchIndex: -1,
	}
}

func (viewport viewportModel) update(msg tea.Msg) (viewportModel, tea.Cmd) {
	newModel := viewport
	var cmd tea.Cmd
	newModel.viewport, cmd = newModel.viewport.Update(msg)
	return newModel, cmd
}

func (viewport viewportModel) view() string {
	return viewport.viewport.View()
}

func (viewport viewportModel) setSize(width, height int) viewportModel {
	newModel := viewport
	newModel.viewport.Width = width
	newModel.viewport.Height = height
	newModel.width = width
	newModel.height = height
	return newModel
}

func (viewport viewportModel) renderViewportContent(file coverage.FileModel, filterQuery string) viewportModel {
	newModel := viewport
	newModel.filterQuery = filterQuery
	newModel.matchLines = nil
	newModel.matchIndex = -1
	if len(file.LineInfo) == 0 {
		newModel.viewport.SetContent("No coverage information for this file.")
		return newModel
	}

	var content strings.Builder
	newModel.lineStatus = make([]lineStatus, len(file.LineInfo))
	lowerQuery := strings.ToLower(filterQuery)

	for i, line := range file.LineInfo {
		lineText := line.Text
		if filterQuery != "" {
			lineText = highlightLine(lineText, filterQuery)
		}

		if filterQuery != "" && strings.Contains(strings.ToLower(line.Text), lowerQuery) {
			newModel.matchLines = append(newModel.matchLines, i)
		}

		formattedLine := fmt.Sprintf("%4d %s %s", line.LineNo, statusSymbol(line.Status), lineText)
		if newModel.width > 0 {
			formattedLine = ansi.TruncateWc(formattedLine, newModel.width, "")
		}

		content.WriteString(formattedLine + "\n")
		newModel.lineStatus[i] = lineStatus{
			lineNo: line.LineNo,
			status: line.Status,
			text:   line.Text,
		}
	}
	newModel.viewport.SetContent(content.String())
	if filterQuery != "" && len(newModel.matchLines) > 0 {
		newModel = newModel.jumpToMatch(0)
	}
	return newModel
}

// scrolls to the match at the given index
func (viewport viewportModel) jumpToMatch(index int) viewportModel {
	if len(viewport.matchLines) == 0 {
		return viewport
	}

	newModel := viewport
	if index < 0 {
		index = len(newModel.matchLines) - 1
	}
	if index >= len(newModel.matchLines) {
		index = 0
	}

	newModel.matchIndex = index
	newModel.viewport.SetYOffset(newModel.matchLines[index])
	return newModel
}

func (viewport viewportModel) nextMatch() viewportModel {
	return viewport.jumpToMatch(viewport.matchIndex + 1)
}

func (viewport viewportModel) prevMatch() viewportModel {
	return viewport.jumpToMatch(viewport.matchIndex - 1)
}

func (viewport viewportModel) scrollUp(lines int) viewportModel {
	newModel := viewport
	newModel.viewport.ScrollUp(lines)
	return newModel
}

func (viewport viewportModel) scrollDown(lines int) viewportModel {
	newModel := viewport
	newModel.viewport.ScrollDown(lines)
	return newModel
}
