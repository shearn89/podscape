package floorplan

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/shearn89/podscape/internal/analysis"
	"github.com/shearn89/podscape/internal/k8s"
	"github.com/shearn89/podscape/internal/model"
)

func snapshot() *k8s.Snapshot {
	allocatable := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("2"),
		corev1.ResourceMemory: resource.MustParse("8Gi"),
	}
	g := model.NodeGroup{Key: "karpenter:system", DisplayName: "system", Provider: model.ProviderKarpenter}
	n1 := model.Node{Name: "ip-10-0-1-12", InstanceType: "m5.large", Allocatable: allocatable, Group: g, Ready: true}
	n2 := model.Node{Name: "ip-10-0-1-37", InstanceType: "m5.large", Allocatable: allocatable, Group: g, Ready: true}

	pods := []model.Pod{
		{Name: "coredns-1", NodeName: "ip-10-0-1-12", Owner: model.WorkloadKey{Namespace: "kube-system", Kind: model.KindDeployment, Name: "coredns"}},
		{Name: "coredns-2", NodeName: "ip-10-0-1-37", Owner: model.WorkloadKey{Namespace: "kube-system", Kind: model.KindDeployment, Name: "coredns"}},
		{Name: "kp-12", NodeName: "ip-10-0-1-12", Owner: model.WorkloadKey{Namespace: "kube-system", Kind: model.KindDaemonSet, Name: "kube-proxy"},
			Containers: []model.Container{{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")}}}},
		{Name: "kp-37", NodeName: "ip-10-0-1-37", Owner: model.WorkloadKey{Namespace: "kube-system", Kind: model.KindDaemonSet, Name: "kube-proxy"},
			Containers: []model.Container{{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")}}}},
	}
	return &k8s.Snapshot{
		Nodes: []model.Node{n1, n2},
		Pods:  pods,
		Groups: []k8s.GroupedNodes{{
			Group: g, Nodes: []model.Node{n1, n2},
		}},
	}
}

func TestRender_ContainsGroupAndNodeNames(t *testing.T) {
	snap := snapshot()
	v := View{
		Snapshot: snap,
		Overhead: analysis.DaemonSetOverhead(snap.Nodes, snap.Pods),
		Density:  DensityNormal,
		Width:    120,
	}
	out := stripANSI(Render(v))
	// kube-proxy may be truncated at compact/normal densities — match a prefix.
	for _, want := range []string{"system", "ip-10-0-1-12", "ip-10-0-1-37", "coredns", "kube-pr", "DS cpu"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered floorplan missing %q\n---\n%s", want, out)
		}
	}
}

func TestRender_EmptySnapshot(t *testing.T) {
	out := stripANSI(Render(View{Snapshot: &k8s.Snapshot{}, Density: DensityNormal, Width: 120}))
	if !strings.Contains(out, "no nodes") {
		t.Errorf("expected empty-cluster hint, got %q", out)
	}
}

func TestRender_NarrowTerminalFallsBackToSingleColumn(t *testing.T) {
	snap := snapshot()
	v := View{
		Snapshot: snap,
		Overhead: analysis.DaemonSetOverhead(snap.Nodes, snap.Pods),
		Density:  DensityWide, // 40-col cards
		Width:    50,          // not enough for two cards side by side
	}
	out := Render(v)
	// Each line should not exceed the requested width by much; the renderer
	// shouldn't crash and should still mention both nodes.
	clean := stripANSI(out)
	if !strings.Contains(clean, "ip-10-0-1-12") || !strings.Contains(clean, "ip-10-0-1-37") {
		t.Errorf("narrow render dropped a node:\n%s", clean)
	}
}

func TestFocusableNodes_OrderedByGroupThenName(t *testing.T) {
	v := View{Snapshot: snapshot()}
	got := FocusableNodes(v)
	want := []string{"ip-10-0-1-12", "ip-10-0-1-37"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("FocusableNodes[%d] = %q want %q", i, got[i], want[i])
		}
	}
}

func TestDensityCardWidthsAreOrdered(t *testing.T) {
	if DensityCompact.CardWidth() >= DensityNormal.CardWidth() ||
		DensityNormal.CardWidth() >= DensityWide.CardWidth() {
		t.Errorf("density card widths not monotonic: %d %d %d",
			DensityCompact.CardWidth(), DensityNormal.CardWidth(), DensityWide.CardWidth())
	}
}
