// Package styles holds the Lipgloss theme used across the TUI: borders,
// foreground colours, focus states. Keeping it in one place means a single
// edit re-themes the whole app.
package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Foundational palette.
	ColorBG      = lipgloss.Color("#11131a")
	ColorFG      = lipgloss.Color("#e5e9f0")
	ColorMuted   = lipgloss.Color("#5e6779")
	ColorBorder  = lipgloss.Color("#3b4252")
	ColorAccent  = lipgloss.Color("#88c0d0")
	ColorWarn    = lipgloss.Color("#ebcb8b")
	ColorDanger  = lipgloss.Color("#bf616a")
	ColorGroupFG = lipgloss.Color("#d08770")

	// Group "region" — the outer red-ish border in the sketch. Renders in a
	// warmer hue to read as a container, not a card.
	GroupBorder = lipgloss.Border{
		Top: "═", Bottom: "═", Left: "║", Right: "║",
		TopLeft: "╔", TopRight: "╗", BottomLeft: "╚", BottomRight: "╝",
	}
	Group = lipgloss.NewStyle().
		Border(GroupBorder).
		BorderForeground(ColorGroupFG).
		Padding(0, 1)

	GroupHeader = lipgloss.NewStyle().
			Foreground(ColorGroupFG).
			Bold(true)

	// Node card — the inner black-bordered cards holding pods.
	NodeBorder = lipgloss.RoundedBorder()
	Node       = lipgloss.NewStyle().
			Border(NodeBorder).
			BorderForeground(ColorBorder).
			Padding(0, 1)
	NodeFocused = Node.BorderForeground(ColorAccent)

	NodeHeader = lipgloss.NewStyle().
			Foreground(ColorFG).
			Bold(true)
	NodeSub = lipgloss.NewStyle().
		Foreground(ColorMuted)

	// Pod chip — coloured block representing a pod.
	PodChip = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#11131a")).
		Padding(0, 1)

	// DaemonSet band divider.
	DSBand = lipgloss.NewStyle().
		Foreground(ColorMuted)

	// Help / status bar.
	Help = lipgloss.NewStyle().Foreground(ColorMuted)

	// Title bar at the top of the app.
	Title = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		Padding(0, 1)
)
