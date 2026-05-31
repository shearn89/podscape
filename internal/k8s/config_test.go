package k8s

import (
	"testing"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func cfg(contexts ...string) *clientcmdapi.Config {
	c := clientcmdapi.NewConfig()
	for _, n := range contexts {
		c.Contexts[n] = clientcmdapi.NewContext()
	}
	return c
}

func TestResolveContext_Explicit(t *testing.T) {
	c := cfg("a", "b")
	r, err := ResolveContext(c, "b")
	if err != nil || r.Context != "b" || r.NeedsPicker {
		t.Fatalf("got %+v err=%v", r, err)
	}
}

func TestResolveContext_ExplicitUnknown(t *testing.T) {
	c := cfg("a")
	if _, err := ResolveContext(c, "missing"); err == nil {
		t.Errorf("expected error for unknown context")
	}
}

func TestResolveContext_FallsBackToCurrent(t *testing.T) {
	c := cfg("a", "b")
	c.CurrentContext = "a"
	r, err := ResolveContext(c, "")
	if err != nil || r.Context != "a" {
		t.Fatalf("got %+v err=%v", r, err)
	}
}

func TestResolveContext_PickerWhenNoCurrent(t *testing.T) {
	c := cfg("a", "b")
	r, err := ResolveContext(c, "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !r.NeedsPicker {
		t.Errorf("expected NeedsPicker, got %+v", r)
	}
}

func TestResolveContext_EmptyConfig(t *testing.T) {
	c := cfg()
	if _, err := ResolveContext(c, ""); err == nil {
		t.Errorf("expected error for empty kubeconfig")
	}
}

func TestContextNames_Sorted(t *testing.T) {
	c := cfg("c", "a", "b")
	got := ContextNames(c)
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ContextNames[%d] = %q want %q", i, got[i], want[i])
		}
	}
}
