package kubernetescrd

import (
	"context"
	"fmt"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/forward"
	corednsv1alpha1 "github.com/coredns/coredns/plugin/kubernetescrd/apis/coredns/v1alpha1"

	"github.com/miekg/dns"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	APIConn           dnsZoneCRDController
	Next              plugin.Handler

	pluginInstanceMap *PluginInstanceMap
}

// New returns a new KubernetesCRD instance.
func New() *KubernetesCRD {
	return &KubernetesCRD{
		pluginInstanceMap: NewPluginInstanceMap(),
	}
}

// Name implements plugin.Handler.
func (k *KubernetesCRD) Name() string { return "kubernetescrd" }

// ServeDNS implements plugin.Handler.
func (k *KubernetesCRD) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	question := strings.ToLower(r.Question[0].Name)

	var (
		offset int
		end    bool
	)

	for {
		p, ok := k.pluginInstanceMap.Get(question[offset:])
		if ok {
			a, b := p.ServeDNS(ctx, w, r)
			return a, b
		}

		offset, end = dns.NextLabel(question, offset)
		if end {
			break
		}
	}

	return plugin.NextOrFailure(k.Name(), k.Next, ctx, w, r)
}

// InitKubeCache initializes a new Kubernetes cache.
func (k *KubernetesCRD) InitKubeCache(ctx context.Context) error {
	config, err := k.getClientConfig()
	if err != nil {
		return err
	}

	dynamicKubeClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetescrd controller: %q", err)
	}

	scheme := runtime.NewScheme()
	err = corednsv1alpha1.AddToScheme(scheme)
	if err != nil {
		return fmt.Errorf("failed to create kubernetescrd controller: %q", err)
	}

	k.APIConn = newDNSZoneCRDController(ctx, dynamicKubeClient, scheme, k.pluginInstanceMap, func(cfg forward.ForwardConfig) (plugin.Handler, error) {
		return forward.NewWithConfig(cfg)
	})

	return nil
}

func (k *KubernetesCRD) getClientConfig() (*rest.Config, error) {
	if k.ClientConfig != nil {
		return k.ClientConfig.ClientConfig()
	}
	loadingRules := &clientcmd.ClientConfigLoadingRules{}
	overrides := &clientcmd.ConfigOverrides{}
	clusterinfo := clientcmdapi.Cluster{}
	authinfo := clientcmdapi.AuthInfo{}

	// Connect to API from in cluster
	if k.APIServerEndpoint == "" {
		cc, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		cc.ContentType = "application/vnd.kubernetes.protobuf"
		return cc, err
	}

	// Connect to API from out of cluster
	clusterinfo.Server = k.APIServerEndpoint

	if len(k.APICertAuth) > 0 {
		clusterinfo.CertificateAuthority = k.APICertAuth
	}
	if len(k.APIClientCert) > 0 {
		authinfo.ClientCertificate = k.APIClientCert
	}
	if len(k.APIClientKey) > 0 {
		authinfo.ClientKey = k.APIClientKey
	}

	overrides.ClusterInfo = clusterinfo
	overrides.AuthInfo = authinfo
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	cc, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	cc.ContentType = "application/vnd.kubernetes.protobuf"
	return cc, err
}
