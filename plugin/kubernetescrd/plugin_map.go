package kubernetescrd

import (
	"sync"

	"github.com/coredns/coredns/plugin"
)

// PluginInstanceMap represents a map of zones to coredns plugin instances that
// is thread-safe. It enables the kubernetescrd plugin to save the state of
// which plugin instances should be delegated to for a given zone.
type PluginInstanceMap struct {
	mutex          *sync.RWMutex
	zonesToPlugins map[string]plugin.Handler
	keyToZones     map[string]string
}

// NewPluginInstanceMap returns a new instance of PluginInstanceMap.
func NewPluginInstanceMap() *PluginInstanceMap {
	return &PluginInstanceMap{
		mutex:          &sync.RWMutex{},
		zonesToPlugins: make(map[string]plugin.Handler),
		keyToZones:     make(map[string]string),
	}
}

// Upsert adds or updates the map with a zone to plugin handler mapping. If
// the same key is provided it will overwrite the old zone for that key with
// the new zone.
func (p *PluginInstanceMap) Upsert(key, zone string, handler plugin.Handler) {
	p.mutex.Lock()
	normalizedZone := plugin.Host(zone).Normalize()
	if oldZone, ok := p.keyToZones[key]; ok {
		delete(p.zonesToPlugins, oldZone)
	}

	p.keyToZones[key] = normalizedZone
	p.zonesToPlugins[normalizedZone] = handler
	p.mutex.Unlock()
}

// Get gets the plugin handler provided a zone name. It will return true if the
// plugin handler exists and false if it does not exist.
func (pm *PluginInstanceMap) Get(zone string) (plugin.Handler, bool) {
	pm.mutex.RLock()
	normalizedZone := plugin.Host(zone).Normalize()
	handler, ok := pm.zonesToPlugins[normalizedZone]
	pm.mutex.RUnlock()
	return handler, ok
}

// Delete deletes the zone and plugin handler from the map.
func (p *PluginInstanceMap) Delete(key string) {
	p.mutex.RLock()
	zone := p.keyToZones[key]
	delete(p.zonesToPlugins, zone)
	delete(p.keyToZones, key)
	p.mutex.RUnlock()
}
