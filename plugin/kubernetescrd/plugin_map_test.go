package kubernetescrd

import (
	"sync"
	"testing"

	"github.com/coredns/coredns/plugin/forward"
)

func TestPluginMap(t *testing.T) {
	pluginInstanceMap := NewPluginInstanceMap()

	zone1ForwardPlugin := forward.New()
	zone2ForwardPlugin := forward.New()

	// Testing concurrency to ensure map is thread-safe
	// i.e should run with `go test -race`
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		pluginInstanceMap.Upsert("default/some-dns-zone", "zone-1.test", zone1ForwardPlugin)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		pluginInstanceMap.Upsert("default/another-dns-zone", "zone-2.test", zone2ForwardPlugin)
		wg.Done()
	}()
	wg.Wait()

	if plugin, exists := pluginInstanceMap.Get("zone-1.test."); exists && plugin != zone1ForwardPlugin {
		t.Fatalf("Expected plugin instance map to get plugin with address: %p but was: %p", zone1ForwardPlugin, plugin)
	}

	if plugin, exists := pluginInstanceMap.Get("zone-2.test"); exists && plugin != zone2ForwardPlugin {
		t.Fatalf("Expected plugin instance map to get plugin with address: %p but was: %p", zone2ForwardPlugin, plugin)
	}

	if _, exists := pluginInstanceMap.Get("non-existant-zone.test"); exists {
		t.Fatal("Expected plugin instance map to not return a plugin")
	}

	// update record with the same key

	pluginInstanceMap.Upsert("default/some-dns-zone", "new-zone-1.test", zone1ForwardPlugin)

	if plugin, exists := pluginInstanceMap.Get("new-zone-1.test"); exists && plugin != zone1ForwardPlugin {
		t.Fatalf("Expected plugin instance map to get plugin with address: %p but was: %p", zone1ForwardPlugin, plugin)

	}
	if _, exists := pluginInstanceMap.Get("zone-1.test"); exists {
		t.Fatalf("Expected plugin instance map to not get plugin with zone: %s", "zone-1.test")
	}

	// delete record by key

	pluginInstanceMap.Delete("default/some-dns-zone")

	if _, exists := pluginInstanceMap.Get("new-zone-1.test"); exists {
		t.Fatalf("Expected plugin instance map to not get plugin with zone: %s", "new-zone-1.test")
	}
}
