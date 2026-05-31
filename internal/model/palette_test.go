package model

import "testing"

func TestColorFor_DeterministicAcrossCalls(t *testing.T) {
	k := WorkloadKey{Namespace: "kube-system", Kind: KindDeployment, Name: "coredns"}
	first := ColorFor(k)
	for i := 0; i < 10; i++ {
		if got := ColorFor(k); got != first {
			t.Fatalf("ColorFor not deterministic: %v vs %v", first, got)
		}
	}
}

func TestColorFor_DaemonSetIsReservedGreen(t *testing.T) {
	k := WorkloadKey{Namespace: "kube-system", Kind: KindDaemonSet, Name: "kube-proxy"}
	if got := ColorFor(k); got != DaemonSetColor {
		t.Errorf("expected DaemonSet to map to DaemonSetColor, got %v", got)
	}
}

func TestColorFor_DifferentWorkloadsCanShare(t *testing.T) {
	// With only ~10 palette slots, collisions are expected by design — this is
	// just a sanity check that the function returns _some_ palette colour.
	k := WorkloadKey{Namespace: "default", Kind: KindDeployment, Name: "api"}
	got := ColorFor(k)
	found := false
	for _, c := range WorkloadPalette {
		if c == got {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("returned colour %v not in palette", got)
	}
}
