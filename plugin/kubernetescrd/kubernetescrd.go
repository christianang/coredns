package kubernetescrd

import (
	"context"

	"github.com/coredns/coredns/plugin"

	"github.com/miekg/dns"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesCRD represents a plugin instance that can watch DNSZone CRDs
// within a Kubernetes clusters to dynamically configure stub-domains to proxy
// requests to an upstream resolver.
type KubernetesCRD struct {
	Zones             []string
	APIServerEndpoint string
	APIClientCert     string
	APIClientKey      string
	APICertAuth       string
	Namespace         string
	ClientConfig      clientcmd.ClientConfig

	Next plugin.Handler
}

// Name implements plugin.Handler.
func (k *KubernetesCRD) Name() string { return "kubernetescrd" }

// ServeDNS implements plugin.Handler.
func (k *KubernetesCRD) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return 0, nil
}
