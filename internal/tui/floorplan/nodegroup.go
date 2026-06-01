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
// group; pass "" to focus nothing. When `collapsed` is set the body is hidden
// and only the header is drawn.
//
// It returns the rendered block plus the starting line (relative to the block's
// own top) of each focus target it contains — one entry per node when expanded,
// or a single entry for the header when collapsed.
func renderGroup(g k8s.GroupedNodes, pods []model.Pod, overhead map[string]analysis.NodeOverhead, d Density, availWidth int, focusedNode string, flagged map[string]bool, collapsed, groupFocused bool) (string, []int) {
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

	header := renderGroupHeader(g, collapsed, groupFocused)

	if collapsed {
		block := styles.Group.Width(innerWidth + 2).Render(header)
		// The single focus target is the header, one line below the top border.
		return block, []int{1}
	}

	cards := make([]string, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		cards = append(cards, renderNodeCard(n, model.PodsOnNode(pods, n.Name), overhead[n.Name], d, n.Name == focusedNode, flagged[n.Name]))
	}

	body, rowStarts := gridRows(cards, cols)
	bodyStyled := lipgloss.JoinVertical(lipgloss.Left, header, body)
	block := styles.Group.Width(innerWidth + 2).Render(bodyStyled)

	// Offsets: top border (1) + header height + the card's row offset.
	topBorder := 1
	headerH := lipgloss.Height(header)
	starts := make([]int, len(cards))
	for i := range cards {
		starts[i] = topBorder + headerH + rowStarts[i/cols]
	}
	return block, starts
}

func renderGroupHeader(g k8s.GroupedNodes, collapsed, focused bool) string {
	icon := groupIcon(g.Group.Provider)
	indicator := "▾"
	if collapsed {
		indicator = "▸"
	}
	name := fmt.Sprintf("%s %s %s: %s", indicator, icon, providerLabel(g.Group.Provider), g.Group.DisplayName)
	hs := styles.GroupHeader
	if focused {
		hs = hs.Foreground(styles.ColorAccent)
	}
	if collapsed {
		summary := fmt.Sprintf("(%d nodes — enter/x to expand)", len(g.Nodes))
		return hs.Render(name) + "  " + styles.NodeSub.Render(summary)
	}

	taints := "no shared taints"
	if len(g.SharedTaints) > 0 {
		parts := make([]string, 0, len(g.SharedTaints))
		for _, t := range g.SharedTaints {
			parts = append(parts, t.String())
		}
		taints = "taints: " + strings.Join(parts, ", ")
	}
	return hs.Render(name) + "  " + styles.NodeSub.Render(taints)
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

// gridRows arranges cards into rows of `cols` columns. It returns the joined
// body plus the starting line of each row (relative to the body's top), so
// callers can locate a card vertically for scroll-to-focus.
func gridRows(cards []string, cols int) (string, []int) {
	if len(cards) == 0 {
		return "", nil
	}
	var rows []string
	var rowStarts []int
	cum := 0
	for i := 0; i < len(cards); i += cols {
		end := i + cols
		if end > len(cards) {
			end = len(cards)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, interspersed(cards[i:end], " ")...)
		rowStarts = append(rowStarts, cum)
		cum += lipgloss.Height(row)
		rows = append(rows, row)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...), rowStarts
}
