package main

import (
	"github.com/tmc/langchaingo/llms"
)

func (pm ParkerModel) isValidTool(tool string) bool {
	for _, action := range pm.actions {
		if action.Name == tool || tool == "tell.user" { // tell user is a special case
			return true
		}
	}
	return false
}

func (pm ParkerModel) generateFunctionDefinitions() []llms.FunctionDefinition {
	functionDefinitions := []llms.FunctionDefinition{}
	for _, action := range pm.actions {
		functionDefinitions = append(functionDefinitions, action.FunctionDefinition)
	}

	return functionDefinitions
}
