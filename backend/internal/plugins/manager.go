package plugins

import (
	"log"
	"os"
	"strings"
)

type Manager struct {
	registry map[string]Plugin
	enabled  map[string]bool
}

var globalManager *Manager

// InitManager creates the singleton plugin manager
func InitManager() *Manager {
	globalManager = &Manager{
		registry: make(map[string]Plugin),
		enabled:  make(map[string]bool),
	}
	return globalManager
}

// Register adds a plugin to the manager (should be called in init() functions of plugins)
func Register(p Plugin) {
	if globalManager == nil {
		InitManager()
	}
	globalManager.registry[p.ID()] = p
}

// Start checks environment variables and initializes enabled plugins
func (m *Manager) Start() {
	for id, p := range m.registry {
		envKey := "SUPERNOVA_PLUGIN_" + strings.ToUpper(id)
		if strings.ToLower(os.Getenv(envKey)) == "true" {
			log.Printf("Starting plugin: %s (%s)", p.Name(), id)
			err := p.Init()
			if err != nil {
				log.Printf("Failed to initialize plugin %s: %v", id, err)
				m.enabled[id] = false
			} else {
				m.enabled[id] = true
			}
		} else {
			m.enabled[id] = false
		}
	}
}

// GetManifest returns a list of all compiled plugins and their enabled status
func (m *Manager) GetManifest() []PluginInfo {
	var manifest []PluginInfo
	for id, p := range m.registry {
		manifest = append(manifest, PluginInfo{
			ID:          id,
			Name:        p.Name(),
			Description: p.Description(),
			Enabled:     m.enabled[id],
		})
	}
	return manifest
}
