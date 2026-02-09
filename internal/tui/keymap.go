package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Tab   key.Binding
	Rerun key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Rerun}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab},
		{k.Rerun},
	}
}

func defaultKeyMap() keyMap {
	return keyMap{
		Tab: key.NewBinding(
			key.WithKeys("tab"), // Tab
			key.WithHelp("tab", "switch focus"),
		),
		Rerun: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rerun tests"),
		),
	}
}
