// Package detail renders the side-panel that slides over the floor plan when a
// node card is opened with Enter.
package detail

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	corev1 "k8s.io/api/core/v1"

	"github.com/shearn89/podscape/internal/analysis"
	"github.com/shearn89/podscape/internal/model"
	"github.com/shearn89/podscape/internal/tui/styles"
)

// Render produces the detail panel for a given node. `width` is the inner
// width available; the panel handles its own border + padding.
func Render(node model.Node, pods []model.Pod, overhead analysis.NodeOverhead, width int) string {
	if width < 30 {
		width = 30
	}
	header := styles.NodeHeader.Render(node.Name)
	sub := styles.NodeSub.Render(fmt.Sprintf("%s · %s · %s",
		node.InstanceType, node.Group.DisplayName, readyLabel(node.Ready)))

	taints := section("taints", taintLines(node.Taints))
	alloc := section("allocatable", []string{
		fmt.Sprintf("cpu %s", quantityString(node.Allocatable, corev1.ResourceCPU)),
		fmt.Sprintf("mem %s", quantityString(node.Allocatable, corev1.ResourceMemory)),
	})
	ds := section("daemonset overhead", []string{
		fmt.Sprintf("pods %d", overhead.DaemonSetPods),
		fmt.Sprintf("cpu %s (%.0f%% alloc)", overhead.CPURequest.String(), overhead.CPUPercent()),
		fmt.Sprintf("mem %s (%.0f%% alloc)", overhead.MemRequest.String(), overhead.MemPercent()),
	})
	pdsec := section("pods", podLines(pods))

	body := lipgloss.JoinVertical(lipgloss.Left, header, sub, "", taints, alloc, ds, pdsec)
	return styles.Node.Width(width).Render(body)
}

func readyLabel(ok bool) string {
	if ok {
		return "Ready"
	}
	return "NotReady"
}

func section(title string, items []string) string {
	t := styles.GroupHeader.Render(title)
	if len(items) == 0 {
		return t + "\n" + styles.NodeSub.Render("  —") + "\n"
	}
	var lines []string
	lines = append(lines, t)
	for _, it := range items {
		lines = append(lines, "  "+it)
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func taintLines(taints []model.Taint) []string {
	out := make([]string, 0, len(taints))
	for _, t := range taints {
		out = append(out, t.String())
	}
	return out
}

func podLines(pods []model.Pod) []string {
	sorted := make([]model.Pod, len(pods))
	copy(sorted, pods)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Owner.Kind != sorted[j].Owner.Kind {
			return sorted[i].Owner.Kind < sorted[j].Owner.Kind
		}
		return sorted[i].Name < sorted[j].Name
	})
	out := make([]string, 0, len(sorted))
	for _, p := range sorted {
		priority := p.Profile.PriorityClass
		if priority == "" {
			priority = "—"
		}
		out = append(out, fmt.Sprintf("%-12s %s/%s  pc=%s",
			abbrev(string(p.Owner.Kind)), p.Namespace, p.Name, priority))
	}
	return out
}

func abbrev(kind string) string {
	switch kind {
	case "Deployment":
		return "Deploy"
	case "StatefulSet":
		return "STS"
	case "DaemonSet":
		return "DS"
	case "Job":
		return "Job"
	case "ReplicaSet":
		return "RS"
	default:
		return kind
	}
}

func quantityString(rl corev1.ResourceList, name corev1.ResourceName) string {
	if q, ok := rl[name]; ok {
		return q.String()
	}
	return "—"
}
