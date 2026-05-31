// Package findings renders the Findings tab — a scrollable, severity-coloured
// list of correctness signals from the analysis package.
package findings

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shearn89/podscape/internal/analysis"
	"github.com/shearn89/podscape/internal/tui/styles"
)

// Render returns the rendered findings list (no viewport chrome).
func Render(items []analysis.Finding, width int) string {
	if len(items) == 0 {
		return styles.Help.Render("✓ no findings — your cluster looks healthy on the checks we run")
	}
	var b strings.Builder
	for _, f := range items {
		b.WriteString(renderLine(f))
		b.WriteString("\n")
	}
	return b.String()
}

func renderLine(f analysis.Finding) string {
	b := badge(f.Severity)
	code := lipgloss.NewStyle().Foreground(styles.ColorAccent).Render("[" + f.Code + "]")
	header := fmt.Sprintf("%s %s", b, code)
	body := lipgloss.NewStyle().Foreground(styles.ColorFG).Render(f.Message)
	loc := locationString(f)
	if loc != "" {
		loc = styles.NodeSub.Render("    " + loc)
	}
	out := header + "\n  " + body
	if loc != "" {
		out += "\n" + loc
	}
	return out
}

func badge(s analysis.Severity) string {
	switch s {
	case analysis.SeverityError:
		return lipgloss.NewStyle().Foreground(styles.ColorBG).Background(styles.ColorDanger).Bold(true).Padding(0, 1).Render("ERR")
	case analysis.SeverityWarn:
		return lipgloss.NewStyle().Foreground(styles.ColorBG).Background(styles.ColorWarn).Bold(true).Padding(0, 1).Render("WARN")
	default:
		return lipgloss.NewStyle().Foreground(styles.ColorBG).Background(styles.ColorAccent).Bold(true).Padding(0, 1).Render("INFO")
	}
}

func locationString(f analysis.Finding) string {
	switch {
	case f.Pod != "" && f.Node != "":
		return fmt.Sprintf("at %s on node %s", f.Pod, f.Node)
	case f.Pod != "":
		return "at " + f.Pod
	case f.Node != "":
		return "on node " + f.Node
	}
	return ""
}

// NodesWithFindings returns the set of node names that have at least one
// finding attached — used by the floor-plan to overlay a ⚠ marker.
func NodesWithFindings(items []analysis.Finding) map[string]bool {
	out := map[string]bool{}
	for _, f := range items {
		if f.Node != "" {
			out[f.Node] = true
		}
	}
	return out
}

// Summary returns a one-line tally like "2 err · 3 warn".
func Summary(items []analysis.Finding) string {
	var e, w, i int
	for _, f := range items {
		switch f.Severity {
		case analysis.SeverityError:
			e++
		case analysis.SeverityWarn:
			w++
		default:
			i++
		}
	}
	if e+w+i == 0 {
		return styles.Help.Render("0 findings")
	}
	var parts []string
	if e > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorDanger).Render(fmt.Sprintf("%d err", e)))
	}
	if w > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorWarn).Render(fmt.Sprintf("%d warn", w)))
	}
	if i > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(fmt.Sprintf("%d info", i)))
	}
	return strings.Join(parts, " · ")
}
