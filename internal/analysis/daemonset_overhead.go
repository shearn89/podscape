// Package analysis contains pure functions that derive cluster-correctness
// signals from the internal model. Everything here is testable without a live
// API server: the floor-plan renderer calls these and visualises the results.
package analysis

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/shearn89/podscape/internal/model"
)

// NodeOverhead reports the DaemonSet footprint on a single node.
type NodeOverhead struct {
	NodeName       string
	DaemonSetPods  int
	CPURequest     resource.Quantity
	MemRequest     resource.Quantity
	CPUAllocatable resource.Quantity
	MemAllocatable resource.Quantity
}

// CPUPercent returns DS CPU request as a percent (0-100) of allocatable CPU.
// Returns 0 if allocatable is zero.
func (n NodeOverhead) CPUPercent() float64 {
	alloc := n.CPUAllocatable.MilliValue()
	if alloc <= 0 {
		return 0
	}
	return float64(n.CPURequest.MilliValue()) / float64(alloc) * 100
}

// MemPercent returns DS memory request as a percent (0-100) of allocatable mem.
func (n NodeOverhead) MemPercent() float64 {
	alloc := n.MemAllocatable.Value()
	if alloc <= 0 {
		return 0
	}
	return float64(n.MemRequest.Value()) / float64(alloc) * 100
}

// DaemonSetOverhead computes per-node DaemonSet resource overhead.
//
// Pods are attributed to a node by .Spec.NodeName; only pods whose Owner kind
// is DaemonSet count. Containers without requests contribute zero (matching
// the kube scheduler's view).
func DaemonSetOverhead(nodes []model.Node, pods []model.Pod) map[string]NodeOverhead {
	out := make(map[string]NodeOverhead, len(nodes))
	for _, n := range nodes {
		out[n.Name] = NodeOverhead{
			NodeName:       n.Name,
			CPUAllocatable: quantityOf(n.Allocatable, corev1.ResourceCPU),
			MemAllocatable: quantityOf(n.Allocatable, corev1.ResourceMemory),
		}
	}
	for _, p := range pods {
		if p.Owner.Kind != model.KindDaemonSet || p.NodeName == "" {
			continue
		}
		entry, ok := out[p.NodeName]
		if !ok {
			// Pod scheduled to a node we don't know about — count it under that
			// name so the user still sees something, but allocatable is zero.
			entry = NodeOverhead{NodeName: p.NodeName}
		}
		cpu, mem := model.SumRequests(p.Containers)
		entry.CPURequest.Add(cpu)
		entry.MemRequest.Add(mem)
		entry.DaemonSetPods++
		out[p.NodeName] = entry
	}
	return out
}

func quantityOf(rl corev1.ResourceList, name corev1.ResourceName) resource.Quantity {
	if rl == nil {
		return resource.Quantity{}
	}
	if q, ok := rl[name]; ok {
		return q
	}
	return resource.Quantity{}
}
