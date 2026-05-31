// Package floorplan renders the main "node-group floor plan" view: every node
// group as an outer bordered region containing a grid of node cards; each card
// has a workload band on top, a DaemonSet band on the bottom.
package floorplan

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shearn89/podscape/internal/analysis"
	"github.com/shearn89/podscape/internal/k8s"
	"github.com/shearn89/podscape/internal/tui/styles"
)

// View captures everything the renderer needs. It is data-only — no
// bubbletea types — so the renderer is easy to test.
type View struct {
	Snapshot     *k8s.Snapshot
	Overhead     map[string]analysis.NodeOverhead
	Density      Density
	Width        int
	FocusedNode  string          // optional; empty for "no focus"
	FlaggedNodes map[string]bool // nodes with at least one finding — header gets a ⚠
}

// Render produces the full floor-plan string for the given view.
func Render(v View) string {
	if v.Snapshot == nil || len(v.Snapshot.Groups) == 0 {
		return styles.Help.Render("(no nodes — is the cluster empty or unreachable?)")
	}
	parts := make([]string, 0, len(v.Snapshot.Groups))
	for _, g := range v.Snapshot.Groups {
		parts = append(parts, renderGroup(g, v.Snapshot.Pods, v.Overhead, v.Density, v.Width, v.FocusedNode, v.FlaggedNodes))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...) + "\n"
}

// FocusableNodes returns the ordered list of node names a user can move focus
// between. Used by the app to drive arrow/hjkl navigation deterministically.
func FocusableNodes(v View) []string {
	if v.Snapshot == nil {
		return nil
	}
	var out []string
	for _, g := range v.Snapshot.Groups {
		for _, n := range g.Nodes {
			out = append(out, n.Name)
		}
	}
	return out
}

// stripANSI is exported for tests so golden snapshots compare raw structure
// rather than colour codes.
func stripANSI(s string) string {
	// Lipgloss emits CSI sequences like ESC[…m. Strip them for tests.
	var b strings.Builder
	b.Grow(len(s))
	in := false
	for _, r := range s {
		if r == 0x1b {
			in = true
			continue
		}
		if in {
			if r == 'm' {
				in = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
