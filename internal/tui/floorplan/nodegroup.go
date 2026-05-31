package floorplan

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shearn89/podscape/internal/analysis"
	"github.com/shearn89/podscape/internal/k8s"
	"github.com/shearn89/podscape/internal/model"
	"github.com/shearn89/podscape/internal/tui/styles"
)

// renderGroup renders one node-group: an outer bordered region whose body is a
// grid of node cards. `focusedNode` highlights one card if it belongs to the
// group; pass "" to focus nothing.
func renderGroup(g k8s.GroupedNodes, pods []model.Pod, overhead map[string]analysis.NodeOverhead, d Density, availWidth int, focusedNode string, flagged map[string]bool) string {
	cardW := d.CardWidth()
	gap := 1
	innerWidth := availWidth - 4 // outer border + padding
	if innerWidth < cardW {
		innerWidth = cardW
	}
	cols := (innerWidth + gap) / (cardW + gap)
	if cols < 1 {
		cols = 1
	}

	cards := make([]string, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		cards = append(cards, renderNodeCard(n, model.PodsOnNode(pods, n.Name), overhead[n.Name], d, n.Name == focusedNode, flagged[n.Name]))
	}

	body := gridRows(cards, cols)
	header := renderGroupHeader(g)

	bodyStyled := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return styles.Group.Width(innerWidth + 2).Render(bodyStyled)
}

func renderGroupHeader(g k8s.GroupedNodes) string {
	icon := groupIcon(g.Group.Provider)
	name := fmt.Sprintf("%s %s: %s", icon, providerLabel(g.Group.Provider), g.Group.DisplayName)
	taints := "no shared taints"
	if len(g.SharedTaints) > 0 {
		parts := make([]string, 0, len(g.SharedTaints))
		for _, t := range g.SharedTaints {
			parts = append(parts, t.String())
		}
		taints = "taints: " + strings.Join(parts, ", ")
	}
	return styles.GroupHeader.Render(name) + "  " + styles.NodeSub.Render(taints)
}

func groupIcon(p model.Provider) string {
	switch p {
	case model.ProviderKarpenter:
		return "◆"
	case model.ProviderEKS:
		return "▣"
	case model.ProviderGKE:
		return "▢"
	case model.ProviderAKS:
		return "▤"
	case model.ProviderRole:
		return "●"
	default:
		return "○"
	}
}

func providerLabel(p model.Provider) string {
	switch p {
	case model.ProviderKarpenter:
		return "NodePool"
	case model.ProviderEKS:
		return "EKS NodeGroup"
	case model.ProviderGKE:
		return "GKE NodePool"
	case model.ProviderAKS:
		return "AKS Agentpool"
	case model.ProviderRole:
		return "Role"
	default:
		return "Group"
	}
}

func gridRows(cards []string, cols int) string {
	if len(cards) == 0 {
		return ""
	}
	var rows []string
	for i := 0; i < len(cards); i += cols {
		end := i + cols
		if end > len(cards) {
			end = len(cards)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, interspersed(cards[i:end], " ")...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
