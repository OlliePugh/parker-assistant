package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func closeHandler() {
	sigc := make(chan os.Signal, 1)
	// We want to be notified about Strg+C etc.
	signal.Notify(sigc, syscall.SIGINT)

	go func() {
		<-sigc
		slog.Info("Received SIGINT. Closing all connections...")
		// Close all connections
		os.Exit(1)
	}()
}
