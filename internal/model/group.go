package model

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
)

// Provider identifies where a node-group came from.
type Provider string

const (
	ProviderKarpenter Provider = "karpenter"
	ProviderEKS       Provider = "eks"
	ProviderGKE       Provider = "gke"
	ProviderAKS       Provider = "aks"
	ProviderRole      Provider = "role"
	ProviderUngrouped Provider = "ungrouped"
)

// NodeGroup is the visual grouping bucket for nodes on the floor plan.
type NodeGroup struct {
	Key         string   // stable identifier used for sorting / dedup
	DisplayName string   // shown in the group header
	Provider    Provider // for icon / label
}

// nodeGroupLabels are the labels (in priority order) used to derive the group
// a node belongs to. The first label that is set on the node wins.
var nodeGroupLabels = []struct {
	label    string
	provider Provider
}{
	{"karpenter.sh/nodepool", ProviderKarpenter},
	{"eks.amazonaws.com/nodegroup", ProviderEKS},
	{"cloud.google.com/gke-nodepool", ProviderGKE},
	{"agentpool", ProviderAKS},
}

// GroupOf derives a NodeGroup from a Kubernetes node's labels.
//
// Resolution order: karpenter → eks → gke → aks → role label → ungrouped.
func GroupOf(node *corev1.Node) NodeGroup {
	labels := node.Labels
	for _, candidate := range nodeGroupLabels {
		if v, ok := labels[candidate.label]; ok && v != "" {
			return NodeGroup{
				Key:         string(candidate.provider) + ":" + v,
				DisplayName: v,
				Provider:    candidate.provider,
			}
		}
	}
	// Fallback: pick the highest-precedence node-role label.
	const rolePrefix = "node-role.kubernetes.io/"
	var roles []string
	for k := range labels {
		if len(k) > len(rolePrefix) && k[:len(rolePrefix)] == rolePrefix {
			roles = append(roles, k[len(rolePrefix):])
		}
	}
	if len(roles) > 0 {
		sort.Strings(roles)
		return NodeGroup{
			Key:         string(ProviderRole) + ":" + roles[0],
			DisplayName: roles[0],
			Provider:    ProviderRole,
		}
	}
	return NodeGroup{
		Key:         string(ProviderUngrouped),
		DisplayName: "ungrouped",
		Provider:    ProviderUngrouped,
	}
}

// SharedTaints returns the intersection of taints across the given nodes — only
// taints present on every member are returned. Used as the node-group header
// taint summary (per-node extras render on each card).
func SharedTaints(nodes []Node) []Taint {
	if len(nodes) == 0 {
		return nil
	}
	counts := map[Taint]int{}
	for _, n := range nodes {
		seen := map[Taint]bool{}
		for _, t := range n.Taints {
			if !seen[t] {
				counts[t]++
				seen[t] = true
			}
		}
	}
	var out []Taint
	for t, c := range counts {
		if c == len(nodes) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].String() < out[j].String()
	})
	return out
}
