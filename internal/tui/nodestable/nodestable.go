// Package nodestable renders the secondary tab — a sortable Bubbles table of
// per-node DaemonSet overhead. Useful when the floor-plan view gets dense.
package nodestable

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/shearn89/podscape/internal/analysis"
	"github.com/shearn89/podscape/internal/k8s"
	"github.com/shearn89/podscape/internal/tui/styles"
)

// SortKey identifies the column to sort by.
type SortKey int

const (
	SortNode SortKey = iota
	SortGroup
	SortDSPods
	SortCPUPct
	SortMemPct
)

func (k SortKey) Next() SortKey {
	return (k + 1) % 5
}

// New returns a Bubbles table populated from the snapshot + overhead map.
func New(snap *k8s.Snapshot, overhead map[string]analysis.NodeOverhead, sortKey SortKey, width int) table.Model {
	cols := []table.Column{
		{Title: "NODE", Width: 28},
		{Title: "GROUP", Width: 18},
		{Title: "DS PODS", Width: 8},
		{Title: "DS CPU", Width: 10},
		{Title: "CPU %", Width: 7},
		{Title: "DS MEM", Width: 12},
		{Title: "MEM %", Width: 7},
	}
	rows := buildRows(snap, overhead, sortKey)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	style := table.DefaultStyles()
	style.Header = style.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorBorder).
		BorderBottom(true).
		Foreground(styles.ColorAccent).
		Bold(true)
	style.Selected = style.Selected.
		Foreground(styles.ColorBG).
		Background(styles.ColorAccent).
		Bold(true)
	t.SetStyles(style)
	return t
}

func buildRows(snap *k8s.Snapshot, overhead map[string]analysis.NodeOverhead, sortKey SortKey) []table.Row {
	if snap == nil {
		return nil
	}
	type row struct {
		node  string
		group string
		oh    analysis.NodeOverhead
	}
	items := make([]row, 0, len(snap.Nodes))
	for _, n := range snap.Nodes {
		items = append(items, row{node: n.Name, group: n.Group.DisplayName, oh: overhead[n.Name]})
	}
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		switch sortKey {
		case SortGroup:
			if a.group != b.group {
				return a.group < b.group
			}
			return a.node < b.node
		case SortDSPods:
			return a.oh.DaemonSetPods > b.oh.DaemonSetPods
		case SortCPUPct:
			return a.oh.CPUPercent() > b.oh.CPUPercent()
		case SortMemPct:
			return a.oh.MemPercent() > b.oh.MemPercent()
		default:
			return a.node < b.node
		}
	})
	rows := make([]table.Row, 0, len(items))
	for _, it := range items {
		rows = append(rows, table.Row{
			it.node,
			it.group,
			fmt.Sprintf("%d", it.oh.DaemonSetPods),
			it.oh.CPURequest.String(),
			fmt.Sprintf("%.0f", it.oh.CPUPercent()),
			it.oh.MemRequest.String(),
			fmt.Sprintf("%.0f", it.oh.MemPercent()),
		})
	}
	return rows
}

// SortLabel returns the human-friendly label for the current sort key.
func SortLabel(k SortKey) string {
	switch k {
	case SortNode:
		return "node"
	case SortGroup:
		return "group"
	case SortDSPods:
		return "ds pods"
	case SortCPUPct:
		return "cpu %"
	case SortMemPct:
		return "mem %"
	}
	return "?"
}
