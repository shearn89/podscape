// Package picker is a tiny Bubble Tea program that lets the user choose a
// kubeconfig context when one wasn't pre-selected. It's run as its own
// tea.Program before the main app starts so the main model never has to deal
// with an "I don't know what cluster to talk to" state.
package picker

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shearn89/podscape/internal/tui/styles"
)

type Model struct {
	contexts []string
	cursor   int
	chosen   string
	quitted  bool
}

func New(contexts []string) Model {
	return Model{contexts: contexts}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q", "esc":
			m.quitted = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.contexts)-1 {
				m.cursor++
			}
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = len(m.contexts) - 1
		case "enter":
			m.chosen = m.contexts[m.cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	title := styles.Title.Render("podscape — choose a context")
	lines := make([]string, 0, len(m.contexts))
	for i, c := range m.contexts {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(styles.ColorFG)
		if i == m.cursor {
			prefix = "▸ "
			style = style.Foreground(styles.ColorAccent).Bold(true)
		}
		lines = append(lines, style.Render(prefix+c))
	}
	help := styles.Help.Render("↑/↓ move · enter select · q quit")
	return fmt.Sprintf("%s\n\n%s\n\n%s\n", title, lipgloss.JoinVertical(lipgloss.Left, lines...), help)
}

func (m Model) Chosen() string { return m.chosen }
func (m Model) Quit() bool     { return m.quitted }
