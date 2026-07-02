package plugins

import (
	"log"
	"net/http"
	"os"
	"strings"
)

type Manager struct {
	registry map[string]Plugin
	enabled  map[string]bool
}

var globalManager *Manager

// GetManager returns the singleton plugin manager, initializing it if necessary
func GetManager() *Manager {
	if globalManager == nil {
		globalManager = &Manager{
			registry: make(map[string]Plugin),
			enabled:  make(map[string]bool),
		}
	}
	return globalManager
}

// InitManager is kept for backwards compatibility
func InitManager() *Manager {
	return GetManager()
}

// Register adds a plugin to the manager (should be called in init() functions of plugins)
func Register(p Plugin) {
	GetManager().registry[p.ID()] = p
}

// Start checks environment variables and initializes enabled plugins
func (m *Manager) Start(config PluginConfig) {
	for id, p := range m.registry {
		envKey := "SUPERNOVA_PLUGIN_" + strings.ToUpper(id)
		if strings.ToLower(os.Getenv(envKey)) != "false" {
			log.Printf("Starting plugin: %s (%s)", p.Name(), id)
			err := p.Init(config)
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

// SetupPluginRoutes allows all enabled plugins to register their API endpoints
func (m *Manager) SetupPluginRoutes(mux *http.ServeMux) {
	for id, p := range m.registry {
		if m.enabled[id] {
			p.SetupRoutes(mux)
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
