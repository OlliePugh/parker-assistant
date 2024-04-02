package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/langchaingo/llms"
)

var CorePlugin = Plugin{
	Name: "core",
	Actions: []Action{
		{
			FunctionDefinition: llms.FunctionDefinition{
				Name:        "tell.user",
				Description: "Reads out a message to the user, refer to the user as sir",
				Parameters: json.RawMessage(`{
					"type": "object",
					"profperties": {
						"response": {
							"type": "string",
							"description": "Your response to the user request"
						}
					},
					"required": [
						"response"
					]
					}`),
			},
			execute: tellUser,
			plugin:  nil,
		},
		{
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
			execute: getTime,
			plugin:  nil,
		},
		{
			FunctionDefinition: llms.FunctionDefinition{
				Name:        "day.get",
				Description: "Get the current day in the week",
				Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {},
				"required": [],
				"returns": "The current day, i.e. Tuesday"
			}`),
			},
			execute: getDay,
			plugin:  nil,
		},
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
			execute: getDate,
			plugin:  nil,
		},
	},
}

func getCorePlugin() Plugin {
	// assign plugin to CorePlugin
	for i := 0; i < len(CorePlugin.Actions); i++ {
		CorePlugin.Actions[i].plugin = &CorePlugin
	}
	return CorePlugin
}

type tellUserInput struct {
	Response string `json:"response"`
}

func tellUser(input map[string]any) (string, error) {
	slog.Debug("Executing action: tell.user")
	// convert input to json
	jsonString, err := json.Marshal(input)
	if err != nil {
		slog.Error("Error marshalling input", "error", err)
		return "", err
	}

	// unmarshal json
	var parsedInput tellUserInput
	err = json.Unmarshal(jsonString, &parsedInput)
	if err != nil {
		slog.Error("Error unmarshalling input", "error", err)
		return "", err
	}

	fmt.Println(parsedInput.Response)
	return "Message read to user", nil
}

func getTime(_ map[string]any) (string, error) {
	return time.Now().Format("15:04:05"), nil
}

func getDate(_ map[string]any) (string, error) {
	return time.Now().Format("2006-01-02"), nil
}

func getDay(_ map[string]any) (string, error) {
	return time.Now().Weekday().String(), nil
}
