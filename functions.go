package main

import (
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

func (pm ParkerModel) isValidTool(tool string) bool {
	for _, action := range pm.actions {
		fmt.Println(action.Name)
		if action.Name == tool || tool == "tell.user" { // tell user is a special case
			return true
		}
	}
	return false
}

func (pm ParkerModel) generateFunctionDefinitions() []llms.FunctionDefinition {
	var functions = []llms.FunctionDefinition{
		{
			Name:        "tell.user",
			Description: "Reads out a message to the user",
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
		}}

	var functionDefinitions = []llms.FunctionDefinition{}
	for _, action := range pm.actions {
		functionDefinitions = append(functionDefinitions, action.FunctionDefinition)
	}
	functions = append(functions, (functionDefinitions)...)

	return functions
}
