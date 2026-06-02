// podscape is a Bubble Tea TUI that draws the cluster as a floor plan of
// node groups, with pods colour-coded inside each node card and DaemonSet
// overhead visualised on every card. Read-only by design.
package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"

	"github.com/shearn89/podscape/internal/config"
	"github.com/shearn89/podscape/internal/k8s"
	"github.com/shearn89/podscape/internal/tui/app"
	"github.com/shearn89/podscape/internal/tui/picker"
	"github.com/shearn89/podscape/internal/tui/styles"
)

// version is injected at build time via `-ldflags="-X main.version=..."`.
// Defaults to "dev" for `go run` / `go build` without ldflags.
var version = "dev"

func main() {
	var (
		ctxFlag     string
		kubeconfig  string
		namespace   string
		configPath  string
		refresh     time.Duration
		showVersion bool
	)
	pflag.StringVar(&ctxFlag, "context", "", "kubeconfig context to use (skips the picker)")
	pflag.StringVar(&kubeconfig, "kubeconfig", "", "path(s) to kubeconfig, colon-separated (default: $KUBECONFIG or ~/.kube/config)")
	pflag.StringVarP(&namespace, "namespace", "n", "", "limit pods to a namespace (default: all)")
	pflag.StringVar(&configPath, "config", "", "path to config file (default: <user-config-dir>/podscape/config.toml)")
	pflag.DurationVar(&refresh, "refresh", 15*time.Second, "snapshot refresh interval")
	pflag.BoolVar(&showVersion, "version", false, "print version and exit")
	pflag.Parse()

	if showVersion {
		fmt.Println("podscape", version)
		return
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		die(err)
	}
	// Apply the theme before any TUI runs so the picker is themed too.
	styles.Apply(styles.Resolve(cfg.Theme))

	var paths []string
	if kubeconfig != "" {
		paths = []string{kubeconfig}
	}
	kubeCfg, err := k8s.LoadKubeconfig(paths)
	if err != nil {
		die(err)
	}

	resolution, err := k8s.ResolveContext(kubeCfg, ctxFlag)
	if err != nil {
		die(err)
	}
	if resolution.NeedsPicker {
		chosen, err := runPicker(k8s.ContextNames(kubeCfg))
		if err != nil {
			die(err)
		}
		if chosen == "" {
			os.Exit(0)
		}
		resolution.Context = chosen
	}

	cs, err := k8s.BuildClient(kubeCfg, resolution.Context)
	if err != nil {
		die(err)
	}

	model := app.New(cs, resolution.Context, namespace, refresh, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		die(err)
	}
}

func runPicker(contexts []string) (string, error) {
	p := tea.NewProgram(picker.New(contexts), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return "", err
	}
	pm, ok := m.(picker.Model)
	if !ok {
		return "", nil
	}
	if pm.Quit() {
		return "", nil
	}
	return pm.Chosen(), nil
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "podscape:", err)
	os.Exit(1)
}
