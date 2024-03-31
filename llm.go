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
	_, rawResponse := pm.fetchActions()

	// is the response valid?

	// does the response contain a tell.user

	pm.conversation = append(pm.conversation, llms.TextParts("ai", rawResponse))
	fmt.Println("\n\n" + rawResponse)
	return rawResponse
}

func (pm ParkerModel) fetchActions() ([]ParkerAction, string) {
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

	result := completions.Choices[0].Content
	actionsToComplete, err := NewParkerResponse(result, &pm)

	if err != nil {
		// TODO try again and but tell the agent that the response was invalid
		log.Fatal("Error parsing response", err, result)
	}

	return actionsToComplete, result
}
