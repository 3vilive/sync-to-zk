package main

import (
	"gopkg.in/fsnotify.v1"
)

type FileSystemWatcher struct {
	Watcher *fsnotify.Watcher
	OnEvent func(watcher *fsnotify.Watcher, event fsnotify.Event)
	OnError func(err error)
}

func (w *FileSystemWatcher) Run(watchDirs []string) error {
	for _, dir := range watchDirs {
		if err := w.Watcher.Add(dir); err != nil {
			return err
		}
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-w.Watcher.Events:
				if !ok {
					done <- true
					return
				}

				if w.OnEvent != nil {
					w.OnEvent(w.Watcher, event)
				}

			case err, ok := <-w.Watcher.Errors:
				if !ok {
					done <- true
					return
				}

				if w.OnError != nil {
					w.OnError(err)
				}
			}
		}
	}()

	<-done
	return nil
}
