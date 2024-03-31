package main

import (
	"encoding/json"
	"fmt"
)

type ParkerAction struct {
	Tool      string         `json:"tool"`
	ToolInput map[string]any `json:"tool_input"`
}

func NewParkerResponse(response string, pm *ParkerModel) ([]ParkerAction, error) {
	var actions []ParkerAction
	err := json.Unmarshal([]byte(response), &actions)
	if err != nil {
		return nil, err
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("no actions found in response")
	}

	// print all actions
	for _, action := range actions {
		if !pm.isValidTool(action.Tool) {
			return nil, fmt.Errorf("invalid tool")
		}
	}

	return actions, nil
}
