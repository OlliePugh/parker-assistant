package main

import (
	"encoding/json"
	"time"

	"github.com/tmc/langchaingo/llms"
)

var CorePlugin = Plugin{
	Name: "core",
	Actions: []Action{{
		FunctionDefinition: llms.FunctionDefinition{
			Name:        "time.get",
			Description: "Get the current time",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {},
				"required": [],
				"returns": "The current time"
			}`),
		},
		plugin: nil},
		{
			FunctionDefinition: llms.FunctionDefinition{
				Name:        "date.get",
				Description: "Get the current date",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {},
					"required": [],
					"returns": "The current date in the format 'YYYY-MM-DD'"
				}`),
			},
			plugin: nil},
	},
}

func getCorePlugin() Plugin {
	// assign plugin to CorePlugin
	for _, action := range CorePlugin.Actions {
		action.plugin = &CorePlugin
	}
	return CorePlugin
}

func getTime() time.Time {
	return time.Now()
}
