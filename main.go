package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/0xABAD/filewatch"
	"github.com/0xABAD/gooey"
	"github.com/russross/blackfriday"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: meadow markdown-file")
		return
	}

	const GiB = 1 << 30

	file := os.Args[1]
	if info, err := os.Stat(file); err != nil {
		fmt.Printf("[FATAL] Can't inspect file %s: %s\n", file, err)
		return
	} else if info.IsDir() {
		fmt.Printf("[FATAL] %s is a directory.\n", file)
		return
	} else if info.Size() > 1*GiB {
		fmt.Printf("[FATAL] %s is HUGE (size > 1 GiB).\n", file)
		return
	}

	var (
		app    = Meadow{file, make(chan struct{})}
		done   = make(chan struct{})
		server = gooey.Server{
			IndexHtml:            INDEX,
			WebServeDir:          ".",
			ForceIndexAndFavIcon: true,
		}
	)
	defer close(done)

	go server.Start(done, &app)
	<-app.Done
}

type Meadow struct {
	Watched string        // File path of markdown file to be watched.
	Done    chan struct{} // Indicates when the app is ready to shutdown.
}

func (m *Meadow) Start(closed <-chan struct{}, incoming <-chan []byte, outgoing chan<- interface{}) {
	doneWatching := make(chan struct{})
	defer close(doneWatching)

	notify := make(chan os.Signal)
	defer close(notify)
	signal.Notify(notify, os.Kill, os.Interrupt)

	interval := 500 * time.Millisecond
	updates, err := filewatch.Watch(doneWatching, m.Watched, false, &interval)
	if err != nil {
		fmt.Printf("[FATAL] Failed to watch file, %s: %s", m.Watched, err)
		close(m.Done)
		return
	}

	for {
		select {
		case <-closed:
			close(m.Done)
			return

		case <-notify:
			close(m.Done)
			return

		case ups := <-updates:
			for _, u := range ups {
				bp := filepath.Base(u.AbsPath)
				if bp != filepath.Base(m.Watched) {
					continue
				}
				if u.WasRemoved {
					fmt.Printf("[FATAL] %s has been removed, exiting.\n", bp)
					close(m.Done)
					return
				}
				if u.Error != nil {
					fmt.Printf("[ERROR] %s update error: %s\n", bp, u.Error)
					fmt.Printf("[FATAL] Can't watch %s any longer, exiting.\n", bp)
					close(m.Done)
					return
				}
				if f, err := os.Open(u.AbsPath); err != nil {
					fmt.Println("[ERROR] Failed to open file:", err)
				} else {
					contents, err := ioutil.ReadAll(f)
					if err != nil {
						fmt.Println("[ERROR] Failed to read file contents:", err)
					} else {
						ext := blackfriday.CommonExtensions | blackfriday.Footnotes
						out := blackfriday.Run(contents, blackfriday.WithExtensions(ext))
						outgoing <- fmt.Sprintf("%s", string(out))
					}
				}
			}
		}
	}
}
