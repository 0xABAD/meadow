package main

import (
	"flag"
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

var (
	css  = flag.String("css", "", "Watch a custom CSS file for changes to include")
	help bool
)

func init() {
	const helpMsg = "Show this message"
	flag.BoolVar(&help, "help", false, helpMsg)
	flag.BoolVar(&help, "h", false, helpMsg)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [OPTIONS] Markdown-File\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if help || len(flag.Args()) != 1 {
		flag.Usage()
		return
	}

	const GiB = 1 << 30

	file := flag.Arg(0)
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
		done = make(chan struct{})
		app  = Meadow{
			Watched: file,
			Done:    make(chan struct{}),
			CSS:     *css,
		}
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
	CSS     string        // File path of custom CSS file to be watched (empty for none).
	Done    chan struct{} // Indicates when the app is ready to shutdown.
}

func (m *Meadow) Start(closed <-chan struct{}, incoming <-chan []byte, outgoing chan<- interface{}) {
	doneWatching := make(chan struct{})
	defer close(doneWatching)

	notify := make(chan os.Signal)
	defer close(notify)
	signal.Notify(notify, os.Kill, os.Interrupt)

	interval := 500 * time.Millisecond
	updatesMD, err := filewatch.Watch(doneWatching, m.Watched, false, &interval)
	if err != nil {
		fmt.Printf("[FATAL] Failed to watch file, %s: %s", m.Watched, err)
		close(m.Done)
		return
	}

	var updatesCSS <-chan []filewatch.Update
	if m.CSS != "" {
		ups, err := filewatch.Watch(doneWatching, m.CSS, false, &interval)
		if err != nil {
			fmt.Printf("[ERROR] Failed to watch file, %s: %s", m.CSS, err)
		} else {
			updatesCSS = ups
		}
	}

	type Message struct {
		Tag     string
		Content string
	}

	for {
		select {
		case <-closed:
			close(m.Done)
			return

		case <-notify:
			close(m.Done)
			return

		case ups := <-updatesMD:
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
						outgoing <- Message{"body", string(out)}
					}
				}
			}

		case ups := <-updatesCSS:
			for _, u := range ups {
				bp := filepath.Base(u.AbsPath)
				if bp != filepath.Base(m.CSS) {
					continue
				}
				if u.WasRemoved {
					fmt.Printf("[ERROR] %s has been removed, ignoring for now on.\n", bp)
					m.CSS = ""
				}
				if u.Error != nil {
					fmt.Printf("[ERROR] %s update error: %s\n", bp, u.Error)
					fmt.Printf("[ERROR] Can't watch %s any longer, ignoring for now on.\n", bp)
					m.CSS = ""
				}
				if f, err := os.Open(u.AbsPath); err != nil {
					fmt.Println("[ERROR] Failed to open file:", err)
				} else {
					contents, err := ioutil.ReadAll(f)
					if err != nil {
						fmt.Println("[ERROR] Failed to read file contents:", err)
					} else {
						outgoing <- Message{"css", string(contents)}
					}
				}
			}
		}
	}
}
