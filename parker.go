package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	listener := setupSocket()
	pluginConnections := make(chan PluginConnection)

	go listenForPlugins(listener, pluginConnections)
	plugins, err := fetchPlugins("./plugins")

	if err != nil {
		fmt.Println("Error fetching plugins:", err)
		return
	}

	corePlugin := getCorePlugin()

	actions := make([]Action, 0)
	actions = append(actions, corePlugin.Actions...)
	for _, plugin := range plugins {
		actions = append(actions, plugin.Actions...)
	}

	llm, err := openai.New()

	if err != nil {
		log.Fatal(err)
	}

	pm := NewParkerModel(
		llm,
		actions,
	)

	connections := make([]PluginConnection, 0)
	closeHandler(&connections)

	// start all plugins
	for _, plugin := range plugins {
		go plugin.Initialise()
	}

	go listenForUserInput(&pm)

	// listen for information on plugin channel
	for pc := range pluginConnections {
		connections = append(connections, pc)
	}
}

func listenForUserInput(pm *ParkerModel) {
	for {
		buf := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		sentence, err := buf.ReadBytes('\n')
		if err != nil {
			fmt.Println(err)
		} else {
			pm.executeQuery(string(sentence))
		}
	}
}
