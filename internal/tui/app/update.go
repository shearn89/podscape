package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shearn89/podscape/internal/analysis"
	"github.com/shearn89/podscape/internal/tui/floorplan"
	"github.com/shearn89/podscape/internal/tui/nodestable"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.rebuildTable()
		m.syncPlan()
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
			m.seedGroups()
			m.clampFocus()
			m.syncPlan()
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
		m.syncPlan()
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
		m.syncPlan()
		return m, nil
	case key.Matches(msg, k.Normal):
		m.density = floorplan.DensityNormal
		m.syncPlan()
		return m, nil
	case key.Matches(msg, k.Wide):
		m.density = floorplan.DensityWide
		m.syncPlan()
		return m, nil
	case key.Matches(msg, k.Accordion):
		m.accordion = !m.accordion
		if m.accordion {
			m.enforceAccordion()
		} else {
			// Leaving accordion mode reveals everything again.
			m.collapsed = map[string]bool{}
		}
		m.syncPlan()
		return m, nil
	}

	if m.tab == tabFloorPlan {
		return m.onFloorPlanKey(msg)
	}
	return m.onTableKey(msg)
}

func (m Model) onFloorPlanKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := m.keys
	targets := floorplan.FocusTargets(m.currentView())

	// Manual scroll keys move the viewport directly and must not be overridden
	// by the scroll-to-focus logic, so they return early.
	switch {
	case key.Matches(msg, k.ScrollUp):
		m.plan.HalfPageUp()
		return m, nil
	case key.Matches(msg, k.ScrollDown):
		m.plan.HalfPageDown()
		return m, nil
	case key.Matches(msg, k.ScrollTop):
		m.plan.GotoTop()
		return m, nil
	case key.Matches(msg, k.ScrollBot):
		m.plan.GotoBottom()
		return m, nil
	}

	switch {
	case key.Matches(msg, k.Collapse):
		if m.focusIdx >= 0 && m.focusIdx < len(targets) {
			m.toggleCollapse(targets[m.focusIdx].GroupKey)
		}
	case key.Matches(msg, k.Enter):
		if m.focusIdx >= 0 && m.focusIdx < len(targets) {
			t := targets[m.focusIdx]
			if t.Node != "" {
				m.showDetail = true
			} else {
				// A collapsed group header is focused — Enter expands it.
				m.toggleCollapse(t.GroupKey)
			}
		}
	case key.Matches(msg, k.Esc):
		m.showDetail = false
	case key.Matches(msg, k.Up), key.Matches(msg, k.Left):
		if m.focusIdx > 0 {
			m.focusIdx--
			m.enforceAccordion()
		}
	case key.Matches(msg, k.Down), key.Matches(msg, k.Right):
		if m.focusIdx < len(targets)-1 {
			m.focusIdx++
			m.enforceAccordion()
		}
	}
	m.syncPlan()
	return m, nil
}

// seedGroups applies the configured collapse/accordion defaults once, on the
// first snapshot. It's guarded so periodic refreshes don't undo the user's
// later expand/collapse actions.
func (m *Model) seedGroups() {
	if m.groupsInit || m.snap == nil {
		return
	}
	m.groupsInit = true
	if m.collapseByDefault {
		for _, g := range m.snap.Groups {
			m.collapsed[g.Group.Key] = true
		}
	}
	m.enforceAccordion()
}

// enforceAccordion, when accordion mode is on, keeps exactly one group expanded:
// the one owning the focused target. Everything else collapses. When this
// actually changes which group is open, focus is re-pointed at that group so
// the user lands on its first node rather than an out-of-range index. Navigating
// within the already-open group is a no-op, so focus isn't yanked around.
func (m *Model) enforceAccordion() {
	if !m.accordion || m.snap == nil {
		return
	}
	targets := floorplan.FocusTargets(m.currentView())
	focusKey := ""
	if m.focusIdx >= 0 && m.focusIdx < len(targets) {
		focusKey = targets[m.focusIdx].GroupKey
	}
	if focusKey == "" {
		return
	}
	changed := false
	for _, g := range m.snap.Groups {
		key := g.Group.Key
		if key == focusKey {
			if m.collapsed[key] {
				delete(m.collapsed, key)
				changed = true
			}
		} else if !m.collapsed[key] {
			m.collapsed[key] = true
			changed = true
		}
	}
	if changed {
		m.refocusGroup(focusKey)
	}
}

