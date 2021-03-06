package kubernetescrd

import (
	"fmt"
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/forward"
	"github.com/coredns/coredns/plugin/kubernetescrd/apis/coredns/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
)

type crdController struct {
	client dynamic.Interface

	crdController cache.Controller
	crdLister     cache.Store

	stopLock sync.Mutex
	shutdown bool
	stopCh   chan struct{}

	Forwards map[string]*forward.Forward
}

func (c *crdController) Run() {
	go c.crdController.Run(c.stopCh)
	<-c.stopCh
}

func (c *crdController) Stop() error {
	c.stopLock.Lock()
	defer c.stopLock.Unlock()

	// Only try draining the workqueue if we haven't already.
	if !c.shutdown {
		close(c.stopCh)
		c.shutdown = true

		return nil
	}

	return fmt.Errorf("shutdown already in progress")
}

func (c *crdController) HasSynced() bool {
	return c.crdController.HasSynced()
}

func (c *crdController) Add(obj interface{}) {
	unstructured := obj.(*unstructured.Unstructured).UnstructuredContent()
	var dnsZone v1alpha1.DNSZone
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &dnsZone)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Added DNSZone zoneName: %s forwardTo: %s\n", dnsZone.Spec.ZoneName, dnsZone.Spec.ForwardTo)
	f, err := forward.Instance(forward.ForwardOptions{
		From: dnsZone.Spec.ZoneName,
		To:   []string{dnsZone.Spec.ForwardTo},
	})
	if err != nil {
		panic(err)
	}

	err = f.OnStartup()
	if err != nil {
		panic(err)
	}

	c.Forwards[plugin.Name(dnsZone.Spec.ZoneName).Normalize()] = f
}
