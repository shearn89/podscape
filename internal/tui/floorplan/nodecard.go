package floorplan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shearn89/podscape/internal/analysis"
	"github.com/shearn89/podscape/internal/model"
	"github.com/shearn89/podscape/internal/tui/styles"
)

// renderNodeCard returns the rendered string for one node card at the given
// density, optionally focused. Pods running on the node are split into a
// workload band (top) and a DaemonSet band (bottom) per the sketch.
func renderNodeCard(node model.Node, pods []model.Pod, overhead analysis.NodeOverhead, d Density, focused, flagged bool) string {
	cardW := d.CardWidth()
	inner := cardW - 4 // borders + padding

	header := renderHeader(node, inner, flagged)
	workloadBand := renderWorkloadBand(pods, d, inner)
	dsBand := renderDSBand(pods, overhead, d, inner)

	body := lipgloss.JoinVertical(lipgloss.Left, header, workloadBand, dsBand)

	style := styles.Node.Width(inner + 2)
	if focused {
		style = styles.NodeFocused.Width(inner + 2)
	}
	return style.Render(body)
}

func renderHeader(node model.Node, width int, flagged bool) string {
	name := truncate(node.Name, width)
	sub := node.InstanceType
	if sub == "" {
		sub = "—"
	}
	if !node.Ready {
		sub = "NotReady"
	}
	headerText := name
	if flagged {
		marker := lipgloss.NewStyle().Foreground(styles.ColorWarn).Bold(true).Render("⚠ ")
		headerText = marker + truncate(node.Name, width-2)
	}
	header := styles.NodeHeader.Render(headerText)
	subline := styles.NodeSub.Render(truncate(sub, width))
	return lipgloss.JoinVertical(lipgloss.Left, header, subline)
}

func renderWorkloadBand(pods []model.Pod, d Density, width int) string {
	chips := chipsFor(pods, d, false)
	if len(chips) == 0 {
		return styles.NodeSub.Render("(no app pods)")
	}
	return wrapChips(chips, width, d.PodChipWidth())
}

func renderDSBand(pods []model.Pod, overhead analysis.NodeOverhead, d Density, width int) string {
	divider := strings.Repeat("─", width)
	chips := chipsFor(pods, d, true)
	var pct string
	if overhead.DaemonSetPods > 0 {
		pct = styles.NodeSub.Render(fmt.Sprintf("DS cpu %.0f%%  mem %.0f%%",
			overhead.CPUPercent(), overhead.MemPercent()))
	} else {
		pct = styles.NodeSub.Render("DS —")
	}
	chipRow := wrapChips(chips, width, d.PodChipWidth())
	if chipRow == "" {
		chipRow = styles.NodeSub.Render("(no DS pods)")
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		styles.DSBand.Render(divider),
		chipRow,
		pct,
	)
}

// chipsFor returns one chip per pod. When daemonset is true, only DaemonSet
// pods are included; otherwise only non-DaemonSet pods.
func chipsFor(pods []model.Pod, d Density, daemonset bool) []string {
	filtered := make([]model.Pod, 0, len(pods))
	for _, p := range pods {
		isDS := p.Owner.Kind == model.KindDaemonSet
		if daemonset != isDS {
			continue
		}
		filtered = append(filtered, p)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Owner.Name != filtered[j].Owner.Name {
			return filtered[i].Owner.Name < filtered[j].Owner.Name
		}
		return filtered[i].Name < filtered[j].Name
	})
	chipW := d.PodChipWidth()
	out := make([]string, 0, len(filtered))
	for _, p := range filtered {
		colour := model.ColorFor(p.Owner)
		label := chipLabel(p, d)
		style := styles.PodChip.
			Background(colour).
			Width(chipW)
		out = append(out, style.Render(label))
	}
	return out
}

func chipLabel(p model.Pod, d Density) string {
	switch d {
	case DensityCompact:
		return initials(p.Owner.Name)
	case DensityWide:
		return truncate(p.Name, d.PodChipWidth()-2)
	default:
		return truncate(p.Owner.Name, d.PodChipWidth()-2)
	}
}

func initials(s string) string {
	if s == "" {
		return "?"
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
	switch len(parts) {
	case 0:
		return string([]rune(s)[:1])
	case 1:
		r := []rune(parts[0])
		if len(r) >= 2 {
			return string(r[:2])
		}
		return string(r)
	default:
		return string([]rune(parts[0])[:1]) + string([]rune(parts[1])[:1])
	}
}

func wrapChips(chips []string, width, chipW int) string {
	if len(chips) == 0 {
		return ""
	}
	perRow := width / (chipW + 1)
	if perRow < 1 {
		perRow = 1
	}
	var rows []string
	for i := 0; i < len(chips); i += perRow {
		end := i + perRow
		if end > len(chips) {
			end = len(chips)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, interspersed(chips[i:end], " ")...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// interspersed returns items separated by sep — like strings.Join but for
// pre-styled lipgloss boxes that need their own slice entry to render.
func interspersed(items []string, sep string) []string {
	if len(items) <= 1 {
		return items
	}
	out := make([]string, 0, len(items)*2-1)
	for i, it := range items {
		if i > 0 {
			out = append(out, sep)
		}
		out = append(out, it)
	}
	return out
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}
