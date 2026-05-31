package model

import (
	"hash/fnv"

	"github.com/charmbracelet/lipgloss"
)

// WorkloadPalette is a small fixed palette of distinct hues used to colour pod
// boxes. The DaemonSet hue is reserved and not picked from this list.
var WorkloadPalette = []lipgloss.Color{
	lipgloss.Color("#ef476f"), // red
	lipgloss.Color("#f78c6b"), // orange
	lipgloss.Color("#ffd166"), // amber
	lipgloss.Color("#83e377"), // lime
	lipgloss.Color("#06d6a0"), // teal
	lipgloss.Color("#118ab2"), // blue
	lipgloss.Color("#7678ed"), // indigo
	lipgloss.Color("#c490e4"), // lavender
	lipgloss.Color("#ec9deb"), // pink
	lipgloss.Color("#f7b801"), // gold
}

// DaemonSetColor is always used for DaemonSet pods so they read as "the bottom
// band" rather than competing for palette slots with regular workloads.
const DaemonSetColor = lipgloss.Color("#26a269")

// ColorFor returns a deterministic, stable colour for a workload. The same
// WorkloadKey always returns the same colour within a run (and across runs,
// since the hash is deterministic) — so a Deployment renders the same hue on
// every node it lands on.
//
// DaemonSets always map to DaemonSetColor and bypass the palette.
func ColorFor(k WorkloadKey) lipgloss.Color {
	if k.Kind == KindDaemonSet {
		return DaemonSetColor
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(k.String()))
	return WorkloadPalette[int(h.Sum32())%len(WorkloadPalette)]
}
