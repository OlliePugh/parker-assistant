package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// messageContextAmount is the number of previous messages to include in the context
const messageContextAmount = 10

func systemMessage(functions []llms.FunctionDefinition) string {
	slog.Debug("Generating system message")
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

You MUST always return a JSON ARRAY with the above structure.

You can use tools to gather more infomation before providing a final response to the user.

Provide a tell.user tool to read out a message to the user.
`, string(bs))
}

type ParkerModel struct {
	llm          llms.Model
	actions      map[string]Action
	conversation []llms.MessageContent
}

func NewParkerModel(llm llms.Model, actions map[string]Action) ParkerModel {
	slog.Debug("Creating new ParkerModel")
	return ParkerModel{
		llm:          llm,
		actions:      actions,
		conversation: []llms.MessageContent{},
	}
}

func (pm *ParkerModel) getLlmDecisioning(request string, role schema.ChatMessageType) ([]ParkerAction, error) {
	// add the user input to the conversation
	pm.conversation = append(pm.conversation, llms.TextParts(role, request))
	slog.Debug("conversation", "value", pm.conversation)

	// fetch parkers actions given the conversation
	actions, rawResponse, err := pm.fetchActions([]llms.MessageContent{})
	slog.Debug("actions", "value", actions)

	// not sure this should live here
	if err != nil {
		slog.Warn("action failed, trying again", err)
		actions, rawResponse, err = pm.correctInvalidResponse(rawResponse)
		if err != nil {
			return nil, err
		}
	}

	pm.conversation = append(pm.conversation, llms.TextParts("ai", rawResponse))
	return actions, err
}

type ToolResult struct {
	Tool       string
	ToolOutput any
}

func (pm *ParkerModel) executeUserInput(request string) (string, error) {
	actions, err := pm.getLlmDecisioning(request, "human")
	var finalString string
	// do while tell.user not in actions
loop:
	for {
		// Create a channel to receive the results from each routine
		resultCh := make(chan ToolResult)

		// Execute the actions in parallel
		for _, action := range actions {
			go func(action ParkerAction) {
				result, err := pm.actions[action.Tool].execute(action.ToolInput)
				if err != nil {
					slog.Error("error executing action", "error", err)
					resultCh <- ToolResult{action.Tool, err.Error()}
					return
				}
				resultCh <- ToolResult{action.Tool, result}
			}(action)
		}

		// make an array of results
		results := make([]ToolResult, 0)

		// collect the results
		for range actions {
			result := <-resultCh
			results = append(results, result)
		}

		slog.Debug("results", results)

		stringResults, err := json.Marshal(results)

		if err != nil {
			slog.Error("error marshalling results", "error", err)
			break
		}

		// if tell.user is in actions
		for _, result := range results {
			if result.Tool == "tell.user" {
				finalString = result.ToolOutput.(string)
				break loop // the user has been told something, no need for internal processing
			}
		}

		actions, err = pm.getLlmDecisioning("Here is the result of the tools you used: "+string(stringResults), "human")
		if err != nil {
			slog.Error("error fetching actions", "error", err)
			break
		}

	}

	if err != nil {
		slog.Error("error fetching actions", "error", err)
		return "", err
	}

	return finalString, nil
}

func (pm ParkerModel) correctInvalidResponse(invalidResponse string) ([]ParkerAction, string, error) {
	var userInvalidMessage = llms.TextParts("human", "{\"error\": \"Something went wrong processing your response please try again following the rules previously mentioned in the system message. Do not apologise for the issue as the user will not be aware there have been any issues, pretend as if you have not seen this message but still fix your response\"}")

	// this will perform better if we include an actual error message
	invalidMessagesAndAttempts := []llms.MessageContent{
		llms.TextParts("ai", invalidResponse),
		// put in JSON to show that it isnt read from the user
		userInvalidMessage,
	}

	var reattempt func(int) ([]ParkerAction, string, error)
	// closure to reattempt fetching actions
	reattempt = func(attemptNumber int) ([]ParkerAction, string, error) {
		slog.Debug("reattempting", "value", attemptNumber)

		actions, rawResponse, err := pm.fetchActions(invalidMessagesAndAttempts)
		slog.Debug("actions", "value", actions)

		// if it failed again and we have retries left to use
		if err != nil && attemptNumber < 3 {
			slog.Warn("failed to complete actions, attempting again", err)
			invalidMessagesAndAttempts = append(invalidMessagesAndAttempts, llms.TextParts("ai", rawResponse))
			invalidMessagesAndAttempts = append(invalidMessagesAndAttempts, userInvalidMessage)
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
		slog.Error("Failed to generate content", "error", err)
		return nil, "", err
	}

	result := completions.Choices[0].Content
	actionsToComplete, err := NewParkerResponse(result, &pm)

	if err != nil {
		slog.Error("Error parsing response", "error", err, "result", result)
		return nil, "", err
	}

	return actionsToComplete, result, nil
}
