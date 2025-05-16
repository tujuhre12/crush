package setup

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/lsp/protocol"
)

// InstallationResult represents the result of an LSP server installation
type InstallationResult struct {
	ServerName string
	Success    bool
	Error      error
	Output     string
}

// InstallLSPServer installs an LSP server for the given language
func InstallLSPServer(ctx context.Context, server LSPServerInfo) InstallationResult {
	result := InstallationResult{
		ServerName: server.Name,
		Success:    false,
	}

	// Check if the server is already installed
	if _, err := exec.LookPath(server.Command); err == nil {
		result.Success = true
		result.Output = fmt.Sprintf("%s is already installed", server.Name)
		return result
	}

	// Parse the installation command
	installCmd, installArgs := parseInstallCommand(server.InstallCmd)

	// If the installation command is a URL or instructions, return with error
	if strings.HasPrefix(installCmd, "http") || strings.Contains(installCmd, "Manual installation") {
		result.Error = fmt.Errorf("manual installation required: %s", server.InstallCmd)
		result.Output = server.InstallCmd
		return result
	}

	// Execute the installation command
	cmd := exec.CommandContext(ctx, installCmd, installArgs...)

	// Set up pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		result.Error = fmt.Errorf("failed to create stdout pipe: %w", err)
		return result
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		result.Error = fmt.Errorf("failed to create stderr pipe: %w", err)
		return result
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		result.Error = fmt.Errorf("failed to start installation: %w", err)
		return result
	}

	// Read output
	stdoutBytes, _ := io.ReadAll(stdout)
	stderrBytes, _ := io.ReadAll(stderr)

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		result.Error = fmt.Errorf("installation failed: %w", err)
		result.Output = fmt.Sprintf("stdout: %s\nstderr: %s", string(stdoutBytes), string(stderrBytes))
		return result
	}

	// Check if the server is now installed
	if _, err := exec.LookPath(server.Command); err != nil {
		result.Error = fmt.Errorf("installation completed but server not found in PATH")
		result.Output = fmt.Sprintf("stdout: %s\nstderr: %s", string(stdoutBytes), string(stderrBytes))
		return result
	}

	result.Success = true
	result.Output = fmt.Sprintf("Successfully installed %s\nstdout: %s\nstderr: %s",
		server.Name, string(stdoutBytes), string(stderrBytes))

	return result
}

// parseInstallCommand parses an installation command string into command and arguments
func parseInstallCommand(installCmd string) (string, []string) {
	parts := strings.Fields(installCmd)
	if len(parts) == 0 {
		return "", nil
	}

	return parts[0], parts[1:]
}

// GetInstallationCommands returns the installation commands for the given servers
func GetInstallationCommands(servers LSPServerMap) map[string]string {
	commands := make(map[string]string)

	for _, serverList := range servers {
		for _, server := range serverList {
			if server.Recommended {
				commands[server.Name] = server.InstallCmd
			}
		}
	}

	return commands
}

// VerifyInstallation verifies that an LSP server is correctly installed
func VerifyInstallation(serverName string) bool {
	_, err := exec.LookPath(serverName)
	return err == nil
}

// UpdateLSPConfig updates the LSP configuration in the config file
func UpdateLSPConfig(servers map[protocol.LanguageKind]LSPServerInfo) error {
	// Create a map for the LSP configuration
	lspConfig := make(map[string]config.LSPConfig)

	for lang, server := range servers {
		langStr := string(lang)

		lspConfig[langStr] = config.LSPConfig{
			Disabled: false,
			Command:  server.Command,
			Args:     server.Args,
			Options:  server.Options,
		}
	}

	return config.SaveLocalLSPConfig(lspConfig)
}
