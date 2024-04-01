package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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
			Timeout: time.Second * 2, // Timeout after 2 seconds
		}

		// url builders
		url := url.URL{
			Scheme: "http",
			Host:   a.plugin.Address,
			Path:   "/" + a.Name}

		req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(toolParams))
		req.Header.Set("Content-Type", "application/json")

		if err != nil {
			slog.Error("Error creating request:", err)
			return "", err
		}

		res, err := client.Do(req)

		if err != nil {
			slog.Error("Error executing action:", a.Name, err)
			return "", err
		}

		if res.StatusCode != http.StatusOK {
			slog.Error("Error executing action:", a.Name, "status code:", res.StatusCode)
			return "", nil
		}

		if res.Body != nil {
			defer res.Body.Close()
		}

		body, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			slog.Error("Error reading response body:", readErr)
			return "", readErr
		}

		slog.Debug("Executing action:", a.Name)
		return string(body), nil
	}
}

type Plugin struct {
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Actions  []Action `json:"actions"`
	RootPath string
	cmd      *exec.Cmd
}

func fetchPlugins(searchPath string) ([]Plugin, error) {
	plugins := []Plugin{}
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		slog.Error("Error reading directory:", err)
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
		slog.Error("Error reading plugin file:", err)
		return plugin, err
	}

	err = json.Unmarshal(pluginData, &plugin)
	if err != nil {
		slog.Error("Error parsing plugin JSON:", err)
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

// returns the id of the plugin
func handlePlugin(conn net.Conn) (string, error) {
	// Read and process messages from plugin
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		slog.Error("Error reading from plugin:", err)
		return "", err
	}

	request := string(buffer[:n])
	return request, nil
}

type PluginConnection struct {
	id   string
	conn net.Conn
}
