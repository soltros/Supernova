package plugins

import (
	"net/http"
	"github.com/soltros/Supernova/internal/database"
)

type PluginConfig struct {
	Repo *database.Repository
}

// Plugin defines the standard interface for all Supernova plugins
type Plugin interface {
	ID() string
	Name() string
	Description() string
	Init(config PluginConfig) error
	SetupRoutes(mux *http.ServeMux)
}

// PluginInfo is used to serialize plugin metadata for the frontend
type PluginInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}
