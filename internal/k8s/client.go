package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// BuildClient returns a typed kubernetes client for the named context.
func BuildClient(cfg *clientcmdapi.Config, context string) (*kubernetes.Clientset, error) {
	overrides := &clientcmd.ConfigOverrides{CurrentContext: context}
	cc := clientcmd.NewDefaultClientConfig(*cfg, overrides)
	rest, err := cc.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build rest config for context %q: %w", context, err)
	}
	rest.UserAgent = "podscape"
	cs, err := kubernetes.NewForConfig(rest)
	if err != nil {
		return nil, fmt.Errorf("build clientset: %w", err)
	}
	return cs, nil
}
