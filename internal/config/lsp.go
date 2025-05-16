package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/lsp/protocol"
)

// LSPServerInfo contains information about an LSP server
type LSPServerInfo struct {
	Name        string   // Display name of the server
	Command     string   // Command to execute
	Args        []string // Arguments to pass to the command
	InstallCmd  string   // Command to install the server
	Description string   // Description of the server
	Recommended bool     // Whether this is the recommended server for the language
	Options     any      // Additional options for the server
}

// UpdateLSPConfig updates the LSP configuration with the provided servers in the local config file
func UpdateLSPConfig(servers map[protocol.LanguageKind]LSPServerInfo) error {
	// Create a map for the LSP configuration
	lspConfig := make(map[string]LSPConfig)
	
	for lang, server := range servers {
		langStr := string(lang)
		
		lspConfig[langStr] = LSPConfig{
			Disabled: false,
			Command:  server.Command,
			Args:     server.Args,
			Options:  server.Options,
		}
	}
	
	return SaveLocalLSPConfig(lspConfig)
}

// SaveLocalLSPConfig saves only the LSP configuration to the local config file
func SaveLocalLSPConfig(lspConfig map[string]LSPConfig) error {
	// Get the working directory
	workingDir := WorkingDirectory()
	
	// Define the local config file path
	configPath := filepath.Join(workingDir, ".opencode.json")
	
	// Create a new configuration with only the LSP settings
	localConfig := make(map[string]any)
	
	// Read existing local config if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := json.Unmarshal(data, &localConfig); err != nil {
				logging.Warn("Failed to parse existing local config", "error", err)
				// Continue with empty config if we can't parse the existing one
				localConfig = make(map[string]any)
			}
		}
	}
	
	// Update only the LSP configuration
	localConfig["lsp"] = lspConfig
	
	// Marshal the configuration to JSON
	data, err := json.MarshalIndent(localConfig, "", "  ")
	if err != nil {
		return err
	}
	
	// Write the configuration to the file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}
	
	logging.Info("LSP configuration saved to local config file", configPath)
	return nil
}

// IsLSPConfigured checks if LSP is already configured
func IsLSPConfigured() bool {
	cfg := Get()
	return len(cfg.LSP) > 0
}