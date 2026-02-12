package tui

import (
	"fmt"

	"github.com/alesr/lacune/internal/coverage"
	"github.com/charmbracelet/lipgloss"
)

type MessageType int

const (
	MessageTypeStatus MessageType = iota
	MessageTypeError
)

type HeaderModel struct {
	moduleName  string
	totals      coverage.Totals
	message     string
	messageType MessageType
}

func newHeaderModel(moduleName string, totals coverage.Totals) HeaderModel {
	return HeaderModel{
		moduleName: moduleName,
		totals:     totals,
	}
}

func (headerModel HeaderModel) View(packageName string, currentFile coverage.FileModel) string {
	styles := defaultStyles()

	if headerModel.messageType == MessageTypeError {
		return styles.error.Render(headerModel.message)
	}

	packageHeader := styles.packageHeader.Render(
		fmt.Sprintf("Package: %s", packageName),
	)

	headerText := fmt.Sprintf(
		"Total Coverage: %.2f%% | File Coverage: %.2f%%",
		headerModel.totals.Percent,
		currentFile.Percent,
	)

	if headerModel.messageType == MessageTypeStatus && headerModel.message != "" {
		headerText += fmt.Sprintf(" | Status: %s", headerModel.message)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		packageHeader,
		styles.normalDesc.Render(headerText),
	)
}

func (headerModel HeaderModel) SetStatus(msg string) HeaderModel {
	newModel := headerModel
	newModel.message = msg
	newModel.messageType = MessageTypeStatus
	return newModel
}

func (headerModel HeaderModel) SetError(err error) HeaderModel {
	newModel := headerModel
	newModel.message = fmt.Sprintf("Error: %v", err)
	newModel.messageType = MessageTypeError
	return newModel
}

func (headerModel HeaderModel) SetTotals(totals coverage.Totals) HeaderModel {
	newModel := headerModel
	newModel.totals = totals
	return newModel
}
