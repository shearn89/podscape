package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/shearn89/podscape/internal/analysis"
	"github.com/shearn89/podscape/internal/tui/floorplan"
	"github.com/shearn89/podscape/internal/tui/nodestable"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.rebuildTable()
		return m, nil

	case loadedMsg:
		m.snap = msg.snap
		m.loadErr = msg.err
		if msg.err != nil {
			m.statusMsg = "load failed: " + msg.err.Error()
		} else if msg.snap != nil {
			m.overhead = analysis.DaemonSetOverhead(msg.snap.Nodes, msg.snap.Pods)
			m.findings = analysis.RunChecks(msg.snap.Nodes, msg.snap.Pods)
			m.flaggedNodes = findingsToFlagged(m.findings)
			m.statusMsg = fmt.Sprintf("loaded %d nodes / %d pods at %s",
				len(msg.snap.Nodes), len(msg.snap.Pods), time.Now().Format("15:04:05"))
			m.rebuildTable()
			m.clampFocus()
		}
		return m, nil

	case tickMsg:
		return m, tea.Batch(m.fetchCmd(), m.tickCmd())

	case tea.KeyMsg:
		return m.onKey(msg)
	}

	// While the table tab is active, route unhandled keys to the table.
	if m.tab == tabNodesTable {
		var cmd tea.Cmd
		m.tableM, cmd = m.tableM.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) onKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := m.keys
	switch {
	case key.Matches(msg, k.Quit):
		return m, tea.Quit
	case key.Matches(msg, k.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case key.Matches(msg, k.Refresh):
		m.statusMsg = "refreshing…"
		return m, m.fetchCmd()
	case key.Matches(msg, k.TabPlan):
		m.tab = tabFloorPlan
		return m, nil
	case key.Matches(msg, k.TabTable):
		m.tab = tabNodesTable
		return m, nil
	case key.Matches(msg, k.TabFindings):
		m.tab = tabFindings
		return m, nil
	case key.Matches(msg, k.NextTab):
		m.tab = (m.tab + 1) % 3
		return m, nil
	case key.Matches(msg, k.Compact):
		m.density = floorplan.DensityCompact
		return m, nil
	case key.Matches(msg, k.Normal):
		m.density = floorplan.DensityNormal
		return m, nil
	case key.Matches(msg, k.Wide):
		m.density = floorplan.DensityWide
		return m, nil
	}

	if m.tab == tabFloorPlan {
		return m.onFloorPlanKey(msg)
	}
	return m.onTableKey(msg)
}

func (m Model) onFloorPlanKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := m.keys
	focusable := floorplan.FocusableNodes(m.currentView())
	switch {
	case key.Matches(msg, k.Enter):
		if len(focusable) > 0 {
			m.showDetail = true
		}
	case key.Matches(msg, k.Esc):
		m.showDetail = false
	case key.Matches(msg, k.Up), key.Matches(msg, k.Left):
		if m.focusIdx > 0 {
			m.focusIdx--
		}
	case key.Matches(msg, k.Down), key.Matches(msg, k.Right):
		if m.focusIdx < len(focusable)-1 {
			m.focusIdx++
		}
	}
	return m, nil
}

func (m Model) onTableKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := m.keys
	if key.Matches(msg, k.SortNext) {
		m.sortKey = m.sortKey.Next()
		m.rebuildTable()
		return m, nil
	}
	var cmd tea.Cmd
	m.tableM, cmd = m.tableM.Update(msg)
	return m, cmd
}

func (m *Model) rebuildTable() {
	if m.snap == nil {
		return
	}
	m.tableM = nodestable.New(m.snap, m.overhead, m.sortKey, m.width)
}

func (m *Model) clampFocus() {
	max := len(floorplan.FocusableNodes(m.currentView())) - 1
	if m.focusIdx > max {
		m.focusIdx = max
	}
	if m.focusIdx < 0 {
		m.focusIdx = 0
	}
}

func (m Model) currentView() floorplan.View {
	focusName := ""
	if m.snap != nil {
		names := floorplan.FocusableNodes(floorplan.View{Snapshot: m.snap})
		if m.focusIdx >= 0 && m.focusIdx < len(names) {
			focusName = names[m.focusIdx]
		}
	}
	w := m.width
	if w < 60 {
		w = 60
	}
	return floorplan.View{
		Snapshot:     m.snap,
		Overhead:     m.overhead,
		Density:      m.density,
		Width:        w,
		FocusedNode:  focusName,
		FlaggedNodes: m.flaggedNodes,
	}
}

func findingsToFlagged(items []analysis.Finding) map[string]bool {
	out := map[string]bool{}
	for _, f := range items {
		if f.Node != "" {
			out[f.Node] = true
		}
	}
	return out
}
