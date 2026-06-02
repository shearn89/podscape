package styles

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestResolveNamedThemes(t *testing.T) {
	if got := Resolve("dark"); got.Name != "dark" {
		t.Errorf("Resolve(dark).Name = %q", got.Name)
	}
	if got := Resolve("light"); got.Name != "light" {
		t.Errorf("Resolve(light).Name = %q", got.Name)
	}
	// auto resolves to one of the registered presets.
	if got := Resolve("auto"); got.Name != "dark" && got.Name != "light" {
		t.Errorf("Resolve(auto).Name = %q, want dark or light", got.Name)
	}
	// Unknown falls back to dark rather than an empty theme.
	if got := Resolve("nope"); got.Name != "dark" {
		t.Errorf("Resolve(unknown).Name = %q, want dark fallback", got.Name)
	}
}

func TestApplyUpdatesActivePalette(t *testing.T) {
	Apply(Resolve("light"))
	if ColorBG != lightTheme.BG {
		t.Errorf("ColorBG = %v after Apply(light), want %v", ColorBG, lightTheme.BG)
	}
	Apply(Resolve("dark"))
	if ColorMuted != lipgloss.Color("#8b94a7") {
		t.Errorf("dark ColorMuted = %v, want the lifted-contrast #8b94a7", ColorMuted)
	}
	// Composed styles must track the active palette, not stale init values.
	if got := Help.GetForeground(); got != ColorMuted {
		t.Errorf("Help foreground = %v, want active ColorMuted %v", got, ColorMuted)
	}
}

func TestIsValidTheme(t *testing.T) {
	for _, ok := range []string{"auto", "dark", "light"} {
		if !IsValidTheme(ok) {
			t.Errorf("IsValidTheme(%q) = false, want true", ok)
		}
	}
	if IsValidTheme("solarized") {
		t.Error("IsValidTheme(solarized) = true, want false")
	}
}

func TestRegisterTheme(t *testing.T) {
	RegisterTheme(Theme{Name: "midnight", BG: lipgloss.Color("#000000")})
	if !IsValidTheme("midnight") {
		t.Error("registered theme not reported valid")
	}
	if Resolve("midnight").Name != "midnight" {
		t.Error("registered theme not resolvable")
	}
	delete(registry, "midnight") // keep the registry clean for other tests
}
