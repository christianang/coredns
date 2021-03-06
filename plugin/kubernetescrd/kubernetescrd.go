package kubernetescrd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/forward"
	"github.com/coredns/coredns/plugin/kubernetescrd/apis/coredns/v1alpha1"
	"github.com/miekg/dns"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type KubernetesCRD struct {
	Next    plugin.Handler
	Forward *forward.Forward

	ClientConfig  clientcmd.ClientConfig
	APIServerList []string
	APICertAuth   string
	APIClientCert string
	APIClientKey  string
	APIConn       *crdController
}

func (k KubernetesCRD) Name() string { return "kubernetescrd" }

func (k KubernetesCRD) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	question := strings.ToLower(r.Question[0].Name)

	var (
		offset int
		end    bool
	)

	for {
		if forward, ok := k.APIConn.Forwards[question[offset:]]; ok {
			fmt.Printf("kubernetescrd ServeDNS: found match %s\n", question[offset:])
			return forward.ServeDNS(ctx, w, r)
		}

		offset, end = dns.NextLabel(question, offset)
		if end {
			break
		}
	}

	fmt.Println("kubernetescrd ServeDNS: no matching forward plugins")
	return plugin.NextOrFailure(k.Name(), k.Next, ctx, w, r)
}

func (k *KubernetesCRD) InitKubeCache(ctx context.Context) error {
	config, err := k.getClientConfig()
	if err != nil {
		return err
	}

	kubeClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetescrd controller: %q", err)
	}

	k.APIConn = newKubernetesCRDController(ctx, kubeClient)

	return nil
}

func newKubernetesCRDController(ctx context.Context, kubeClient dynamic.Interface) *crdController {
	crdCtrl := &crdController{
		client:   kubeClient,
		stopCh:   make(chan struct{}),
		Forwards: make(map[string]*forward.Forward),
	}

	crdCtrl.crdLister, crdCtrl.crdController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(opts meta.ListOptions) (runtime.Object, error) {
				return crdCtrl.client.Resource(v1alpha1.GroupVersionResource).List(ctx, opts)
			},
			WatchFunc: func(opts meta.ListOptions) (watch.Interface, error) {
				return crdCtrl.client.Resource(v1alpha1.GroupVersionResource).Watch(ctx, opts)
			},
		},
		&unstructured.Unstructured{},
		time.Second*30,
		cache.ResourceEventHandlerFuncs{
			// Implement other event handler funcs
			AddFunc: crdCtrl.Add,
		},
	)

	return crdCtrl
}

// Slimmed down version from the Kubernetes plugin. Actual implementation will
// probably do most of the same from the Kubernetes plugin in terms of where a
// client config can be loaded from.
func (k *KubernetesCRD) getClientConfig() (*rest.Config, error) {
	if k.ClientConfig != nil {
		return k.ClientConfig.ClientConfig()
	}

	return nil, errors.New("must supply kubeconfig, everything else is unsupported for now")
}
