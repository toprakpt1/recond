package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/recond/internal/daemon"
)

func main() {
	d, err := daemon.New()
	if err != nil {
		log.Fatalf("failed to create daemon: %v", err)
	}

	if err := d.Start(); err != nil {
		log.Fatalf("failed to start daemon: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down daemon...")
	d.Stop()
}
