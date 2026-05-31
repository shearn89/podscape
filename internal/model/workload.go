package model

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Kind identifies the controller kind that owns a pod.
type Kind string

const (
	KindDeployment  Kind = "Deployment"
	KindStatefulSet Kind = "StatefulSet"
	KindDaemonSet   Kind = "DaemonSet"
	KindJob         Kind = "Job"
	KindReplicaSet  Kind = "ReplicaSet"
	KindStandalone  Kind = "Pod"
)

// WorkloadKey uniquely identifies a workload for colouring and grouping.
type WorkloadKey struct {
	Namespace string
	Kind      Kind
	Name      string
}

func (k WorkloadKey) String() string {
	return string(k.Kind) + "/" + k.Namespace + "/" + k.Name
}

// Toleration is a trimmed-down view of corev1.Toleration.
type Toleration struct {
	Key      string
	Operator corev1.TolerationOperator
	Value    string
	Effect   corev1.TaintEffect
}

// AffinitySummary captures the practically-useful bits of pod/node affinity
// without dragging the full PodAffinity API into the renderer.
type AffinitySummary struct {
	RequiredNodeMatch  []string // human-readable required matchExpressions
	PreferredNodeMatch []string
	HasPodAffinity     bool
	HasPodAntiAffinity bool

	// RequiredPodAntiAffinity captures requiredDuringScheduling pod
	// anti-affinity terms. Each term gives us the topology key (typically
	// kubernetes.io/hostname) and the label selector that identifies the
	// peer pods we must not be co-located with — enough to detect when a
	// pod with hostname-level anti-affinity has landed on the same node as
	// a matching peer.
	RequiredPodAntiAffinity []PodAffinityTerm
}

// PodAffinityTerm is the model's view of corev1.PodAffinityTerm — only the
// fields the checks need. Namespaces left empty means "the pod's own
// namespace" per the k8s API contract.
type PodAffinityTerm struct {
	TopologyKey string
	MatchLabels map[string]string
	Namespaces  []string
}

type SchedulingProfile struct {
	PriorityClass string
	Tolerations   []Toleration
	NodeSelector  map[string]string
	Affinity      *AffinitySummary
}

type Container struct {
	Name     string
	Requests corev1.ResourceList
	Limits   corev1.ResourceList
}

// SumRequests returns the total CPU / memory requested by a slice of containers.
func SumRequests(containers []Container) (cpu, mem resource.Quantity) {
	for _, c := range containers {
		if q, ok := c.Requests[corev1.ResourceCPU]; ok {
			cpu.Add(q)
		}
		if q, ok := c.Requests[corev1.ResourceMemory]; ok {
			mem.Add(q)
		}
	}
	return cpu, mem
}

type Pod struct {
	Namespace  string
	Name       string
	NodeName   string
	Phase      corev1.PodPhase
	Labels     map[string]string
	Owner      WorkloadKey
	Profile    SchedulingProfile
	Containers []Container
}

type Workload struct {
	Key      WorkloadKey
	Replicas int32
	Profile  SchedulingProfile
	Pods     []Pod
}

// PodsOnNode returns the subset of pods scheduled on the given node name. It
// is intentionally cheap and O(N) — the TUI calls it per-card per-refresh.
func PodsOnNode(pods []Pod, nodeName string) []Pod {
	out := make([]Pod, 0, 8)
	for _, p := range pods {
		if p.NodeName == nodeName {
			out = append(out, p)
		}
	}
	return out
}
