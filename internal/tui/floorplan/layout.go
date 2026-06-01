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
	Collapsed    map[string]bool // group key -> collapsed (body hidden)
	FocusedGroup string          // group key whose collapsed header is focused
}

// FocusTarget is one stop in the floor-plan's focus ring. A target is either a
// node (Node set) or the header of a collapsed group (Node empty), so the user
// can land on a collapsed group and re-expand it.
type FocusTarget struct {
	GroupKey string
	Node     string // "" => the group's (collapsed) header itself
}

// Render produces the full floor-plan string for the given view.
func Render(v View) string {
	content, _ := RenderPlan(v)
	return content + "\n"
}

// RenderPlan produces the floor-plan string together with the starting line of
// every focus target (in FocusTargets order). The line offsets let the app
// scroll a viewport so the focused target stays in view.
func RenderPlan(v View) (string, []int) {
	if v.Snapshot == nil || len(v.Snapshot.Groups) == 0 {
		return styles.Help.Render("(no nodes — is the cluster empty or unreachable?)"), nil
	}
	parts := make([]string, 0, len(v.Snapshot.Groups))
	var starts []int
	cum := 0
	for _, g := range v.Snapshot.Groups {
		collapsed := v.Collapsed[g.Group.Key]
		groupFocused := collapsed && v.FocusedGroup == g.Group.Key
		block, offsets := renderGroup(g, v.Snapshot.Pods, v.Overhead, v.Density, v.Width, v.FocusedNode, v.FlaggedNodes, collapsed, groupFocused)
		for _, off := range offsets {
			starts = append(starts, cum+off)
		}
		parts = append(parts, block)
		cum += lipgloss.Height(block)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...), starts
}

// FocusTargets returns the ordered focus ring: nodes of expanded groups, plus a
// single header stop for each collapsed group. Order matches RenderPlan's line
// offsets one-for-one.
func FocusTargets(v View) []FocusTarget {
	if v.Snapshot == nil {
		return nil
	}
	var out []FocusTarget
	for _, g := range v.Snapshot.Groups {
		if v.Collapsed[g.Group.Key] {
			out = append(out, FocusTarget{GroupKey: g.Group.Key})
			continue
		}
		for _, n := range g.Nodes {
			out = append(out, FocusTarget{GroupKey: g.Group.Key, Node: n.Name})
		}
	}
	return out
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
