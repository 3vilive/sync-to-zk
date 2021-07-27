package main

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/go-zookeeper/zk"
	"gopkg.in/fsnotify.v1"
)

type Args struct {
	Servers []string `arg:"required,env:ZK_SERVERS"`
	Dirs    []string `arg:"required,env:SYNC_DIRS"`
}

func main() {
	var args Args
	arg.MustParse(&args)
	log.Printf("args: servers=%s, dirs=%s\n", args.Servers, args.Dirs)

	// connect to zookeeper
	zkConn, _, err := zk.Connect(args.Servers, 60*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	// instantiate zksync
	zkSync := ZkSync{ZkConn: zkConn}

	// init sync
	watchDirs := make([]string, 0, 64)
	for _, dir := range args.Dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			log.Printf("init sync: path=%s\n", path)

			if info.IsDir() {
				watchDirs = append(watchDirs, path)
				return nil
			}

			if !zkSync.NeedSyncFile(path) {
				return nil
			}

			if err := zkSync.SyncNodeWithFile(path); err != nil {
				log.Printf("sync node error: path=%s, error=%s", path, err.Error())
				return err
			}

			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	// start wathcing
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	fsWatcher := FileSystemWatcher{
		Watcher: watcher,
		OnEvent: func(watcher *fsnotify.Watcher, event fsnotify.Event) {
			log.Println("event:", event)

			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// create or update
				stat, err := os.Stat(event.Name)
				if err != nil {
					log.Printf("get file stat error: file=%s, error=%s\n", event.Name, err.Error())
					return
				}

				if stat.IsDir() {
					if event.Op&fsnotify.Create == fsnotify.Create {
						// watch file
						if err := watcher.Add(event.Name); err != nil {
							log.Panicf("watch error: file=%s error=%s\n", event.Name, err.Error())
						}
					}

					// do not sync dir
					return
				}

				// filter files that no need to sync
				if !zkSync.NeedSyncFile(event.Name) {
					return
				}

				if err := zkSync.SyncNodeWithFile(event.Name); err != nil {
					log.Printf("sync node error: file=%s error=%s", event.Name, err.Error())
				}
			}

			if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
				maybeDir := path.Ext(event.Name) == ""
				if maybeDir {
					// stop watching
					watcher.Remove(event.Name)
				}

				// remove node
				if err := zkSync.RemoveNode(event.Name); err != nil {
					log.Printf("remove node error: node=%s error=%s\n", event.Name, err.Error())
				}
			}
		},
		OnError: func(err error) {
			log.Printf("watch error: error=%s\n", err.Error())
		},
	}

	if err := fsWatcher.Run(watchDirs); err != nil {
		log.Fatal(err)
	}
}
