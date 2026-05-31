// Package app is the root Bubble Tea model: it owns the snapshot, knows which
// tab is showing, dispatches key presses, and schedules refreshes.
package app

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/client-go/kubernetes"

	"github.com/shearn89/podscape/internal/analysis"
	"github.com/shearn89/podscape/internal/k8s"
	"github.com/shearn89/podscape/internal/tui/floorplan"
	"github.com/shearn89/podscape/internal/tui/nodestable"
)

type tab int

const (
	tabFloorPlan tab = iota
	tabNodesTable
	tabFindings
)

// Model is the root tea.Model.
type Model struct {
	cs        kubernetes.Interface
	ctxName   string
	namespace string
	refresh   time.Duration

	keys keyMap
	help help.Model

	tab        tab
	density    floorplan.Density
	width      int
	height     int
	focusIdx   int
	showDetail bool
	sortKey    nodestable.SortKey
	statusMsg  string

	snap         *k8s.Snapshot
	overhead     map[string]analysis.NodeOverhead
	loadErr      error
	tableM       table.Model
	findings     []analysis.Finding
	flaggedNodes map[string]bool
}

// New returns a fresh root model ready for tea.NewProgram.
func New(cs kubernetes.Interface, ctxName, namespace string, refresh time.Duration) Model {
	h := help.New()
	h.ShowAll = false
	return Model{
		cs:        cs,
		ctxName:   ctxName,
		namespace: namespace,
		refresh:   refresh,
		keys:      defaultKeys(),
		help:      h,
		density:   floorplan.DensityNormal,
		statusMsg: "loading…",
	}
}

// loadedMsg carries the result of a Fetch.
type loadedMsg struct {
	snap *k8s.Snapshot
	err  error
}

// tickMsg fires periodically to schedule the next refresh.
type tickMsg struct{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetchCmd(), m.tickCmd())
}

func (m Model) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		snap, err := k8s.Fetch(ctx, m.cs, m.namespace)
		return loadedMsg{snap: snap, err: err}
	}
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.refresh, func(time.Time) tea.Msg { return tickMsg{} })
}
