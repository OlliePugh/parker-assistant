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

func (pm *ParkerModel) getLlmDecisioning(request string) ([]ParkerAction, error) {
	// add the user input to the conversation
	pm.conversation = append(pm.conversation, llms.TextParts("human", request))

	// fetch parkers actions given the conversation
	actions, rawResponse, err := pm.fetchActions([]llms.MessageContent{})

	// not sure this should live here
	if err != nil {
		actions, rawResponse, err = pm.correctInvalidResponse(rawResponse)
		if err != nil {
			return nil, err
		}
	}

	pm.conversation = append(pm.conversation, llms.TextParts("ai", rawResponse))
	return actions, err
}

func (pm *ParkerModel) executeUserInput(request string) ([]ParkerAction, error) {
	actions, err := pm.getLlmDecisioning(request)

	// print actions
	for _, action := range actions {
		fmt.Println(action)
	}

	if err != nil {
		fmt.Println("error fetching actions", err)
		return nil, err
	}

	return actions, nil
}

func (pm ParkerModel) correctInvalidResponse(invalidResponse string) ([]ParkerAction, string, error) {

	var userInvalidMessage = llms.TextParts("user", "{\"error\": \"Invalid response please try again\"}")

	// this will perform better if we include an actual error message
	invalidMessagesAndAttempts := []llms.MessageContent{
		llms.TextParts("ai", invalidResponse),
		// put in JSON to show that it isnt read from the user
		userInvalidMessage,
	}

	var reattempt func(int) ([]ParkerAction, string, error)
	// closure to reattempt fetching actions
	reattempt = func(attemptNumber int) ([]ParkerAction, string, error) {
		if attemptNumber > 3 {
			return nil, "", fmt.Errorf("failed to correct invalid response after 3 attempts")
		}

		actions, rawResponse, err := pm.fetchActions(invalidMessagesAndAttempts)
		invalidMessagesAndAttempts = append(invalidMessagesAndAttempts, llms.TextParts("ai", rawResponse))
		invalidMessagesAndAttempts = append(invalidMessagesAndAttempts, userInvalidMessage)

		// if we have retries left to use
		if err != nil && attemptNumber < 3 {
			reattempt(attemptNumber + 1)
		}

		return actions, rawResponse, err
	}

	correctedResult, rawResponse, err := reattempt(1)

	if err != nil {
		return nil, "", err
	}
	return correctedResult, rawResponse, err
}

func (pm ParkerModel) buildConversationHistory(historyAmount int) []llms.MessageContent {
	content := []llms.MessageContent{
		llms.TextParts("system", systemMessage(pm.generateFunctionDefinitions())),
	}
	l := len(pm.conversation)
	content = append(content, pm.conversation[max(0, l-historyAmount):l]...)
	return content
}

func (pm ParkerModel) fetchActions(tempMessages []llms.MessageContent) ([]ParkerAction, string, error) {
	ctx := context.Background()
	defer ctx.Done()

	co := llms.WithOptions(llms.CallOptions{
		Temperature: 0.2,
	})

	conversation := append(pm.buildConversationHistory(messageContextAmount), tempMessages...)

	completions, err := pm.llm.GenerateContent(ctx, conversation, co)

	if err != nil {
		log.Fatal(err)
		return nil, "", err
	}

	result := completions.Choices[0].Content
	actionsToComplete, err := NewParkerResponse(result, &pm)

	if err != nil {
		// TODO try again and but tell the agent that the response was invalid
		log.Fatal("Error parsing response", err, result)
	}

	return actionsToComplete, result, nil
}
