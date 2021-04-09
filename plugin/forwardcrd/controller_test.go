package forwardcrd

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/forward"
	corednsv1alpha1 "github.com/coredns/coredns/plugin/forwardcrd/apis/coredns/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic/fake"
)

func TestCreateDNSZone(t *testing.T) {
	controller, client, testPluginInstancer, pluginInstanceMap := setupControllerTestcase(t, "")
	dnsZone := &corednsv1alpha1.DNSZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dns-zone",
			Namespace: "default",
		},
		Spec: corednsv1alpha1.DNSZoneSpec{
			ZoneName:  "crd.test",
			ForwardTo: "127.0.0.2",
		},
	}

	_, err := client.Resource(corednsv1alpha1.GroupVersion.WithResource("dnszones")).
		Namespace("default").
		Create(context.Background(), mustDNSZoneToUnstructured(dnsZone), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Expected not to error: %s", err)
	}

	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		return testPluginInstancer.NewWithConfigCallCount() == 1, nil
	})
	if err != nil {
		t.Fatalf("Expected plugin instance to have been called: %s", err)
	}

	handler := testPluginInstancer.NewWithConfigArgsForCall(0)
	if handler.ReceivedConfig.From != "crd.test" {
		t.Fatalf("Expected plugin to be created for zone: %s but was: %s", "crd.test", handler.ReceivedConfig.From)
	}

	if len(handler.ReceivedConfig.To) != 1 {
		t.Fatalf("Expected plugin to contain exactly 1 server to forward to but contains: %#v", handler.ReceivedConfig.To)
	}

	if handler.ReceivedConfig.To[0] != "127.0.0.2" {
		t.Fatalf("Expected plugin to be created to forward to: %s but was: %s", "127.0.0.2", handler.ReceivedConfig.To[0])
	}

	pluginHandler, ok := pluginInstanceMap.Get("crd.test")
	if !ok {
		t.Fatal("Expected plugin lookup to succeed")
	}

	if pluginHandler != handler {
		t.Fatalf("Exepcted plugin lookup to match what the instancer provided: %#v but was %#v", handler, pluginHandler)
	}

	if testPluginInstancer.testPluginHandlers[0].OnStartupCallCount() != 1 {
		t.Fatalf("Expected plugin OnStartup to have been called once, but got: %d", testPluginInstancer.testPluginHandlers[0].OnStartupCallCount())
	}

	if err := controller.Stop(); err != nil {
		t.Fatalf("Expected no error: %s", err)
	}

	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		return testPluginInstancer.testPluginHandlers[0].OnShutdownCallCount() == 1, nil
	})
	if err != nil {
		t.Fatalf("Expected plugin OnShutdown to have been called once, but got: %d", testPluginInstancer.testPluginHandlers[0].OnShutdownCallCount())
	}
}

