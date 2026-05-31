package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	up           key.Binding
	down         key.Binding
	tab          key.Binding
	rerun        key.Binding
	quit         key.Binding
	filter       key.Binding
	details      key.Binding
	bench        key.Binding
	confirm      key.Binding
	dockerToggle key.Binding
	help         key.Binding
	incIters     key.Binding
	decIters     key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.tab, k.rerun, k.details, k.bench, k.quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.up, k.down},
		{k.tab},
		{k.rerun},
		{k.details, k.bench, k.filter, k.quit},
	}
}

func defaultKeyMap() keyMap {
	return keyMap{
		up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch focus"),
		),
		rerun: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rerun tests"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q/esc", "quit"),
		),
		filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "details"),
		),
		bench: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "benchmark gc"),
		),
		confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "run"),
		),
		dockerToggle: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle docker"),
		),
		help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "explain metrics"),
		),
		incIters: key.NewBinding(
			key.WithKeys("+", "="),
			key.WithHelp("+", "more iterations"),
		),
		decIters: key.NewBinding(
			key.WithKeys("-"),
			key.WithHelp("-", "fewer iterations"),
		),
	}
}
