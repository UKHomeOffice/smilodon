package controller

import (
	"github.com/UKHomeOffice/smilodon/pkg/backend"
)

// Controller TODO
type Controller interface {
	Run()
}

// Manager TODO
type Manager struct {
	Config
}

// Config is a backend configuration type
type Config struct {
	Backend backend.Backend
}

// New TODO
func New(cfg Config) *Manager {
	return &Manager{
		Config: cfg,
	}
}

// Run TODO
func (Manager) Run() {
	// TODO
}
