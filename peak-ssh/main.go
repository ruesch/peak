package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aleksana/peak/internal/vfs"
	"github.com/aleksana/peak/internal/vfs/afero"
)

func main() {
	var (
		socketPath = flag.String("s", "", "Unix socket path to listen on")
		peakSocket = flag.String("p", "", "Peak editor 9P socket to create windows in (optional)")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -s socket_path [-p peak_socket] [initial_host]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *socketPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	var peakFs afero.Fs
	if *peakSocket != "" {
		var err error
		peakFs, err = vfs.NewNinePClientFs("unix", *peakSocket)
		if err != nil {
			log.Printf("Warning: failed to connect to peak at %s: %v", *peakSocket, err)
			peakFs = nil
		} else {
			log.Printf("Connected to peak editor at %s", *peakSocket)
		}
	}

	srv := vfs.NewNinePSrv(newHostFs(NewSftpFs(), peakFs))
	os.Remove(*socketPath)

	log.Printf("Starting SSH/SFTP 9P server on %s", *socketPath)
	if err := srv.Serve("unix", *socketPath); err != nil {
		log.Fatalf("9P server error: %v", err)
	}
}
