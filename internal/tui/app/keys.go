package app

import "github.com/charmbracelet/bubbles/key"

// keyMap holds every binding the app exposes. Centralising it lets the help
// bubble pick them up automatically and means rebinding is a one-line change.
type keyMap struct {
	Quit        key.Binding
	Refresh     key.Binding
	Help        key.Binding
	TabPlan     key.Binding
	TabTable    key.Binding
	TabFindings key.Binding
	NextTab     key.Binding
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Enter       key.Binding
	Esc         key.Binding
	Compact     key.Binding
	Normal      key.Binding
	Wide        key.Binding
	SortNext    key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		TabPlan:     key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "floor plan")),
		TabTable:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "nodes table")),
		TabFindings: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "findings")),
		NextTab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch tab")),
		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:        key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
		Right:       key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
		Enter:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open detail")),
		Esc:         key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close detail")),
		Compact:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "compact")),
		Normal:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "normal")),
		Wide:        key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "wide")),
		SortNext:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort column")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextTab, k.Enter, k.Refresh, k.Compact, k.Normal, k.Wide, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.TabPlan, k.TabTable, k.TabFindings, k.NextTab},
		{k.Up, k.Down, k.Left, k.Right, k.Enter, k.Esc},
		{k.Compact, k.Normal, k.Wide, k.SortNext},
		{k.Refresh, k.Help, k.Quit},
	}
}
