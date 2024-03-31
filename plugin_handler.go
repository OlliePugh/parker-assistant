package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tmc/langchaingo/llms"
)

type Param struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Action struct {
	llms.FunctionDefinition
	plugin *Plugin
}

type Plugin struct {
	Name         string   `json:"name"`
	StartCommand string   `json:"start"`
	Actions      []Action `json:"actions"`
	RootPath     string
	cmd          *exec.Cmd
}

func (p *Plugin) Initialise() error {
	if (p.cmd != nil) && (p.cmd.Process != nil) {
		fmt.Println("Plugin already running")
		return nil
	}

	// Change to the specified directory
	err := os.Chdir(p.RootPath)
	if err != nil {
		fmt.Println("Error changing directory:", err)
		return err
	}

	// Create the command
	p.cmd = exec.Command("sh", "-c", p.StartCommand)

	// Run the command
	_, err = p.cmd.Output()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return err
	}
	return nil
}

func (p *Plugin) Kill() error {
	if p.cmd == nil {
		fmt.Println("Plugin not running")
		return nil
	}

	// Kill the plugin
	err := p.cmd.Process.Kill()
	if err != nil {
		fmt.Println("Error killing plugin:", err)
		return err
	}

	// Wait for the plugin to exit
	err = p.cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for plugin to exit:", err)
		return err
	}

	return nil

}

func initialiseSocket() (net.Listener, error) {
	socketPath := "/tmp/plugin_socket"
	os.Remove(socketPath) // Remove existing socket file if exists

	// Create Unix domain socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Println("Error listening:", err)
		return nil, err
	}
	return listener, nil
}

func fetchPlugins(searchPath string) ([]Plugin, error) {
	plugins := []Plugin{}
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		fmt.Println("Error reading directory:", err)
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
		fmt.Println("Error reading plugin file:", err)
		return plugin, err
	}

	err = json.Unmarshal(pluginData, &plugin)
	if err != nil {
		fmt.Println("Error parsing plugin JSON:", err)
		return plugin, err
	}
	plugin.RootPath = filepath.Dir(path)

	// set plugin on each action to be plugin
	for _, action := range plugin.Actions {
		action.plugin = &plugin
	}

	return plugin, nil

}

// returns the id of the plugin
func handlePlugin(conn net.Conn) (string, error) {
	// Read and process messages from plugin
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from plugin:", err)
		return "", err
	}

	request := string(buffer[:n])
	return request, nil
}

func setupSocket() net.Listener {
	// Setup a socket to listen for plugins
	listener, err := initialiseSocket()
	if err != nil {
		fmt.Println("Error initialising socket:", err)
		return nil
	}
	return listener
}

type PluginConnection struct {
	id   string
	conn net.Conn
}

func listenForPlugins(listener net.Listener, out chan PluginConnection) {
	for {
		// Accept connections from plugins
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			return
		}

		id, err := handlePlugin(conn)
		if err != nil {
			fmt.Println("Error handling plugin:", err)
			continue
		}
		fmt.Println("Connected plugin:", id)
		out <- PluginConnection{id, conn}
	}
}
