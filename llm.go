package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/tmc/langchaingo/llms"
)

// messageContextAmount is the number of previous messages to include in the context
const messageContextAmount = 10

func (pm ParkerModel) generateFunctionDefinitions() []llms.FunctionDefinition {
	var functions = []llms.FunctionDefinition{
		{
			Name:        "tell.user",
			Description: "Reads out a message to the user",
			Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
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
		{
			// I found that providing a tool for Ollama to give the final response significantly
			// increases the chances of success.
			Name:        "finalResponse",
			Description: "Provide the final response to the user query",
			Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"response": {
					"type": "string",
					"description": "The final response to the user query"
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

func systemMessage(functions []llms.FunctionDefinition) string {
	bs, err := json.Marshal(functions)
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf(`You are an AI assistant called Parker. You have access to the following tools:
%s

To use a tool, respond with a JSON array object with the following structure: 
[{
	"tool": <name of the called tool>,
	"tool_input": <parameters for the tool matching the above JSON schema>
}]

You MUST always return a JSON payload with the above structure.

You can use tools to gather more infomation before providing a final response to the user.

Provide a tell.user tool to read out a message to the user.
`, string(bs))
}

func (pm ParkerModel) fetchActions() string {
	ctx := context.Background()

	defer ctx.Done()

	content := []llms.MessageContent{
		llms.TextParts("system", systemMessage(pm.generateFunctionDefinitions())),
	}

	l := len(pm.conversation)

	content = append(content, pm.conversation[max(0, l-messageContextAmount):l]...)

	co := llms.WithOptions(llms.CallOptions{
		Temperature: 0.2,
	})
	completions, err := pm.llm.GenerateContent(ctx, content, co)
	if err != nil {
		log.Fatal(err)
	}

	return completions.Choices[0].Content
}

type ParkerModel struct {
	llm          llms.Model
	actions      []Action
	conversation []llms.MessageContent
}

func NewParkerModel(llm llms.Model, actions []Action) ParkerModel {
	return ParkerModel{
		llm:          llm,
		actions:      actions,
		conversation: []llms.MessageContent{},
	}
}

func (pm *ParkerModel) executeQuery(request string) string {
	pm.conversation = append(pm.conversation, llms.TextParts("human", request))
	returnedActions := pm.fetchActions()
	pm.conversation = append(pm.conversation, llms.TextParts("ai", returnedActions))
	fmt.Println("\n\n" + returnedActions)
	return returnedActions
}
