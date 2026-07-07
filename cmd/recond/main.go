package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/toprakpt1/recond/internal/daemon"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug logging for all jobs")
	flag.Parse()

	d, err := daemon.New(*debug)
	if err != nil {
		log.Fatalf("failed to create daemon: %v", err)
	}

	if err := d.Start(); err != nil {
		log.Fatalf("failed to start daemon: %v", err)
	}

	if *debug {
		log.Println("debug mode enabled")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down daemon...")
	d.Stop()
}
