package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/aleksana/peak/internal/vfs"
	"github.com/aleksana/peak/internal/vfs/afero"
)

func main() {
	socket := flag.String("s", "", "peak 9P socket (default: ~/.peak/9p)")
	flag.Parse()
	if *socket == "" {
		home, _ := os.UserHomeDir()
		*socket = filepath.Join(home, ".peak", "9p")
	}
	fs, err := vfs.NewNinePClientFs("unix", *socket)
	if err != nil {
		log.Fatalf("connect to peak: %v", err)
	}
	log.Printf("connected to peak at %s", *socket)
	watchEvents(fs)
}

// watchEvents opens /event and blocks on it, starting a watchWindow goroutine
// for each "new <id>" line. "get <id> <filename>" and "put <id> <filename>"
// lines are forwarded to the corresponding window's retitle channel so it can
// re-detect the language after a file change.
func watchEvents(fs afero.Fs) {
	var mu sync.Mutex
	retitleChans := make(map[int]chan<- string)

	start := func(id int) {
		mu.Lock()
		_, already := retitleChans[id]
		var ch chan string
		if !already {
			ch = make(chan string, 4)
			retitleChans[id] = ch
		}
		mu.Unlock()
		if !already {
			go func() {
				watchWindow(fs, id, ch)
				mu.Lock()
				delete(retitleChans, id)
				mu.Unlock()
			}()
		}
	}

	retitle := func(id int, filename string) {
		mu.Lock()
		ch := retitleChans[id]
		mu.Unlock()
		if ch != nil {
			select {
			case ch <- filename:
			default:
				// channel full; drop — next event will carry the latest name
			}
		}
	}

	// Open the event stream before snapshotting so we don't miss windows
	// that open during the bootstrap.
	eventF, err := fs.Open("/event")
	if err != nil {
		log.Fatalf("open /event: %v", err)
	}
	defer eventF.Close()

	// Bootstrap: start watching windows that are already open.
	if entries, err := afero.ReadDir(fs, "/"); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				if id, err := strconv.Atoi(e.Name()); err == nil {
					start(id)
				}
			}
		}
	}

	scanner := bufio.NewScanner(eventF)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		switch parts[0] {
		case "new":
			start(id)
		case "get", "put":
			if len(parts) >= 3 {
				retitle(id, parts[2])
			}
		}
	}
}
