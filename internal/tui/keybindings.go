package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type keybindingModel struct{ keys keyMap }

func newKeybindingModel(keys keyMap) keybindingModel { return keybindingModel{keys: keys} }
func (m keybindingModel) ShortHelp() []key.Binding   { return m.keys.ShortHelp() }
func (m keybindingModel) FullHelp() [][]key.Binding  { return m.keys.FullHelp() }
func (m keybindingModel) KeyMap() keyMap             { return m.keys }

func toggleFocus(f focusArea) focusArea {
	if f == focusFileList {
		return focusViewport
	}
	return focusFileList
}

func (m keybindingModel) HandleKeyMsg(msg tea.KeyMsg, focus focusArea) (tea.Cmd, focusArea) {
	switch {
	case key.Matches(msg, m.keys.up):
		return nil, focus
	case key.Matches(msg, m.keys.down):
		return nil, focus
	case key.Matches(msg, m.keys.tab):
		return nil, toggleFocus(focus)
	case key.Matches(msg, m.keys.quit):
		return tea.Quit, focus
	case key.Matches(msg, m.keys.rerun):
		return nil, focus
	case key.Matches(msg, m.keys.filter):
		return nil, focus
	case key.Matches(msg, m.keys.details):
		return nil, focus
	}
	return nil, focus
}