func TestUpdateDNSZone(t *testing.T) {
	controller, client, testPluginInstancer, pluginInstanceMap := setupControllerTestcase(t, "")
	dnsZone := &corednsv1alpha1.DNSZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dns-zone",
			Namespace: "default",
		},
		Spec: corednsv1alpha1.DNSZoneSpec{
			ZoneName:  "crd.test",
			ForwardTo: "127.0.0.2",
		},
	}

	unstructuredDNSZone, err := client.Resource(corednsv1alpha1.GroupVersion.WithResource("dnszones")).
		Namespace("default").
		Create(context.Background(), mustDNSZoneToUnstructured(dnsZone), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Expected not to error: %s", err)
	}

	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		return testPluginInstancer.NewWithConfigCallCount() == 1, nil
	})
	if err != nil {
		t.Fatalf("Expected plugin instance to have been called: %s", err)
	}

	dnsZone = mustUnstructuredToDNSZone(unstructuredDNSZone)
	dnsZone.Spec.ZoneName = "other.test"
	dnsZone.Spec.ForwardTo = "127.0.0.3"

	_, err = client.Resource(corednsv1alpha1.GroupVersion.WithResource("dnszones")).
		Namespace("default").
		Update(context.Background(), mustDNSZoneToUnstructured(dnsZone), metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Expected not to error: %s", err)
	}

	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		return testPluginInstancer.NewWithConfigCallCount() == 2, nil
	})
	if err != nil {
		t.Fatalf("Expected plugin instance to have been called: %s", err)
	}

	handler := testPluginInstancer.NewWithConfigArgsForCall(1)
	if handler.ReceivedConfig.From != "other.test" {
		t.Fatalf("Expected plugin to be created for zone: %s but was: %s", "other.test", handler.ReceivedConfig.From)
	}

	if len(handler.ReceivedConfig.To) != 1 {
		t.Fatalf("Expected plugin to contain exactly 1 server to forward to but contains: %#v", handler.ReceivedConfig.To)
	}

	if handler.ReceivedConfig.To[0] != "127.0.0.3" {
		t.Fatalf("Expected plugin to be created to forward to: %s but was: %s", "127.0.0.3", handler.ReceivedConfig.To[0])
	}

	pluginHandler, ok := pluginInstanceMap.Get("other.test")
	if !ok {
		t.Fatal("Expected plugin lookup to succeed")
	}

	if pluginHandler != handler {
		t.Fatalf("Exepcted plugin lookup to match what the instancer provided: %#v but was %#v", handler, pluginHandler)
	}

	_, ok = pluginInstanceMap.Get("crd.test")
	if ok {
		t.Fatal("Expected lookup for crd.test to fail")
	}

	if testPluginInstancer.testPluginHandlers[0].OnShutdownCallCount() != 1 {
		t.Fatalf("Expected plugin OnShutdown to have been called once, but got: %d", testPluginInstancer.testPluginHandlers[0].OnShutdownCallCount())
	}

	if err := controller.Stop(); err != nil {
		t.Fatalf("Expected no error: %s", err)
	}
}

func TestDeleteDNSZone(t *testing.T) {
	controller, client, testPluginInstancer, pluginInstanceMap := setupControllerTestcase(t, "")
	dnsZone := &corednsv1alpha1.DNSZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dns-zone",
			Namespace: "default",
		},
		Spec: corednsv1alpha1.DNSZoneSpec{
			ZoneName:  "crd.test",
			ForwardTo: "127.0.0.2",
		},
	}

	_, err := client.Resource(corednsv1alpha1.GroupVersion.WithResource("dnszones")).
		Namespace("default").
		Create(context.Background(), mustDNSZoneToUnstructured(dnsZone), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Expected not to error: %s", err)
	}

	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		return testPluginInstancer.NewWithConfigCallCount() == 1, nil
	})
	if err != nil {
		t.Fatalf("Expected plugin instance to have been called: %s", err)
	}

	err = client.Resource(corednsv1alpha1.GroupVersion.WithResource("dnszones")).
		Namespace("default").
		Delete(context.Background(), "test-dns-zone", metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Expected not to error: %s", err)
	}

	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		_, ok := pluginInstanceMap.Get("crd.test")
		return !ok, nil
	})
	if err != nil {
		t.Fatalf("Expected lookup for crd.test to fail: %s", err)
	}

	if testPluginInstancer.testPluginHandlers[0].OnShutdownCallCount() != 1 {
		t.Fatalf("Expected plugin OnShutdown to have been called once, but got: %d", testPluginInstancer.testPluginHandlers[0].OnShutdownCallCount())
	}

	if err := controller.Stop(); err != nil {
		t.Fatalf("Expected no error: %s", err)
	}
}

