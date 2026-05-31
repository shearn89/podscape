package model

import (
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FromNode converts a corev1.Node into our internal Node representation.
func FromNode(n *corev1.Node) Node {
	taints := make([]Taint, 0, len(n.Spec.Taints))
	for _, t := range n.Spec.Taints {
		taints = append(taints, Taint{Key: t.Key, Value: t.Value, Effect: t.Effect})
	}
	sort.Slice(taints, func(i, j int) bool { return taints[i].String() < taints[j].String() })

	ready := false
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
			ready = true
			break
		}
	}

	instanceType := n.Labels["node.kubernetes.io/instance-type"]
	if instanceType == "" {
		instanceType = n.Labels["beta.kubernetes.io/instance-type"]
	}

	return Node{
		Name:         n.Name,
		InstanceType: instanceType,
		Taints:       taints,
		Labels:       n.Labels,
		Allocatable:  n.Status.Allocatable,
		Capacity:     n.Status.Capacity,
		Group:        GroupOf(n),
		Ready:        ready,
	}
}

// FromPod converts a corev1.Pod into our internal Pod representation. The
// resolver is consulted to convert ReplicaSet owners into their Deployment
// parents — if nil or no match is found, the ReplicaSet itself is recorded.
func FromPod(p *corev1.Pod, resolver OwnerResolver) Pod {
	owner := ownerOf(p, resolver)
	containers := make([]Container, 0, len(p.Spec.Containers))
	for _, c := range p.Spec.Containers {
		containers = append(containers, Container{
			Name:     c.Name,
			Requests: c.Resources.Requests,
			Limits:   c.Resources.Limits,
		})
	}
	return Pod{
		Namespace:  p.Namespace,
		Name:       p.Name,
		NodeName:   p.Spec.NodeName,
		Phase:      p.Status.Phase,
		Labels:     p.Labels,
		Owner:      owner,
		Profile:    schedulingProfile(p),
		Containers: containers,
	}
}

// OwnerResolver looks up Deployment names for ReplicaSets so pods owned by a
// ReplicaSet are attributed to their parent Deployment.
type OwnerResolver interface {
	ResolveReplicaSet(namespace, name string) (deploymentName string, ok bool)
}

func ownerOf(p *corev1.Pod, resolver OwnerResolver) WorkloadKey {
	for _, ref := range p.OwnerReferences {
		if ref.Controller != nil && !*ref.Controller {
			continue
		}
		switch ref.Kind {
		case "DaemonSet":
			return WorkloadKey{Namespace: p.Namespace, Kind: KindDaemonSet, Name: ref.Name}
		case "StatefulSet":
			return WorkloadKey{Namespace: p.Namespace, Kind: KindStatefulSet, Name: ref.Name}
		case "Job":
			return WorkloadKey{Namespace: p.Namespace, Kind: KindJob, Name: ref.Name}
		case "ReplicaSet":
			if resolver != nil {
				if dep, ok := resolver.ResolveReplicaSet(p.Namespace, ref.Name); ok {
					return WorkloadKey{Namespace: p.Namespace, Kind: KindDeployment, Name: dep}
				}
			}
			return WorkloadKey{Namespace: p.Namespace, Kind: KindReplicaSet, Name: ref.Name}
		}
	}
	return WorkloadKey{Namespace: p.Namespace, Kind: KindStandalone, Name: p.Name}
}

func schedulingProfile(p *corev1.Pod) SchedulingProfile {
	tolerations := make([]Toleration, 0, len(p.Spec.Tolerations))
	for _, t := range p.Spec.Tolerations {
		tolerations = append(tolerations, Toleration{
			Key: t.Key, Operator: t.Operator, Value: t.Value, Effect: t.Effect,
		})
	}
	return SchedulingProfile{
		PriorityClass: p.Spec.PriorityClassName,
		Tolerations:   tolerations,
		NodeSelector:  p.Spec.NodeSelector,
		Affinity:      affinitySummary(p.Spec.Affinity),
	}
}

func affinitySummary(a *corev1.Affinity) *AffinitySummary {
	if a == nil {
		return nil
	}
	out := &AffinitySummary{}
	if na := a.NodeAffinity; na != nil {
		if req := na.RequiredDuringSchedulingIgnoredDuringExecution; req != nil {
			for _, term := range req.NodeSelectorTerms {
				for _, expr := range term.MatchExpressions {
					out.RequiredNodeMatch = append(out.RequiredNodeMatch, exprString(expr))
				}
			}
		}
		for _, pref := range na.PreferredDuringSchedulingIgnoredDuringExecution {
			for _, expr := range pref.Preference.MatchExpressions {
				out.PreferredNodeMatch = append(out.PreferredNodeMatch, exprString(expr))
			}
		}
	}
	out.HasPodAffinity = a.PodAffinity != nil
	out.HasPodAntiAffinity = a.PodAntiAffinity != nil
	if a.PodAntiAffinity != nil {
		for _, term := range a.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
			out.RequiredPodAntiAffinity = append(out.RequiredPodAntiAffinity, PodAffinityTerm{
				TopologyKey: term.TopologyKey,
				MatchLabels: matchLabels(term.LabelSelector),
				Namespaces:  term.Namespaces,
			})
		}
	}
	if len(out.RequiredNodeMatch) == 0 && len(out.PreferredNodeMatch) == 0 &&
		!out.HasPodAffinity && !out.HasPodAntiAffinity {
		return nil
	}
	return out
}

func matchLabels(s *metav1.LabelSelector) map[string]string {
	if s == nil {
		return nil
	}
	if len(s.MatchLabels) == 0 {
		return nil
	}
	out := make(map[string]string, len(s.MatchLabels))
	for k, v := range s.MatchLabels {
		out[k] = v
	}
	return out
}

func exprString(e corev1.NodeSelectorRequirement) string {
	switch e.Operator {
	case corev1.NodeSelectorOpExists:
		return e.Key + " exists"
	case corev1.NodeSelectorOpDoesNotExist:
		return "!" + e.Key
	case corev1.NodeSelectorOpIn:
		return fmt.Sprintf("%s in [%s]", e.Key, joinValues(e.Values))
	case corev1.NodeSelectorOpNotIn:
		return fmt.Sprintf("%s notIn [%s]", e.Key, joinValues(e.Values))
	default:
		return fmt.Sprintf("%s %s [%s]", e.Key, e.Operator, joinValues(e.Values))
	}
}

func joinValues(v []string) string {
	out := ""
	for i, s := range v {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}
