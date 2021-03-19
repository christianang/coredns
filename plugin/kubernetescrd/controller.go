package kubernetescrd

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/forward"
	"github.com/coredns/coredns/plugin/kubernetescrd/apis/coredns/v1alpha1"
	corednsv1alpha1 "github.com/coredns/coredns/plugin/kubernetescrd/apis/coredns/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const defaultResyncPeriod = 0

type dnsZoneCRDController interface {
	Run(threads int)
	HasSynced() bool
	Stop() error
}

type dnsZoneCRDControl struct {
	client            dynamic.Interface
	scheme            *runtime.Scheme
	dnsZoneController cache.Controller
	dnsZoneLister     cache.Store
	workqueue         workqueue.RateLimitingInterface
	pluginMap         *PluginInstanceMap
	instancer         pluginInstancer

	// stopLock is used to enforce only a single call to Stop is active.
	// Needed because we allow stopping through an http endpoint and
	// allowing concurrent stoppers leads to stack traces.
	stopLock sync.Mutex
	shutdown bool
	stopCh   chan struct{}
}

type pluginInstancer func(forward.ForwardConfig) (plugin.Handler, error)

func newDNSZoneCRDController(ctx context.Context, client dynamic.Interface, scheme *runtime.Scheme, pluginMap *PluginInstanceMap, instancer pluginInstancer) dnsZoneCRDController {
	controller := dnsZoneCRDControl{
		client:    client,
		scheme:    scheme,
		stopCh:    make(chan struct{}),
		pluginMap: pluginMap,
		instancer: instancer,
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DNSZoneCRD"),
	}

	controller.dnsZoneLister, controller.dnsZoneController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return controller.client.Resource(v1alpha1.GroupVersion.WithResource("dnszones")).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return controller.client.Resource(v1alpha1.GroupVersion.WithResource("dnszones")).Watch(ctx, options)
			},
		},
		&unstructured.Unstructured{},
		defaultResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					controller.workqueue.Add(key)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(newObj)
				if err == nil {
					controller.workqueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					controller.workqueue.Add(key)
				}
			},
		},
	)

	return &controller
}

// Run starts the controller. Threads is the number of workers that can process
// work on the workqueue in parallel.
func (d *dnsZoneCRDControl) Run(threads int) {
	defer utilruntime.HandleCrash()
	defer d.workqueue.ShutDown()

	go d.dnsZoneController.Run(d.stopCh)

	if !cache.WaitForCacheSync(d.stopCh, d.dnsZoneController.HasSynced) {
		utilruntime.HandleError(errors.New("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threads; i++ {
		go wait.Until(d.runWorker, time.Second, d.stopCh)
	}

	<-d.stopCh
}

// HasSynced returns true once the controller has completed an initial resource
// listing.
func (d *dnsZoneCRDControl) HasSynced() bool {
	return d.dnsZoneController.HasSynced()
}

// Stop stops the controller.
func (d *dnsZoneCRDControl) Stop() error {
	d.stopLock.Lock()
	defer d.stopLock.Unlock()

	// Only try draining the workqueue if we haven't already.
	if !d.shutdown {
		close(d.stopCh)
		d.shutdown = true

		return nil
	}

	return fmt.Errorf("shutdown already in progress")
}

func (d *dnsZoneCRDControl) runWorker() {
	for d.processNextItem() {
	}
}

func (d *dnsZoneCRDControl) processNextItem() bool {
	key, quit := d.workqueue.Get()
	if quit {
		return false
	}

	defer d.workqueue.Done(key)

	err := d.sync(key.(string))
	if err != nil {
		log.Errorf("Error syncing DNSZone %v: %v", key, err)
		d.workqueue.AddRateLimited(key)
		return true
	}

	d.workqueue.Forget(key)

	return true
}

func (d *dnsZoneCRDControl) sync(key string) error {
	obj, exists, err := d.dnsZoneLister.GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		d.pluginMap.Delete(key)
	} else {
		dnsZone, err := d.convertToDNSZone(obj.(runtime.Object))
		if err != nil {
			return err
		}
		forwardConfig := forward.ForwardConfig{
			From: dnsZone.Spec.ZoneName,
			To:   []string{dnsZone.Spec.ForwardTo},
		}
		plugin, err := d.instancer(forwardConfig)
		if err != nil {
			return err
		}
		d.pluginMap.Upsert(key, dnsZone.Spec.ZoneName, plugin)
	}

	return nil
}

func (d *dnsZoneCRDControl) convertToDNSZone(obj runtime.Object) (*corednsv1alpha1.DNSZone, error) {
	unstructured, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("object was not Unstructured")
	}

	switch unstructured.GetKind() {
	case "DNSZone":
		dnsZone := &corednsv1alpha1.DNSZone{}
		err := d.scheme.Convert(unstructured, dnsZone, nil)
		return dnsZone, err
	default:
		return nil, fmt.Errorf("unsupported object type: %T", unstructured)
	}
}
