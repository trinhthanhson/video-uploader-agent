package watcher

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type Event struct {
	Path string
	Op   string
}

type Watcher struct {
	fs *fsnotify.Watcher
}

func New() (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{fs: w}, nil
}

func (w *Watcher) Close() error {
	return w.fs.Close()
}

func (w *Watcher) AddRecursive(root string) error {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			_ = w.fs.Add(path)
		}
		return nil
	})
	return err
}

func (w *Watcher) Events() <-chan fsnotify.Event {
	return w.fs.Events
}

func (w *Watcher) Errors() <-chan error {
	return w.fs.Errors
}
