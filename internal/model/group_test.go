package model

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func nodeWith(labels map[string]string, taints ...corev1.Taint) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Labels: labels},
		Spec:       corev1.NodeSpec{Taints: taints},
	}
}

func TestGroupOf(t *testing.T) {
	cases := []struct {
		name     string
		labels   map[string]string
		wantKey  string
		wantProv Provider
	}{
		{
			name:     "karpenter wins over everything",
			labels:   map[string]string{"karpenter.sh/nodepool": "system", "eks.amazonaws.com/nodegroup": "managed-a"},
			wantKey:  "karpenter:system",
			wantProv: ProviderKarpenter,
		},
		{
			name:     "eks managed node group",
			labels:   map[string]string{"eks.amazonaws.com/nodegroup": "managed-a"},
			wantKey:  "eks:managed-a",
			wantProv: ProviderEKS,
		},
		{
			name:     "gke node pool",
			labels:   map[string]string{"cloud.google.com/gke-nodepool": "default-pool"},
			wantKey:  "gke:default-pool",
			wantProv: ProviderGKE,
		},
		{
			name:     "aks agentpool",
			labels:   map[string]string{"agentpool": "nodepool1"},
			wantKey:  "aks:nodepool1",
			wantProv: ProviderAKS,
		},
		{
			name:     "role fallback",
			labels:   map[string]string{"node-role.kubernetes.io/control-plane": ""},
			wantKey:  "role:control-plane",
			wantProv: ProviderRole,
		},
		{
			name:     "ungrouped",
			labels:   map[string]string{"foo": "bar"},
			wantKey:  "ungrouped",
			wantProv: ProviderUngrouped,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := GroupOf(nodeWith(tc.labels))
			if g.Key != tc.wantKey {
				t.Errorf("Key: got %q want %q", g.Key, tc.wantKey)
			}
			if g.Provider != tc.wantProv {
				t.Errorf("Provider: got %q want %q", g.Provider, tc.wantProv)
			}
		})
	}
}

func TestSharedTaints_Intersection(t *testing.T) {
	a := Taint{Key: "critical", Effect: corev1.TaintEffectNoSchedule}
	b := Taint{Key: "spot", Effect: corev1.TaintEffectPreferNoSchedule}
	c := Taint{Key: "gpu", Effect: corev1.TaintEffectNoSchedule}

	nodes := []Node{
		{Taints: []Taint{a, b}},
		{Taints: []Taint{a, c}},
		{Taints: []Taint{a}},
	}
	got := SharedTaints(nodes)
	if len(got) != 1 || got[0] != a {
		t.Fatalf("expected only the common taint, got %v", got)
	}
}

func TestSharedTaints_EmptyInput(t *testing.T) {
	if got := SharedTaints(nil); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}
