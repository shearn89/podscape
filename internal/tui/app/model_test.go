package app

import (
	"testing"

	"github.com/shearn89/podscape/internal/config"
	"github.com/shearn89/podscape/internal/k8s"
	"github.com/shearn89/podscape/internal/model"
)

// fakeSnapshot builds a minimal two-group snapshot for exercising the model's
// collapse/accordion bookkeeping without touching a cluster.
func fakeSnapshot() *k8s.Snapshot {
	mk := func(key string, nodes ...string) k8s.GroupedNodes {
		g := k8s.GroupedNodes{Group: model.NodeGroup{Key: key, DisplayName: key}}
		for _, n := range nodes {
			g.Nodes = append(g.Nodes, model.Node{Name: n, Group: g.Group})
		}
		return g
	}
	return &k8s.Snapshot{
		Groups: []k8s.GroupedNodes{
			mk("g1", "n1", "n2"),
			mk("g2", "n3", "n4"),
		},
	}
}

func load(m Model, snap *k8s.Snapshot) Model {
	next, _ := m.Update(loadedMsg{snap: snap})
	return next.(Model)
}

func TestNewDefaultView(t *testing.T) {
	m := New(nil, "ctx", "", 0, config.Config{DefaultView: config.ViewFindings, Theme: "dark"})
	if m.tab != tabFindings {
		t.Errorf("tab = %v, want tabFindings", m.tab)
	}
}

func TestCollapseByDefaultSeedsAllGroups(t *testing.T) {
	m := New(nil, "ctx", "", 0, config.Config{DefaultView: config.ViewFloorPlan, GroupsCollapsed: true, Theme: "dark"})
	m = load(m, fakeSnapshot())
	for _, g := range m.snap.Groups {
		if !m.collapsed[g.Group.Key] {
			t.Errorf("group %q not collapsed", g.Group.Key)
		}
	}
}

func TestSeedRunsOnceAcrossRefreshes(t *testing.T) {
	m := New(nil, "ctx", "", 0, config.Config{DefaultView: config.ViewFloorPlan, GroupsCollapsed: true, Theme: "dark"})
	m = load(m, fakeSnapshot())
	// User manually expands g1.
	delete(m.collapsed, "g1")
	// A refresh arrives — seeding must not re-collapse g1.
	m = load(m, fakeSnapshot())
	if m.collapsed["g1"] {
		t.Error("refresh re-collapsed a group the user expanded")
	}
}

func TestAccordionKeepsOneGroupExpanded(t *testing.T) {
	m := New(nil, "ctx", "", 0, config.Config{DefaultView: config.ViewFloorPlan, Accordion: true, Theme: "dark"})
	m = load(m, fakeSnapshot())
	expanded := 0
	for _, g := range m.snap.Groups {
		if !m.collapsed[g.Group.Key] {
			expanded++
		}
	}
	if expanded != 1 {
		t.Errorf("accordion left %d groups expanded, want 1", expanded)
	}
}
