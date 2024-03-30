package main

import (
	"fmt"
	"log"

	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	listener := setupSocket()
	out := make(chan PluginConnection)

	go listenForPlugins(listener, out)
	plugins, err := fetchPlugins("./plugins")

	if err != nil {
		fmt.Println("Error fetching plugins:", err)
		return
	}

	actions := make([]Action, 0)
	for _, plugin := range plugins {
		actions = append(actions, plugin.Actions...)
	}

	llm, err := openai.New()

	if err != nil {
		log.Fatal(err)
	}

	pm := ParkerModel{
		llm,
		actions,
	}
	fmt.Printf("pm: %v\n", pm)

	// pm.executeQuery(os.Args[1])
	connections := make([]PluginConnection, 0)
	closeHandler(&connections)

	// start all plugins
	for _, plugin := range plugins {
		go plugin.Initialise()
	}

	// listen for information on plugin channel
	for pc := range out {
		connections = append(connections, pc)
		fmt.Println(pc.id)
	}
}
