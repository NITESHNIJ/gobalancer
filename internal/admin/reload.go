package admin

import (
	"log/slog"
	"sync/atomic"
	"unsafe"

	"github.com/ninijhawan/gobalancer/config"
)

// ConfigReloader holds a pointer to the live config that can be atomically
// swapped when a hot-reload is triggered (e.g. via POST /admin/reload or SIGHUP).
type ConfigReloader struct {
	ptr unsafe.Pointer // *config.Config
}

func NewConfigReloader(initial *config.Config) *ConfigReloader {
	r := &ConfigReloader{}
	r.Store(initial)
	return r
}

// Load returns the current config. Safe for concurrent reads.
func (r *ConfigReloader) Load() *config.Config {
	return (*config.Config)(atomic.LoadPointer(&r.ptr))
}

// Store atomically swaps in a new config.
func (r *ConfigReloader) Store(cfg *config.Config) {
	atomic.StorePointer(&r.ptr, unsafe.Pointer(cfg))
}

// Reload loads a new config from path and swaps it in.
// The old config stays alive as long as any goroutine holds a reference.
func (r *ConfigReloader) Reload(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	r.Store(cfg)
	slog.Info("config reloaded", "path", path)
	return nil
}
