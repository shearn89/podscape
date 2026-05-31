package analysis

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/shearn89/podscape/internal/model"
)

func mustQ(s string) resource.Quantity { return resource.MustParse(s) }

func node(name, cpu, mem string) model.Node {
	return model.Node{
		Name: name,
		Allocatable: corev1.ResourceList{
			corev1.ResourceCPU:    mustQ(cpu),
			corev1.ResourceMemory: mustQ(mem),
		},
	}
}

func dsPod(name, nodeName string, containers ...model.Container) model.Pod {
	return model.Pod{
		Name:       name,
		NodeName:   nodeName,
		Owner:      model.WorkloadKey{Kind: model.KindDaemonSet, Name: "ds-" + name},
		Containers: containers,
	}
}

func cnt(cpu, mem string) model.Container {
	return model.Container{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    mustQ(cpu),
			corev1.ResourceMemory: mustQ(mem),
		},
	}
}

func TestDaemonSetOverhead_SumsRequestsPerNode(t *testing.T) {
	// vary cpu allocatable so the helper isn't flagged by unparam.
	nodes := []model.Node{node("n1", "2", "8Gi"), node("n2", "4", "16Gi")}
	pods := []model.Pod{
		dsPod("kp1", "n1", cnt("100m", "128Mi")),
		dsPod("kp2", "n2", cnt("100m", "128Mi")),
		dsPod("csi1", "n1", cnt("50m", "64Mi"), cnt("50m", "64Mi")), // multi-container
	}
	got := DaemonSetOverhead(nodes, pods)

	n1 := got["n1"]
	if n1.DaemonSetPods != 2 {
		t.Errorf("n1 DS pods = %d, want 2", n1.DaemonSetPods)
	}
	if n1.CPURequest.MilliValue() != 200 {
		t.Errorf("n1 cpu = %dm, want 200m", n1.CPURequest.MilliValue())
	}
	wantMem := mustQ("256Mi")
	if n1.MemRequest.Cmp(wantMem) != 0 {
		t.Errorf("n1 mem = %s, want %s", n1.MemRequest.String(), wantMem.String())
	}

	n2 := got["n2"]
	if n2.DaemonSetPods != 1 || n2.CPURequest.MilliValue() != 100 {
		t.Errorf("n2 unexpected: %+v", n2)
	}
}

func TestDaemonSetOverhead_NonDaemonSetPodsIgnored(t *testing.T) {
	nodes := []model.Node{node("n1", "2", "8Gi")}
	pods := []model.Pod{
		{NodeName: "n1", Owner: model.WorkloadKey{Kind: model.KindDeployment, Name: "api"}, Containers: []model.Container{cnt("500m", "512Mi")}},
		dsPod("kp", "n1", cnt("100m", "128Mi")),
	}
	got := DaemonSetOverhead(nodes, pods)
	n1 := got["n1"]
	if n1.DaemonSetPods != 1 || n1.CPURequest.MilliValue() != 100 {
		t.Errorf("Deployment pod leaked into DS overhead: %+v", n1)
	}
}

func TestDaemonSetOverhead_NodeWithoutDSPods(t *testing.T) {
	nodes := []model.Node{node("n1", "2", "8Gi")}
	got := DaemonSetOverhead(nodes, nil)
	if got["n1"].DaemonSetPods != 0 || got["n1"].CPUPercent() != 0 {
		t.Errorf("expected zero overhead, got %+v", got["n1"])
	}
}

func TestDaemonSetOverhead_PodWithoutRequests(t *testing.T) {
	nodes := []model.Node{node("n1", "2", "8Gi")}
	pods := []model.Pod{dsPod("bare", "n1", model.Container{})}
	got := DaemonSetOverhead(nodes, pods)
	n1 := got["n1"]
	if n1.DaemonSetPods != 1 {
		t.Errorf("pod with no requests should still count: %+v", n1)
	}
	if n1.CPURequest.MilliValue() != 0 {
		t.Errorf("no requests should sum to zero: %dm", n1.CPURequest.MilliValue())
	}
}

func TestNodeOverhead_Percents(t *testing.T) {
	n := NodeOverhead{
		CPURequest:     mustQ("500m"),
		MemRequest:     mustQ("512Mi"),
		CPUAllocatable: mustQ("2"),
		MemAllocatable: mustQ("4Gi"),
	}
	if got := n.CPUPercent(); got < 24.9 || got > 25.1 {
		t.Errorf("CPUPercent = %v, want ~25", got)
	}
	if got := n.MemPercent(); got < 12.4 || got > 12.6 {
		t.Errorf("MemPercent = %v, want ~12.5", got)
	}
}

func TestNodeOverhead_ZeroAllocatable(t *testing.T) {
	n := NodeOverhead{CPURequest: mustQ("100m")}
	if got := n.CPUPercent(); got != 0 {
		t.Errorf("CPUPercent on zero allocatable = %v, want 0", got)
	}
}

func TestDaemonSetOverhead_PodOnUnknownNode(t *testing.T) {
	pods := []model.Pod{dsPod("kp", "ghost", cnt("100m", "128Mi"))}
	got := DaemonSetOverhead(nil, pods)
	if got["ghost"].DaemonSetPods != 1 {
		t.Errorf("expected to record overhead under unknown node, got %+v", got)
	}
}
