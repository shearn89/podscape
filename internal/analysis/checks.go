package analysis

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/shearn89/podscape/internal/model"
)

// Severity grades a Finding.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarn
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarn:
		return "warn"
	default:
		return "info"
	}
}

// Finding is one correctness signal — something the user probably wants to
// look at. Findings target a specific node and (optionally) a specific pod, so
// the TUI can attach a marker to the right card.
type Finding struct {
	Severity Severity
	Code     string // stable machine-readable code (DS_NO_PRIORITY, TOLERATION_UNUSED, ANTIAFFINITY_VIOLATED)
	Message  string
	Node     string // node name (empty when the finding is workload-wide)
	Pod      string // namespace/name of the affected pod, when applicable
}

// RunChecks runs every check against the snapshot and returns a sorted
// findings list (error-first, then warn, then info; ties broken by code/node/pod).
func RunChecks(nodes []model.Node, pods []model.Pod) []Finding {
	byName := make(map[string]model.Node, len(nodes))
	for _, n := range nodes {
		byName[n.Name] = n
	}
	out := make([]Finding, 0, 16)
	out = append(out, DaemonSetMissingPriorityClass(pods)...)
	out = append(out, TolerationWithoutMatchingTaint(pods, byName)...)
	out = append(out, RequiredAntiAffinityViolations(pods)...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity > out[j].Severity
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		if out[i].Node != out[j].Node {
			return out[i].Node < out[j].Node
		}
		return out[i].Pod < out[j].Pod
	})
	return out
}

// DaemonSetMissingPriorityClass flags DaemonSet pods that lack a
// PriorityClassName. The check is deduplicated per (namespace, ds-name) so a
// 100-node DaemonSet doesn't produce 100 identical findings.
func DaemonSetMissingPriorityClass(pods []model.Pod) []Finding {
	seen := map[string]bool{}
	var out []Finding
	for _, p := range pods {
		if p.Owner.Kind != model.KindDaemonSet {
			continue
		}
		if p.Profile.PriorityClass != "" {
			continue
		}
		key := p.Namespace + "/" + p.Owner.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Finding{
			Severity: SeverityWarn,
			Code:     "DS_NO_PRIORITY",
			Message:  fmt.Sprintf("DaemonSet %s/%s has no priorityClassName — set system-node-critical (or similar) so it isn't evicted under pressure", p.Namespace, p.Owner.Name),
			Pod:      p.Namespace + "/" + p.Name,
		})
	}
	return out
}

// kubeletInjectedTolerations are tolerations the kubelet / controller manager
// add automatically. They reflect cluster operational concerns, not the user's
// intent to schedule on a tainted node, so they should not trigger
// "useless toleration" findings.
var kubeletInjectedTolerations = map[string]bool{
	"node.kubernetes.io/not-ready":           true,
	"node.kubernetes.io/unreachable":         true,
	"node.kubernetes.io/memory-pressure":     true,
	"node.kubernetes.io/disk-pressure":       true,
	"node.kubernetes.io/pid-pressure":        true,
	"node.kubernetes.io/network-unavailable": true,
	"node.kubernetes.io/unschedulable":       true,
	"node.kubernetes.io/out-of-disk":         true,
}

// TolerationWithoutMatchingTaint flags pods whose tolerations don't match any
// taint on their current node. The intuition: if a user added a toleration to
// a workload, they probably meant to land it on a node bearing that taint. If
// the pod is currently on an *untainted* node (or one with a different taint
// set), the toleration is a configuration smell — either the workload landed
// in the wrong pool, or the toleration is dead config.
//
// Tolerations that match nothing (empty key, operator=Exists with no key, or
// kubelet-injected operational tolerations) are skipped.
func TolerationWithoutMatchingTaint(pods []model.Pod, nodesByName map[string]model.Node) []Finding {
	var out []Finding
	for _, p := range pods {
		if p.NodeName == "" {
			continue
		}
		// Daemonsets implicitly tolerate everything via the DS controller —
		// reporting on them just creates noise.
		if p.Owner.Kind == model.KindDaemonSet {
			continue
		}
		node, ok := nodesByName[p.NodeName]
		if !ok {
			continue
		}
		var unused []string
		for _, tol := range p.Profile.Tolerations {
			if isKubeletInjected(tol) {
				continue
			}
			if tol.Key == "" && tol.Operator == corev1.TolerationOpExists {
				// universal-tolerate; matches anything, can't be unused.
				continue
			}
			if !tolerationMatchesAnyTaint(tol, node.Taints) {
				unused = append(unused, describeToleration(tol))
			}
		}
		if len(unused) == 0 {
			continue
		}
		out = append(out, Finding{
			Severity: SeverityWarn,
			Code:     "TOLERATION_UNUSED",
			Message: fmt.Sprintf("pod %s/%s tolerates %s but its node %s has no matching taint — probably landed in the wrong node pool",
				p.Namespace, p.Name, strings.Join(unused, ", "), p.NodeName),
			Node: p.NodeName,
			Pod:  p.Namespace + "/" + p.Name,
		})
	}
	return out
}

