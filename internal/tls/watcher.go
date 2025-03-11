package tls

import (
	"context"
	"log/slog"

	"github.com/fsnotify/fsnotify"
)

// CertReloader calls reloadFn whenever certFile or keyFile change on disk.
// Runs until ctx is cancelled.
type CertReloader struct {
	certFile string
	keyFile  string
	reloadFn func(certFile, keyFile string) error
}

func NewCertReloader(certFile, keyFile string, reloadFn func(string, string) error) *CertReloader {
	return &CertReloader{
		certFile: certFile,
		keyFile:  keyFile,
		reloadFn: reloadFn,
	}
}

func (cr *CertReloader) Watch(ctx context.Context) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	if err := w.Add(cr.certFile); err != nil {
		return err
	}
	if err := w.Add(cr.keyFile); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				slog.Info("cert file changed, hot-reloading", "file", event.Name)
				if err := cr.reloadFn(cr.certFile, cr.keyFile); err != nil {
					slog.Error("cert reload failed", "err", err)
				} else {
					slog.Info("cert reloaded successfully")
				}
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			slog.Error("watcher error", "err", err)
		}
	}
}
