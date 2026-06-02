// Package styles holds the Lipgloss theme used across the TUI: borders,
// foreground colours, focus states. Keeping it in one place means a single
// edit re-themes the whole app.
//
// Colours live in a small, extensible registry of named Themes. Apply swaps the
// active theme — rebuilding the composed style vars below — so the rest of the
// app, which reads these vars at render time, re-themes wholesale. Resolve maps
// a config name ("auto"/"dark"/"light"/…) to a Theme, with "auto" picking dark
// or light from the terminal's detected background.
package styles

import "github.com/charmbracelet/lipgloss"

// Theme is a named palette. Use lipgloss.TerminalColor so a theme could hold an
// AdaptiveColor in future; today the presets use plain lipgloss.Color.
type Theme struct {
	Name    string
	BG      lipgloss.TerminalColor
	FG      lipgloss.TerminalColor
	Muted   lipgloss.TerminalColor
	Border  lipgloss.TerminalColor
	Accent  lipgloss.TerminalColor
	Warn    lipgloss.TerminalColor
	Danger  lipgloss.TerminalColor
	GroupFG lipgloss.TerminalColor
}

// Built-in presets. dark keeps the original Nord-ish palette but lifts the
// muted tone (#5e6779 → #8b94a7) so the help footer and sub-text read clearly
// on the near-black background. light is tuned for a pale terminal, keeping the
// "BG-on-Accent" tab/chip pattern legible.
var (
	darkTheme = Theme{
		Name:    "dark",
		BG:      lipgloss.Color("#11131a"),
		FG:      lipgloss.Color("#e5e9f0"),
		Muted:   lipgloss.Color("#8b94a7"),
		Border:  lipgloss.Color("#3b4252"),
		Accent:  lipgloss.Color("#88c0d0"),
		Warn:    lipgloss.Color("#ebcb8b"),
		Danger:  lipgloss.Color("#bf616a"),
		GroupFG: lipgloss.Color("#d08770"),
	}

	lightTheme = Theme{
		Name:    "light",
		BG:      lipgloss.Color("#f4f5f7"),
		FG:      lipgloss.Color("#1f2430"),
		Muted:   lipgloss.Color("#5a6373"),
		Border:  lipgloss.Color("#c2c8d2"),
		Accent:  lipgloss.Color("#1f6f8b"),
		Warn:    lipgloss.Color("#9a6a00"),
		Danger:  lipgloss.Color("#b3261e"),
		GroupFG: lipgloss.Color("#b5532a"),
	}
)

// registry holds every selectable preset by name. Adding a future theme is a
// single RegisterTheme call (or one literal here) — nothing else needs to know.
var registry = map[string]Theme{
	darkTheme.Name:  darkTheme,
	lightTheme.Name: lightTheme,
}

// RegisterTheme adds (or replaces) a named preset in the registry.
func RegisterTheme(t Theme) { registry[t.Name] = t }

// IsValidTheme reports whether name is a selectable theme. "auto" is always
// valid; otherwise the name must exist in the registry.
func IsValidTheme(name string) bool {
	if name == "auto" {
		return true
	}
	_, ok := registry[name]
	return ok
}

// ThemeNames returns the registry's preset names plus "auto", for help text and
// validation error messages.
func ThemeNames() []string {
	names := make([]string, 0, len(registry)+1)
	names = append(names, "auto")
	for n := range registry {
		names = append(names, n)
	}
	return names
}

// Resolve maps a config theme name to a concrete Theme. "auto" picks dark or
// light from the terminal background; unknown names fall back to dark so the
// app always has a sane palette.
func Resolve(name string) Theme {
	switch name {
	case "auto", "":
		if lipgloss.HasDarkBackground() {
			return darkTheme
		}
		return lightTheme
	default:
		if t, ok := registry[name]; ok {
			return t
		}
		return darkTheme
	}
}

// Active palette — populated by Apply. Render sites reference these at call
// time, so re-applying a theme re-themes the whole UI.
var (
	ColorBG      lipgloss.TerminalColor
	ColorFG      lipgloss.TerminalColor
	ColorMuted   lipgloss.TerminalColor
	ColorBorder  lipgloss.TerminalColor
	ColorAccent  lipgloss.TerminalColor
	ColorWarn    lipgloss.TerminalColor
	ColorDanger  lipgloss.TerminalColor
	ColorGroupFG lipgloss.TerminalColor
)

// chipInk is the dark text colour used on pod chips. Pod-chip backgrounds come
// from the fixed, always-bright WorkloadPalette, so dark text reads well under
// every theme — hence a constant rather than a theme token.
const chipInk = lipgloss.Color("#11131a")

// Borders never change with the theme, so they live as plain values.
var (
	// Group "region" — the outer border in the sketch. A warmer hue reads as a
	// container, not a card.
	GroupBorder = lipgloss.Border{
		Top: "═", Bottom: "═", Left: "║", Right: "║",
		TopLeft: "╔", TopRight: "╗", BottomLeft: "╚", BottomRight: "╝",
	}
	// Node card — the inner cards holding pods.
	NodeBorder = lipgloss.RoundedBorder()
)

// Composed styles — rebuilt by Apply from the active palette.
var (
	Group       lipgloss.Style
	GroupHeader lipgloss.Style
	Node        lipgloss.Style
	NodeFocused lipgloss.Style
	NodeHeader  lipgloss.Style
	NodeSub     lipgloss.Style
	PodChip     lipgloss.Style
	DSBand      lipgloss.Style
	Help        lipgloss.Style
	Title       lipgloss.Style
)

// Apply makes t the active theme: it sets the Color* vars and rebuilds every
// composed style from them. Rebuilding is mandatory — the styles bake colours in
// at construction, so reassigning only the Color* vars would leave them stale.
func Apply(t Theme) {
	ColorBG, ColorFG, ColorMuted = t.BG, t.FG, t.Muted
	ColorBorder, ColorAccent = t.Border, t.Accent
	ColorWarn, ColorDanger, ColorGroupFG = t.Warn, t.Danger, t.GroupFG

	Group = lipgloss.NewStyle().
		Border(GroupBorder).
		BorderForeground(ColorGroupFG).
		Padding(0, 1)

	GroupHeader = lipgloss.NewStyle().
		Foreground(ColorGroupFG).
		Bold(true)

	Node = lipgloss.NewStyle().
		Border(NodeBorder).
		BorderForeground(ColorBorder).
		Padding(0, 1)
	NodeFocused = Node.BorderForeground(ColorAccent)

	NodeHeader = lipgloss.NewStyle().
		Foreground(ColorFG).
		Bold(true)
	NodeSub = lipgloss.NewStyle().
		Foreground(ColorMuted)

	// Pod chip — coloured block representing a pod. Its background is always a
	// bright hue from the fixed WorkloadPalette, so the text uses a constant dark
	// ink (independent of the theme) to stay legible on every chip.
	PodChip = lipgloss.NewStyle().
		Foreground(chipInk).
		Padding(0, 1)

	// DaemonSet band divider.
	DSBand = lipgloss.NewStyle().Foreground(ColorMuted)

	// Help / status bar.
	Help = lipgloss.NewStyle().Foreground(ColorMuted)

	// Title bar at the top of the app.
	Title = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		Padding(0, 1)
}

// init applies the dark preset so the package vars are valid before main calls
// Apply (and for tests that render without configuring a theme).
func init() { Apply(darkTheme) }
