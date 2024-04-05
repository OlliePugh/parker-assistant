package main

import (
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"

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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var payload RequestPayload

		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := pm.executeUserInput(payload.Query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		io.WriteString(w, result)

	})
	http.ListenAndServe(":8090", nil)
}

type RequestPayload struct {
	Query string
}
