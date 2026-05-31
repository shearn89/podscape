package k8s

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/shearn89/podscape/internal/model"
)

// Snapshot is a single point-in-time view of the cluster used by the TUI.
type Snapshot struct {
	Nodes  []model.Node
	Pods   []model.Pod
	Groups []GroupedNodes
}

// GroupedNodes is a NodeGroup with its member nodes, ready for rendering.
type GroupedNodes struct {
	Group        model.NodeGroup
	Nodes        []model.Node
	SharedTaints []model.Taint
}

// rsResolver is an OwnerResolver backed by a ReplicaSet list.
type rsResolver map[string]string

func (r rsResolver) ResolveReplicaSet(ns, name string) (string, bool) {
	v, ok := r[ns+"/"+name]
	return v, ok
}

// Fetch retrieves the full topology in one shot. Intended for periodic
// refreshes triggered by the TUI's tick / 'r' keybind. The optional `namespace`
// argument scopes the pod list (empty = all namespaces).
func Fetch(ctx context.Context, cs kubernetes.Interface, namespace string) (*Snapshot, error) {
	nodeList, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	rsList, err := cs.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list replicasets: %w", err)
	}
	resolver := rsResolver{}
	for _, rs := range rsList.Items {
		if owner := controllerOf(rs.OwnerReferences); owner != nil && owner.Kind == "Deployment" {
			resolver[rs.Namespace+"/"+rs.Name] = owner.Name
		}
	}
	podList, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	nodes := make([]model.Node, 0, len(nodeList.Items))
	for i := range nodeList.Items {
		nodes = append(nodes, model.FromNode(&nodeList.Items[i]))
	}
	pods := make([]model.Pod, 0, len(podList.Items))
	for i := range podList.Items {
		pods = append(pods, model.FromPod(&podList.Items[i], resolver))
	}

	return &Snapshot{
		Nodes:  nodes,
		Pods:   pods,
		Groups: groupNodes(nodes),
	}, nil
}

func controllerOf(refs []metav1.OwnerReference) *metav1.OwnerReference {
	for i, r := range refs {
		if r.Controller != nil && *r.Controller {
			return &refs[i]
		}
	}
	return nil
}

// groupNodes buckets nodes by their NodeGroup and sorts everything stably so
// the rendered floor plan is deterministic across refreshes.
func groupNodes(nodes []model.Node) []GroupedNodes {
	buckets := map[string]*GroupedNodes{}
	for _, n := range nodes {
		g, ok := buckets[n.Group.Key]
		if !ok {
			g = &GroupedNodes{Group: n.Group}
			buckets[n.Group.Key] = g
		}
		g.Nodes = append(g.Nodes, n)
	}
	out := make([]GroupedNodes, 0, len(buckets))
	for _, g := range buckets {
		sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].Name < g.Nodes[j].Name })
		g.SharedTaints = model.SharedTaints(g.Nodes)
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Group.Key < out[j].Group.Key })
	return out
}
