// Package config loads podscape's optional TOML settings file. It exposes a few
// personal defaults — which tab opens first, whether node groups start
// collapsed, floor-plan accordion mode, and the colour theme — that would
// otherwise have to be set by flag on every run.
//
// The file lives at $XDG_CONFIG_HOME/podscape/config.toml (via
// os.UserConfigDir), or wherever --config points. A missing default file is not
// an error: Load returns built-in defaults so the app runs out of the box.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/shearn89/podscape/internal/tui/styles"
)

// View names accepted for default_view.
const (
	ViewFloorPlan = "floorplan"
	ViewNodes     = "nodes"
	ViewFindings  = "findings"
)

// Config is the user's settings. Fields map to TOML keys; unset keys keep the
// values from Default.
type Config struct {
	// DefaultView is the tab shown at launch: "floorplan", "nodes", or
	// "findings".
	DefaultView string `toml:"default_view"`
	// GroupsCollapsed starts every node group collapsed on the floor plan.
	GroupsCollapsed bool `toml:"groups_collapsed"`
	// Accordion keeps only the focused node group expanded on the floor plan.
	Accordion bool `toml:"accordion"`
	// Theme selects the colour scheme: "auto" (detect light/dark), "dark", or
	// "light".
	Theme string `toml:"theme"`
}

// Default returns the built-in settings used when no config file is present.
func Default() Config {
	return Config{
		DefaultView:     ViewFloorPlan,
		GroupsCollapsed: false,
		Accordion:       false,
		Theme:           "auto",
	}
}

// DefaultPath is the standard config location: <user-config-dir>/podscape/config.toml.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "podscape", "config.toml"), nil
}

// Load reads the config file and returns the resulting settings. When
// explicitPath is empty the default path is used and a missing file yields
// Default() with no error. When explicitPath is set, a missing file is an error
// (so a typo'd --config isn't silently ignored).
func Load(explicitPath string) (Config, error) {
	cfg := Default()

	path := explicitPath
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return cfg, err
		}
		path = p
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if os.IsNotExist(err) && explicitPath == "" {
			return Default(), nil
		}
		return Default(), fmt.Errorf("config %s: %w", path, err)
	}

	if err := cfg.normalize(); err != nil {
		return Default(), fmt.Errorf("config %s: %w", path, err)
	}
	return cfg, nil
}

// normalize lower-cases and validates the string fields, mapping a few friendly
// aliases for default_view onto the canonical names.
func (c *Config) normalize() error {
	switch v := strings.ToLower(strings.TrimSpace(c.DefaultView)); v {
	case "", ViewFloorPlan, "floor plan", "plan", "floor-plan":
		c.DefaultView = ViewFloorPlan
	case ViewNodes, "table", "nodes table", "nodestable":
		c.DefaultView = ViewNodes
	case ViewFindings:
		c.DefaultView = ViewFindings
	default:
		return fmt.Errorf("invalid default_view %q (want one of: floorplan, nodes, findings)", c.DefaultView)
	}

	c.Theme = strings.ToLower(strings.TrimSpace(c.Theme))
	if c.Theme == "" {
		c.Theme = "auto"
	}
	if !styles.IsValidTheme(c.Theme) {
		return fmt.Errorf("invalid theme %q (want one of: %s)", c.Theme, strings.Join(styles.ThemeNames(), ", "))
	}
	return nil
}
