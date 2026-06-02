package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingDefaultReturnsDefaults(t *testing.T) {
	// A path that doesn't exist, passed as the *default* (empty explicit path)
	// can't be simulated directly, so verify the documented behaviour: an empty
	// explicit path with no file at the resolved default location is tolerated by
	// having Load fall back to Default when the file is absent.
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	if err == nil {
		t.Fatal("explicit missing --config should error")
	}
	if cfg != Default() {
		t.Errorf("on error Load should return Default(), got %+v", cfg)
	}
}

func TestLoadDefaultsWhenNoExplicitFile(t *testing.T) {
	// With an empty explicit path and (almost certainly) no config in the test
	// environment, Load returns defaults without error.
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") unexpected error: %v", err)
	}
	if cfg.Theme == "" || cfg.DefaultView == "" {
		t.Errorf("expected populated defaults, got %+v", cfg)
	}
}

func writeTOML(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadParsesAllFields(t *testing.T) {
	p := writeTOML(t, `
default_view = "findings"
groups_collapsed = true
accordion = true
theme = "light"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultView != ViewFindings || !cfg.GroupsCollapsed || !cfg.Accordion || cfg.Theme != "light" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLoadPartialKeepsDefaults(t *testing.T) {
	p := writeTOML(t, `theme = "dark"`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Theme != "dark" {
		t.Errorf("theme = %q, want dark", cfg.Theme)
	}
	if cfg.DefaultView != ViewFloorPlan {
		t.Errorf("default_view = %q, want %q (default)", cfg.DefaultView, ViewFloorPlan)
	}
}

func TestDefaultViewAliases(t *testing.T) {
	cases := map[string]string{
		"floor plan":  ViewFloorPlan,
		"plan":        ViewFloorPlan,
		"FloorPlan":   ViewFloorPlan,
		"table":       ViewNodes,
		"nodes table": ViewNodes,
		"findings":    ViewFindings,
	}
	for in, want := range cases {
		p := writeTOML(t, "default_view = \""+in+"\"")
		cfg, err := Load(p)
		if err != nil {
			t.Fatalf("Load(%q): %v", in, err)
		}
		if cfg.DefaultView != want {
			t.Errorf("default_view %q -> %q, want %q", in, cfg.DefaultView, want)
		}
	}
}

func TestInvalidValuesError(t *testing.T) {
	for _, body := range []string{
		`default_view = "dashboard"`,
		`theme = "solarized"`,
	} {
		p := writeTOML(t, body)
		if _, err := Load(p); err == nil {
			t.Errorf("expected error for %q", body)
		}
	}
}
