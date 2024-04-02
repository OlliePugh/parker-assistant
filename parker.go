package main

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	var programLevel = new(slog.LevelVar) // Info by default
	// h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
	// slog.SetDefault(slog.New(h))
	programLevel.Set(slog.LevelDebug)

	plugins, err := fetchPlugins("plugins")
	if err != nil {
		log.Fatal(err)
		return
	}
	corePlugin := getCorePlugin()

	actions := make(map[string]Action, 0)
	for _, action := range corePlugin.Actions {
		actions[action.FunctionDefinition.Name] = action
	}

	for _, plugin := range plugins {
		for _, action := range plugin.Actions {
			actions[action.FunctionDefinition.Name] = action
		}
	}

	llm, err := openai.New()

	if err != nil {
		log.Fatal(err)
	}

	pm := NewParkerModel(
		llm,
		actions,
	)

	closeHandler()

	listenForUserInput(&pm)
}

func listenForUserInput(pm *ParkerModel) {
	for {
		buf := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		sentence, err := buf.ReadBytes('\n')
		if err != nil {
			slog.Error("Error reading user input", "error", err)
		} else {
			slog.Debug("User input:", "value", string(sentence))
			pm.executeUserInput(string(sentence))
		}
	}
}