func isKubeletInjected(t model.Toleration) bool {
	return kubeletInjectedTolerations[t.Key]
}

func tolerationMatchesAnyTaint(t model.Toleration, taints []model.Taint) bool {
	for _, taint := range taints {
		if tolerationMatchesTaint(t, taint) {
			return true
		}
	}
	return false
}

// tolerationMatchesTaint implements the k8s toleration-matching rules
// (effect + operator + key + value); it ignores time-based eviction.
func tolerationMatchesTaint(t model.Toleration, taint model.Taint) bool {
	if t.Effect != "" && t.Effect != taint.Effect {
		return false
	}
	switch t.Operator {
	case corev1.TolerationOpExists, "":
		// Empty operator defaults to Equal per the k8s API, but the API also
		// permits "Exists" with empty key to match all — keep behaviour
		// permissive and treat empty-key Exists as "tolerates anything".
		if t.Operator == corev1.TolerationOpExists {
			if t.Key == "" {
				return true
			}
			return t.Key == taint.Key
		}
		// Default Equal:
		return t.Key == taint.Key && t.Value == taint.Value
	case corev1.TolerationOpEqual:
		return t.Key == taint.Key && t.Value == taint.Value
	}
	return false
}

func describeToleration(t model.Toleration) string {
	if t.Operator == corev1.TolerationOpExists {
		if t.Key == "" {
			return "any taint"
		}
		return t.Key + " exists"
	}
	if t.Value != "" {
		return t.Key + "=" + t.Value
	}
	return t.Key
}

// RequiredAntiAffinityViolations flags pods that declare a required
// PodAntiAffinity at the hostname topology key but have landed on the same
// node as at least one peer matching the selector. (Soft anti-affinity is
// intentionally ignored — it's a preference, not a contract.)
//
// In a healthy cluster the scheduler enforces this, but it slips through
// during scale-up if the scheduler couldn't satisfy the constraint and the
// PodSpec was edited afterwards, or via priority-class preemption.
func RequiredAntiAffinityViolations(pods []model.Pod) []Finding {
	// index pods by node for O(N) lookup.
	byNode := map[string][]int{}
	for i, p := range pods {
		if p.NodeName == "" {
			continue
		}
		byNode[p.NodeName] = append(byNode[p.NodeName], i)
	}

	var out []Finding
	seen := map[string]bool{}
	for i, p := range pods {
		if p.Profile.Affinity == nil || len(p.Profile.Affinity.RequiredPodAntiAffinity) == 0 {
			continue
		}
		peers := byNode[p.NodeName]
		for _, term := range p.Profile.Affinity.RequiredPodAntiAffinity {
			if term.TopologyKey != corev1.LabelHostname {
				// Other topology keys (zone, region, …) need node-label data
				// we don't fully resolve here; the hostname case is the one
				// users hit in practice.
				continue
			}
			for _, j := range peers {
				if i == j {
					continue
				}
				peer := pods[j]
				if !selectorMatches(term, p, peer) {
					continue
				}
				key := pairKey(p, peer, p.NodeName)
				if seen[key] {
					continue
				}
				seen[key] = true
				out = append(out, Finding{
					Severity: SeverityError,
					Code:     "ANTIAFFINITY_VIOLATED",
					Message: fmt.Sprintf("pod %s/%s has required hostname anti-affinity but shares node %s with peer %s/%s",
						p.Namespace, p.Name, p.NodeName, peer.Namespace, peer.Name),
					Node: p.NodeName,
					Pod:  p.Namespace + "/" + p.Name,
				})
			}
		}
	}
	return out
}

// selectorMatches reports whether `peer` matches the term's selector. The
// current implementation honours `matchLabels` only — `matchExpressions` is
// deliberately out of scope for the first pass. Workloads using matchLabels
// (the common case for "spread my replicas") are detected; matchExpressions
// selectors will simply not trigger findings, which is a safe default.
func selectorMatches(term model.PodAffinityTerm, self, peer model.Pod) bool {
	if !inNamespaceScope(term.Namespaces, self.Namespace, peer.Namespace) {
		return false
	}
	if len(term.MatchLabels) == 0 {
		// An empty selector in k8s matches *all* pods; preserve that here.
		return true
	}
	for k, v := range term.MatchLabels {
		if peer.Labels[k] != v {
			return false
		}
	}
	return true
}

func inNamespaceScope(termNamespaces []string, selfNS, peerNS string) bool {
	if len(termNamespaces) == 0 {
		// k8s default: matches pods in the same namespace as the source pod.
		return selfNS == peerNS
	}
	for _, n := range termNamespaces {
		if n == peerNS {
			return true
		}
	}
	return false
}

func pairKey(a, b model.Pod, node string) string {
	x, y := a.Namespace+"/"+a.Name, b.Namespace+"/"+b.Name
	if x > y {
		x, y = y, x
	}
	return node + "|" + x + "|" + y
}