func TestDNSZoneLimitNamespace(t *testing.T) {
	controller, client, testPluginInstancer, pluginInstanceMap := setupControllerTestcase(t, "kube-system")
	dnsZone := &corednsv1alpha1.DNSZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dns-zone",
			Namespace: "default",
		},
		Spec: corednsv1alpha1.DNSZoneSpec{
			ZoneName:  "crd.test",
			ForwardTo: "127.0.0.2",
		},
	}

	_, err := client.Resource(corednsv1alpha1.GroupVersion.WithResource("dnszones")).
		Namespace("default").
		Create(context.Background(), mustDNSZoneToUnstructured(dnsZone), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Expected not to error: %s", err)
	}

	kubeSystemDNSZone := &corednsv1alpha1.DNSZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-dns-zone",
			Namespace: "kube-system",
		},
		Spec: corednsv1alpha1.DNSZoneSpec{
			ZoneName:  "system.test",
			ForwardTo: "127.0.0.3",
		},
	}

	_, err = client.Resource(corednsv1alpha1.GroupVersion.WithResource("dnszones")).
		Namespace("kube-system").
		Create(context.Background(), mustDNSZoneToUnstructured(kubeSystemDNSZone), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Expected not to error: %s", err)
	}

	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		return testPluginInstancer.NewWithConfigCallCount() == 1, nil
	})
	if err != nil {
		t.Fatalf("Expected plugin instance to have been called exactly once: %s, plugin instance call count: %d", err, testPluginInstancer.NewWithConfigCallCount())
	}

	handler := testPluginInstancer.NewWithConfigArgsForCall(0)
	if handler.ReceivedConfig.From != "system.test" {
		t.Fatalf("Expected plugin to be created for zone: %s but was: %s", "system.test", handler.ReceivedConfig.From)
	}

	_, ok := pluginInstanceMap.Get("system.test")
	if !ok {
		t.Fatal("Expected plugin lookup to succeed")
	}

	_, ok = pluginInstanceMap.Get("crd.test")
	if ok {
		t.Fatal("Expected plugin lookup to fail")
	}

	if err := controller.Stop(); err != nil {
		t.Fatalf("Expected no error: %s", err)
	}
}

func setupControllerTestcase(t *testing.T, namespace string) (dnsZoneCRDController, *fake.FakeDynamicClient, *TestPluginInstancer, *PluginInstanceMap) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(corednsv1alpha1.GroupVersion, &corednsv1alpha1.DNSZone{})
	customListKinds := map[schema.GroupVersionResource]string{
		corednsv1alpha1.GroupVersion.WithResource("dnszones"): "DNSZoneList",
	}
	client := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, customListKinds)
	pluginMap := NewPluginInstanceMap()
	testPluginInstancer := &TestPluginInstancer{}
	controller := newDNSZoneCRDController(context.Background(), client, scheme, namespace, pluginMap, func(cfg forward.ForwardConfig) (lifecyclePluginHandler, error) {
		return testPluginInstancer.NewWithConfig(cfg)
	})

	go controller.Run(1)

	err := wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		return controller.HasSynced(), nil
	})
	if err != nil {
		t.Fatalf("Expected controller to have synced: %s", err)
	}

	return controller, client, testPluginInstancer, pluginMap
}

func mustDNSZoneToUnstructured(dnsZone *corednsv1alpha1.DNSZone) *unstructured.Unstructured {
	dnsZone.TypeMeta = metav1.TypeMeta{
		Kind:       "DNSZone",
		APIVersion: "coredns.io/v1alpha1",
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dnsZone)
	if err != nil {
		panic(fmt.Sprintf("coding error: unable to convert to unstructured: %s", err))
	}
	return &unstructured.Unstructured{
		Object: obj,
	}
}

func mustUnstructuredToDNSZone(obj *unstructured.Unstructured) *corednsv1alpha1.DNSZone {
	dnsZone := &corednsv1alpha1.DNSZone{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, dnsZone)
	if err != nil {
		panic(fmt.Sprintf("coding error: unable to convert from unstructured: %s", err))
	}
	return dnsZone
}