// toggleCollapse flips the collapsed state of a group and keeps focus on it so
// the user doesn't lose their place as the focus ring changes shape.
func (m *Model) toggleCollapse(groupKey string) {
	if groupKey == "" {
		return
	}
	if m.collapsed[groupKey] {
		delete(m.collapsed, groupKey)
	} else {
		m.collapsed[groupKey] = true
		m.showDetail = false
	}
	m.refocusGroup(groupKey)
}

// refocusGroup points focus at the first focus target belonging to groupKey.
func (m *Model) refocusGroup(groupKey string) {
	for i, t := range floorplan.FocusTargets(m.currentView()) {
		if t.GroupKey == groupKey {
			m.focusIdx = i
			return
		}
	}
	m.clampFocus()
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
	max := len(floorplan.FocusTargets(m.currentView())) - 1
	if m.focusIdx > max {
		m.focusIdx = max
	}
	if m.focusIdx < 0 {
		m.focusIdx = 0
	}
}

func (m Model) currentView() floorplan.View {
	return m.planViewWidth(m.planWidth())
}

// planWidth is the width available to the floor plan, shrunk when the detail
// pane is open so the two sit side by side.
func (m Model) planWidth() int {
	w := m.width
	if m.showDetail && m.snap != nil {
		detailWidth := 46
		if m.width > 0 && detailWidth > m.width/2 {
			detailWidth = m.width / 2
		}
		w = m.width - detailWidth - 2
	}
	if w < 60 {
		w = 60
	}
	return w
}

// planViewWidth builds the floorplan.View, resolving the focused target into a
// node name or a collapsed-group key for highlighting.
func (m Model) planViewWidth(w int) floorplan.View {
	targets := floorplan.FocusTargets(floorplan.View{Snapshot: m.snap, Collapsed: m.collapsed})
	focusName, focusGroup := "", ""
	if m.focusIdx >= 0 && m.focusIdx < len(targets) {
		t := targets[m.focusIdx]
		focusName = t.Node
		if t.Node == "" {
			focusGroup = t.GroupKey
		}
	}
	return floorplan.View{
		Snapshot:     m.snap,
		Overhead:     m.overhead,
		Density:      m.density,
		Width:        w,
		FocusedNode:  focusName,
		FocusedGroup: focusGroup,
		FlaggedNodes: m.flaggedNodes,
		Collapsed:    m.collapsed,
	}
}

// syncPlan re-renders the floor plan into the viewport, resizes it to fit the
// space left by the title/status/help chrome, and scrolls so the focused target
// stays visible.
func (m *Model) syncPlan() {
	if m.width == 0 || m.height == 0 {
		return
	}
	w := m.planWidth()
	content, starts := floorplan.RenderPlan(m.planViewWidth(w))

	m.plan.Width = w
	h := m.height - m.chromeHeight()
	if h < 1 {
		h = 1
	}
	m.plan.Height = h
	m.plan.SetContent(content)
	m.planReady = true

	if m.focusIdx >= 0 && m.focusIdx < len(starts) {
		m.scrollToLine(starts[m.focusIdx])
	}
}

// chromeHeight is the number of rows consumed by everything except the floor
// plan: the title bar, status line, and help footer.
func (m Model) chromeHeight() int {
	return lipgloss.Height(m.renderTitle()) +
		lipgloss.Height(m.renderStatus()) +
		lipgloss.Height(m.help.View(m.keys))
}

// scrollToLine nudges the viewport so the given content line is on screen,
// keeping a little context when it lands below the fold.
func (m *Model) scrollToLine(line int) {
	top := m.plan.YOffset
	bottom := top + m.plan.Height
	switch {
	case line < top:
		m.plan.SetYOffset(line)
	case line >= bottom-1:
		m.plan.SetYOffset(line - m.plan.Height/2)
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
