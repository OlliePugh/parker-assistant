package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func closeHandler(connections *[]PluginConnection) {
	sigc := make(chan os.Signal, 1)
	// We want to be notified about Strg+C etc.
	signal.Notify(sigc, syscall.SIGINT)

	go func() {
		<-sigc
		fmt.Println("Received SIGINT. Closing all connections...")
		// Close all connections
		for _, pc := range *connections {
			pc.conn.Close()
			fmt.Println("Closing connection to plugin", pc.id)
		}
		os.Exit(1)
	}()
}
