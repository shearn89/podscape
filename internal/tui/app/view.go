package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shearn89/podscape/internal/model"
	"github.com/shearn89/podscape/internal/tui/detail"
	"github.com/shearn89/podscape/internal/tui/findings"
	"github.com/shearn89/podscape/internal/tui/floorplan"
	"github.com/shearn89/podscape/internal/tui/nodestable"
	"github.com/shearn89/podscape/internal/tui/styles"
)

func (m Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderTitle(),
		m.renderBody(),
		m.renderStatus(),
		m.help.View(m.keys),
	)
}

func (m Model) renderTitle() string {
	tabs := []string{"floor plan", "nodes table", "findings"}
	idx := int(m.tab)
	var parts []string
	for i, name := range tabs {
		label := name
		if i == int(tabFindings) {
			label = fmt.Sprintf("%s (%s)", name, findings.Summary(m.findings))
		}
		if i == idx {
			parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorBG).Background(styles.ColorAccent).Padding(0, 1).Render(label))
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorMuted).Padding(0, 1).Render(label))
		}
	}
	ctx := styles.NodeSub.Render(fmt.Sprintf("ctx=%s · density=%s · sort=%s",
		m.ctxName, m.density.String(), nodestable.SortLabel(m.sortKey)))
	title := styles.Title.Render("podscape")
	return lipgloss.JoinHorizontal(lipgloss.Top, title, strings.Join(parts, " "), "  ", ctx)
}

func (m Model) renderBody() string {
	switch m.tab {
	case tabNodesTable:
		return m.tableM.View()
	case tabFindings:
		width := m.width
		if width < 60 {
			width = 60
		}
		return findings.Render(m.findings, width)
	default:
		return m.renderFloorPlanWithDetail()
	}
}

func (m Model) renderFloorPlanWithDetail() string {
	// Before the first window-size message the viewport isn't sized yet; fall
	// back to a plain render so we still show something useful.
	if !m.planReady {
		return floorplan.Render(m.currentView())
	}
	plan := m.plan.View()
	if !m.showDetail || m.snap == nil {
		return plan
	}
	detailWidth := 46
	if m.width > 0 && detailWidth > m.width/2 {
		detailWidth = m.width / 2
	}
	side := clipHeight(m.renderDetailPane(detailWidth), m.plan.Height)
	return lipgloss.JoinHorizontal(lipgloss.Top, plan, " ", side)
}

func (m Model) renderDetailPane(width int) string {
	targets := floorplan.FocusTargets(m.currentView())
	if m.focusIdx < 0 || m.focusIdx >= len(targets) {
		return ""
	}
	target := targets[m.focusIdx].Node
	if target == "" {
		return ""
	}
	for _, n := range m.snap.Nodes {
		if n.Name == target {
			return detail.Render(n, model.PodsOnNode(m.snap.Pods, target), m.overhead[target], width-2)
		}
	}
	return ""
}

// clipHeight truncates s to at most h lines so a tall detail pane can't push the
// overall layout past the viewport height.
func clipHeight(s string, h int) string {
	if h <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= h {
		return s
	}
	return strings.Join(lines[:h], "\n")
}

func (m Model) renderStatus() string {
	if m.loadErr != nil {
		return styles.NodeSub.Render("⚠ " + m.statusMsg)
	}
	return styles.NodeSub.Render(m.statusMsg)
}
