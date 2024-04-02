package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/tmc/langchaingo/llms"
)

type Param struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Action struct {
	llms.FunctionDefinition
	plugin  *Plugin
	execute func(map[string]any) (string, error)
}

func externalPluginExecutor(a Action) func(map[string]any) (string, error) {
	return func(props map[string]any) (string, error) {
		// send HTTP post request to plugin
		toolParams, err := json.Marshal(props)
		if err != nil {
			return "", err
		}

		client := http.Client{
			Timeout: time.Second * 10, // Timeout after 2 seconds
		}

		// url builders
		url := url.URL{
			Scheme: "http",
			Host:   a.plugin.Address,
			Path:   "/" + a.Name}

		req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(toolParams))
		req.Header.Set("Content-Type", "application/json")

		if err != nil {
			slog.Error("Error creating request", "error", err)
			return "", err
		}

		res, err := client.Do(req)

		if err != nil {
			slog.Error("Error executing action", "actionName", a.Name, "error", err)
			return "", err
		}

		if res.Body != nil {
			defer res.Body.Close()
		}

		body, readErr := io.ReadAll(res.Body)

		if readErr != nil {
			slog.Error("Error reading response body", "error", readErr)
			return "", readErr
		}

		if res.StatusCode != http.StatusOK {
			slog.Error("Error executing action:", "actionName", a.Name, "status code", res.StatusCode)
			return "", errors.New(string(body))
		}

		slog.Debug("Executing action", "actionName", a.Name)
		return string(body), nil
	}
}

type Plugin struct {
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Actions  []Action `json:"actions"`
	RootPath string
}

func fetchPlugins(searchPath string) ([]Plugin, error) {
	plugins := []Plugin{}
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		slog.Error("Error reading directory", "error", err)
		return nil, err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(searchPath, entry.Name())
		if entry.IsDir() {
			// Recursively fetch plugins in subdirectories
			subPlugins, err := fetchPlugins(fullPath)
			if err != nil {
				return nil, err
			}
			plugins = append(plugins, subPlugins...)
		} else {
			if entry.Name() != "parker.json" {
				continue
			}

			plugin, err := parsePluginConfig(fullPath)

			if err != nil {
				return nil, err
			}
			plugins = append(plugins, plugin)
		}
	}

	return plugins, nil
}

func parsePluginConfig(path string) (Plugin, error) {
	// Read and parse the parker.json file
	pluginData, err := os.ReadFile(path)
	var plugin Plugin
	if err != nil {
		slog.Error("Error reading plugin file", "error", err)
		return plugin, err
	}

	err = json.Unmarshal(pluginData, &plugin)
	if err != nil {
		slog.Error("Error parsing plugin JSON", "error", err)
		return plugin, err
	}
	plugin.RootPath = filepath.Dir(path)

	// set plugin on each action to be plugin
	for i := 0; i < len(plugin.Actions); i++ {
		plugin.Actions[i].plugin = &plugin
		plugin.Actions[i].execute = externalPluginExecutor(plugin.Actions[i])
	}

	return plugin, nil

}
