// Package k8s wraps the kubernetes client-go bits we need: kubeconfig
// resolution, a typed clientset, and snapshot fetchers used by the TUI.
package k8s

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// DefaultKubeconfigPaths returns the resolution order client-go uses by
// default: $KUBECONFIG (colon-separated), then ~/.kube/config.
func DefaultKubeconfigPaths() []string {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return filepath.SplitList(env)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{filepath.Join(home, ".kube", "config")}
}

// LoadKubeconfig reads kubeconfig from the given paths (or the defaults if
// empty) and returns the merged api.Config.
func LoadKubeconfig(paths []string) (*clientcmdapi.Config, error) {
	if len(paths) == 0 {
		paths = DefaultKubeconfigPaths()
	}
	if len(paths) == 0 {
		return nil, errors.New("no kubeconfig path available (set --kubeconfig or KUBECONFIG)")
	}
	rules := &clientcmd.ClientConfigLoadingRules{Precedence: paths}
	cfg, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	return cfg, nil
}

// ContextNames returns the alphabetically-sorted list of context names in cfg.
func ContextNames(cfg *clientcmdapi.Config) []string {
	out := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Resolution captures the outcome of context selection.
type Resolution struct {
	Context     string // selected context name; empty if the caller must pick
	NeedsPicker bool   // true when no context could be auto-selected
}

// ResolveContext picks the context to use given the flag value and config.
//
//   - explicit non-empty → use it; error if unknown.
//   - empty and CurrentContext set → use CurrentContext.
//   - empty and no CurrentContext → NeedsPicker=true.
func ResolveContext(cfg *clientcmdapi.Config, explicit string) (Resolution, error) {
	if explicit != "" {
		if _, ok := cfg.Contexts[explicit]; !ok {
			return Resolution{}, fmt.Errorf("context %q not found in kubeconfig", explicit)
		}
		return Resolution{Context: explicit}, nil
	}
	if cfg.CurrentContext != "" {
		if _, ok := cfg.Contexts[cfg.CurrentContext]; !ok {
			return Resolution{}, fmt.Errorf("current-context %q is missing from contexts", cfg.CurrentContext)
		}
		return Resolution{Context: cfg.CurrentContext}, nil
	}
	if len(cfg.Contexts) == 0 {
		return Resolution{}, errors.New("kubeconfig contains no contexts")
	}
	return Resolution{NeedsPicker: true}, nil
}
