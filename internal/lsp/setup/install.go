package setup

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"github.com/opencode-ai/opencode/internal/logging"
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

// GetPackageManager returns the appropriate package manager command for the current OS
func GetPackageManager() string {
	switch runtime.GOOS {
	case "darwin":
		// Check for Homebrew
		if _, err := exec.LookPath("brew"); err == nil {
			return "brew"
		}
		// Check for MacPorts
		if _, err := exec.LookPath("port"); err == nil {
			return "port"
		}
	case "linux":
		// Check for apt (Debian/Ubuntu)
		if _, err := exec.LookPath("apt"); err == nil {
			return "apt"
		}
		// Check for dnf (Fedora)
		if _, err := exec.LookPath("dnf"); err == nil {
			return "dnf"
		}
		// Check for yum (CentOS/RHEL)
		if _, err := exec.LookPath("yum"); err == nil {
			return "yum"
		}
		// Check for pacman (Arch)
		if _, err := exec.LookPath("pacman"); err == nil {
			return "pacman"
		}
		// Check for zypper (openSUSE)
		if _, err := exec.LookPath("zypper"); err == nil {
			return "zypper"
		}
	case "windows":
		// Check for Chocolatey
		if _, err := exec.LookPath("choco"); err == nil {
			return "choco"
		}
		// Check for Scoop
		if _, err := exec.LookPath("scoop"); err == nil {
			return "scoop"
		}
	}

	return ""
}

// GetSystemInstallCommand returns the system-specific installation command for a package
func GetSystemInstallCommand(packageName string) string {
	packageManager := GetPackageManager()

	switch packageManager {
	case "brew":
		return fmt.Sprintf("brew install %s", packageName)
	case "port":
		return fmt.Sprintf("sudo port install %s", packageName)
	case "apt":
		return fmt.Sprintf("sudo apt install -y %s", packageName)
	case "dnf":
		return fmt.Sprintf("sudo dnf install -y %s", packageName)
	case "yum":
		return fmt.Sprintf("sudo yum install -y %s", packageName)
	case "pacman":
		return fmt.Sprintf("sudo pacman -S --noconfirm %s", packageName)
	case "zypper":
		return fmt.Sprintf("sudo zypper install -y %s", packageName)
	case "choco":
		return fmt.Sprintf("choco install -y %s", packageName)
	case "scoop":
		return fmt.Sprintf("scoop install %s", packageName)
	}

	return ""
}

// InstallDependencies installs common dependencies for LSP servers
func InstallDependencies(ctx context.Context) []InstallationResult {
	results := []InstallationResult{}

	// Check for Node.js and npm
	if _, err := exec.LookPath("node"); err != nil {
		// Node.js is not installed, try to install it
		cmd := GetSystemInstallCommand("nodejs")
		if cmd == "" {
			results = append(results, InstallationResult{
				ServerName: "nodejs",
				Success:    false,
				Error:      fmt.Errorf("Node.js is not installed and could not determine how to install it"),
				Output:     "Please install Node.js manually: https://nodejs.org/",
			})
		} else {
			// Execute the installation command
			installCmd, installArgs := parseInstallCommand(cmd)
			execCmd := exec.CommandContext(ctx, installCmd, installArgs...)

			output, err := execCmd.CombinedOutput()
			if err != nil {
				results = append(results, InstallationResult{
					ServerName: "nodejs",
					Success:    false,
					Error:      fmt.Errorf("failed to install Node.js: %w", err),
					Output:     string(output),
				})
			} else {
				results = append(results, InstallationResult{
					ServerName: "nodejs",
					Success:    true,
					Output:     string(output),
				})
			}
		}
	}

	// Check for Python and pip
	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}

	if _, err := exec.LookPath(pythonCmd); err != nil {
		// Python is not installed, try to install it
		cmd := GetSystemInstallCommand("python3")
		if cmd == "" {
			results = append(results, InstallationResult{
				ServerName: "python",
				Success:    false,
				Error:      fmt.Errorf("python is not installed and could not determine how to install it"),
				Output:     "Please install Python manually: https://www.python.org/",
			})
		} else {
			// Execute the installation command
			installCmd, installArgs := parseInstallCommand(cmd)
			execCmd := exec.CommandContext(ctx, installCmd, installArgs...)

			output, err := execCmd.CombinedOutput()
			if err != nil {
				results = append(results, InstallationResult{
					ServerName: "python",
					Success:    false,
					Error:      fmt.Errorf("failed to install Python: %w", err),
					Output:     string(output),
				})
			} else {
				results = append(results, InstallationResult{
					ServerName: "python",
					Success:    true,
					Output:     string(output),
				})
			}
		}
	}

	// Check for Go
	if _, err := exec.LookPath("go"); err != nil {
		// Go is not installed, try to install it
		cmd := GetSystemInstallCommand("golang")
		if cmd == "" {
			results = append(results, InstallationResult{
				ServerName: "go",
				Success:    false,
				Error:      fmt.Errorf("go is not installed and could not determine how to install it"),
				Output:     "Please install Go manually: https://golang.org/",
			})
		} else {
			// Execute the installation command
			installCmd, installArgs := parseInstallCommand(cmd)
			execCmd := exec.CommandContext(ctx, installCmd, installArgs...)

			output, err := execCmd.CombinedOutput()
			if err != nil {
				results = append(results, InstallationResult{
					ServerName: "go",
					Success:    false,
					Error:      fmt.Errorf("failed to install Go: %w", err),
					Output:     string(output),
				})
			} else {
				results = append(results, InstallationResult{
					ServerName: "go",
					Success:    true,
					Output:     string(output),
				})
			}
		}
	}

	return results
}

// UpdateLSPConfig updates the LSP configuration in the config file
func UpdateLSPConfig(servers LSPServerMap) error {
	logging.Info("Updating LSP configuration with", len(servers), "servers")
	return nil
}
