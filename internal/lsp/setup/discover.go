package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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

// LSPServerMap maps languages to their available LSP servers
type LSPServerMap map[protocol.LanguageKind][]LSPServerInfo

// ServerDefinition defines an LSP server configuration
type ServerDefinition struct {
	Name       string
	Args       []string
	InstallCmd string
	Languages  []protocol.LanguageKind
}

// Common paths where LSP servers might be installed
var (
	// Common editor-specific paths
	vscodePath = getVSCodeExtensionsPath()
	neovimPath = getNeovimPluginsPath()

	// Common package manager paths
	npmBinPath       = getNpmGlobalBinPath()
	pipBinPath       = getPipBinPath()
	goBinPath        = getGoBinPath()
	cargoInstallPath = getCargoInstallPath()

	// Server definitions
	serverDefinitions = []ServerDefinition{
		{
			Name:       "typescript-language-server",
			Args:       []string{"--stdio"},
			InstallCmd: "npm install -g typescript-language-server typescript",
			Languages:  []protocol.LanguageKind{protocol.LangJavaScript, protocol.LangTypeScript, protocol.LangJavaScriptReact, protocol.LangTypeScriptReact},
		},
		{
			Name:       "deno",
			Args:       []string{"lsp"},
			InstallCmd: "https://deno.com/#installation",
			Languages:  []protocol.LanguageKind{protocol.LangJavaScript, protocol.LangTypeScript},
		},
		{
			Name:       "pylsp",
			Args:       []string{},
			InstallCmd: "pip install python-lsp-server",
			Languages:  []protocol.LanguageKind{protocol.LangPython},
		},
		{
			Name:       "pyright",
			Args:       []string{"--stdio"},
			InstallCmd: "npm install -g pyright",
			Languages:  []protocol.LanguageKind{protocol.LangPython},
		},
		{
			Name:       "jedi-language-server",
			Args:       []string{},
			InstallCmd: "pip install jedi-language-server",
			Languages:  []protocol.LanguageKind{protocol.LangPython},
		},
		{
			Name:       "gopls",
			Args:       []string{},
			InstallCmd: "go install golang.org/x/tools/gopls@latest",
			Languages:  []protocol.LanguageKind{protocol.LangGo},
		},
		{
			Name:       "rust-analyzer",
			Args:       []string{},
			InstallCmd: "rustup component add rust-analyzer",
			Languages:  []protocol.LanguageKind{protocol.LangRust},
		},
		{
			Name:       "jdtls",
			Args:       []string{},
			InstallCmd: "Manual installation required: https://github.com/eclipse/eclipse.jdt.ls",
			Languages:  []protocol.LanguageKind{protocol.LangJava},
		},
		{
			Name:       "clangd",
			Args:       []string{},
			InstallCmd: "Manual installation required: Install via package manager or https://clangd.llvm.org/installation.html",
			Languages:  []protocol.LanguageKind{protocol.LangC, protocol.LangCPP},
		},
		{
			Name:       "omnisharp",
			Args:       []string{"--languageserver"},
			InstallCmd: "npm install -g omnisharp-language-server",
			Languages:  []protocol.LanguageKind{protocol.LangCSharp},
		},
		{
			Name:       "intelephense",
			Args:       []string{"--stdio"},
			InstallCmd: "npm install -g intelephense",
			Languages:  []protocol.LanguageKind{protocol.LangPHP},
		},
		{
			Name:       "solargraph",
			Args:       []string{"stdio"},
			InstallCmd: "gem install solargraph",
			Languages:  []protocol.LanguageKind{protocol.LangRuby},
		},
		{
			Name:       "vscode-html-language-server",
			Args:       []string{"--stdio"},
			InstallCmd: "npm install -g vscode-langservers-extracted",
			Languages:  []protocol.LanguageKind{protocol.LangHTML},
		},
		{
			Name:       "vscode-css-language-server",
			Args:       []string{"--stdio"},
			InstallCmd: "npm install -g vscode-langservers-extracted",
			Languages:  []protocol.LanguageKind{protocol.LangCSS},
		},
		{
			Name:       "vscode-json-language-server",
			Args:       []string{"--stdio"},
			InstallCmd: "npm install -g vscode-langservers-extracted",
			Languages:  []protocol.LanguageKind{protocol.LangJSON},
		},
		{
			Name:       "yaml-language-server",
			Args:       []string{"--stdio"},
			InstallCmd: "npm install -g yaml-language-server",
			Languages:  []protocol.LanguageKind{protocol.LangYAML},
		},
		{
			Name:       "lua-language-server",
			Args:       []string{},
			InstallCmd: "https://github.com/LuaLS/lua-language-server/wiki/Getting-Started",
			Languages:  []protocol.LanguageKind{protocol.LangLua},
		},
		{
			Name:       "docker-langserver",
			Args:       []string{"--stdio"},
			InstallCmd: "npm install -g dockerfile-language-server-nodejs",
			Languages:  []protocol.LanguageKind{protocol.LangDockerfile},
		},
		{
			Name:       "bash-language-server",
			Args:       []string{"start"},
			InstallCmd: "npm install -g bash-language-server",
			Languages:  []protocol.LanguageKind{protocol.LangShellScript},
		},
		{
			Name:       "vls",
			Args:       []string{},
			InstallCmd: "npm install -g @volar/vue-language-server",
			Languages:  []protocol.LanguageKind{"vue"},
		},
		{
			Name:       "svelteserver",
			Args:       []string{"--stdio"},
			InstallCmd: "npm install -g svelte-language-server",
			Languages:  []protocol.LanguageKind{"svelte"},
		},
		{
			Name:       "dart",
			Args:       []string{"language-server"},
			InstallCmd: "https://dart.dev/get-dart",
			Languages:  []protocol.LanguageKind{protocol.LangDart},
		},
		{
			Name:       "elixir-ls",
			Args:       []string{},
			InstallCmd: "https://github.com/elixir-lsp/elixir-ls#installation",
			Languages:  []protocol.LanguageKind{protocol.LangElixir},
		},
	}

	// Recommended servers by language
	recommendedServers = map[protocol.LanguageKind]string{
		protocol.LangJavaScript:      "typescript-language-server",
		protocol.LangTypeScript:      "typescript-language-server",
		protocol.LangJavaScriptReact: "typescript-language-server",
		protocol.LangTypeScriptReact: "typescript-language-server",
		protocol.LangPython:          "pylsp",
		protocol.LangGo:              "gopls",
		protocol.LangRust:            "rust-analyzer",
		protocol.LangJava:            "jdtls",
		protocol.LangC:               "clangd",
		protocol.LangCPP:             "clangd",
		protocol.LangCSharp:          "omnisharp",
		protocol.LangPHP:             "intelephense",
		protocol.LangRuby:            "solargraph",
		protocol.LangHTML:            "vscode-html-language-server",
		protocol.LangCSS:             "vscode-css-language-server",
		protocol.LangJSON:            "vscode-json-language-server",
		protocol.LangYAML:            "yaml-language-server",
		protocol.LangLua:             "lua-language-server",
		protocol.LangDockerfile:      "docker-langserver",
		protocol.LangShellScript:     "bash-language-server",
		"vue":                        "vls",
		"svelte":                     "svelteserver",
		protocol.LangDart:            "dart",
		protocol.LangElixir:          "elixir-ls",
	}
)

// DiscoverInstalledLSPs checks common locations for installed LSP servers
func DiscoverInstalledLSPs() LSPServerMap {
	result := make(LSPServerMap)

	for _, def := range serverDefinitions {
		for _, lang := range def.Languages {
			checkAndAddServer(result, lang, def.Name, def.Args, def.InstallCmd)
		}
	}

	return result
}

// checkAndAddServer checks if an LSP server is installed and adds it to the result map
func checkAndAddServer(result LSPServerMap, lang protocol.LanguageKind, command string, args []string, installCmd string) {
	// Check if the command exists in PATH
	if path, err := exec.LookPath(command); err == nil {
		server := LSPServerInfo{
			Name:        command,
			Command:     path,
			Args:        args,
			InstallCmd:  installCmd,
			Description: fmt.Sprintf("%s language server", lang),
			Recommended: isRecommendedServer(lang, command),
		}

		result[lang] = append(result[lang], server)
	} else {
		// Check in common editor-specific paths
		if path := findInEditorPaths(command); path != "" {
			server := LSPServerInfo{
				Name:        command,
				Command:     path,
				Args:        args,
				InstallCmd:  installCmd,
				Description: fmt.Sprintf("%s language server", lang),
				Recommended: isRecommendedServer(lang, command),
			}
			result[lang] = append(result[lang], server)
		}
	}
}

// findInEditorPaths checks for an LSP server in common editor-specific paths
func findInEditorPaths(command string) string {
	// Check in VSCode extensions
	if vscodePath != "" {
		// VSCode extensions can have different structures, so we need to search for the binary
		matches, err := filepath.Glob(filepath.Join(vscodePath, "*", "**", command))
		if err == nil && len(matches) > 0 {
			for _, match := range matches {
				if isExecutable(match) {
					return match
				}
			}
		}

		// Check for node_modules/.bin in VSCode extensions
		matches, err = filepath.Glob(filepath.Join(vscodePath, "*", "node_modules", ".bin", command))
		if err == nil && len(matches) > 0 {
			for _, match := range matches {
				if isExecutable(match) {
					return match
				}
			}
		}
	}

	// Check in Neovim plugins
	if neovimPath != "" {
		matches, err := filepath.Glob(filepath.Join(neovimPath, "*", "**", command))
		if err == nil && len(matches) > 0 {
			for _, match := range matches {
				if isExecutable(match) {
					return match
				}
			}
		}
	}

	// Check in npm global bin
	if npmBinPath != "" {
		path := filepath.Join(npmBinPath, command)
		if isExecutable(path) {
			return path
		}
	}

	// Check in pip bin
	if pipBinPath != "" {
		path := filepath.Join(pipBinPath, command)
		if isExecutable(path) {
			return path
		}
	}

	// Check in Go bin
	if goBinPath != "" {
		path := filepath.Join(goBinPath, command)
		if isExecutable(path) {
			return path
		}
	}

	// Check in Cargo install
	if cargoInstallPath != "" {
		path := filepath.Join(cargoInstallPath, command)
		if isExecutable(path) {
			return path
		}
	}

	return ""
}

// isExecutable checks if a file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// On Windows, all files are "executable"
	if runtime.GOOS == "windows" {
		return !info.IsDir()
	}

	// On Unix-like systems, check the executable bit
	return !info.IsDir() && (info.Mode()&0111 != 0)
}

// isRecommendedServer checks if a server is the recommended one for a language
func isRecommendedServer(lang protocol.LanguageKind, command string) bool {
	recommended, ok := recommendedServers[lang]
	return ok && recommended == command
}

// GetRecommendedLSPServers returns the recommended LSP servers for the given languages
func GetRecommendedLSPServers(languages []LanguageScore) LSPServerMap {
	result := make(LSPServerMap)

	for _, lang := range languages {
		// Find the server definition for this language
		for _, def := range serverDefinitions {
			for _, defLang := range def.Languages {
				if defLang == lang.Language && isRecommendedServer(lang.Language, def.Name) {
					server := LSPServerInfo{
						Name:        def.Name,
						Command:     def.Name,
						Args:        def.Args,
						InstallCmd:  def.InstallCmd,
						Description: fmt.Sprintf("%s Language Server", lang.Language),
						Recommended: true,
					}
					result[lang.Language] = []LSPServerInfo{server}
					break
				}
			}
		}
	}

	return result
}

// Helper functions to get common paths

func getVSCodeExtensionsPath() string {
	var path string

	switch runtime.GOOS {
	case "windows":
		path = filepath.Join(os.Getenv("USERPROFILE"), ".vscode", "extensions")
	case "darwin":
		path = filepath.Join(os.Getenv("HOME"), ".vscode", "extensions")
	default: // Linux and others
		path = filepath.Join(os.Getenv("HOME"), ".vscode", "extensions")
	}

	if _, err := os.Stat(path); err != nil {
		// Try alternative locations
		switch runtime.GOOS {
		case "darwin":
			altPath := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Code", "User", "extensions")
			if _, err := os.Stat(altPath); err == nil {
				return altPath
			}
		case "linux":
			altPath := filepath.Join(os.Getenv("HOME"), ".config", "Code", "User", "extensions")
			if _, err := os.Stat(altPath); err == nil {
				return altPath
			}
		}
		return ""
	}

	return path
}

func getNeovimPluginsPath() string {
	var paths []string

	switch runtime.GOOS {
	case "windows":
		paths = []string{
			filepath.Join(os.Getenv("LOCALAPPDATA"), "nvim", "plugged"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "nvim", "site", "pack"),
		}
	default: // Linux, macOS, and others
		paths = []string{
			filepath.Join(os.Getenv("HOME"), ".local", "share", "nvim", "plugged"),
			filepath.Join(os.Getenv("HOME"), ".local", "share", "nvim", "site", "pack"),
			filepath.Join(os.Getenv("HOME"), ".config", "nvim", "plugged"),
		}
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func getNpmGlobalBinPath() string {
	// Try to get the npm global bin path
	cmd := exec.Command("npm", "config", "get", "prefix")
	output, err := cmd.Output()
	if err == nil {
		prefix := strings.TrimSpace(string(output))
		if prefix != "" {
			if runtime.GOOS == "windows" {
				return filepath.Join(prefix, "node_modules", ".bin")
			}
			return filepath.Join(prefix, "bin")
		}
	}

	// Fallback to common locations
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "npm")
	default:
		return filepath.Join(os.Getenv("HOME"), ".npm-global", "bin")
	}
}

func getPipBinPath() string {
	// Try to get the pip user bin path
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("python", "-m", "site", "--user-base")
	} else {
		cmd = exec.Command("python3", "-m", "site", "--user-base")
	}

	output, err := cmd.Output()
	if err == nil {
		userBase := strings.TrimSpace(string(output))
		if userBase != "" {
			return filepath.Join(userBase, "bin")
		}
	}

	// Fallback to common locations
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Python", "Scripts")
	default:
		return filepath.Join(os.Getenv("HOME"), ".local", "bin")
	}
}

func getGoBinPath() string {
	// Try to get the GOPATH
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// Fallback to default GOPATH
		switch runtime.GOOS {
		case "windows":
			gopath = filepath.Join(os.Getenv("USERPROFILE"), "go")
		default:
			gopath = filepath.Join(os.Getenv("HOME"), "go")
		}
	}

	return filepath.Join(gopath, "bin")
}

func getCargoInstallPath() string {
	// Try to get the Cargo install path
	cargoHome := os.Getenv("CARGO_HOME")
	if cargoHome == "" {
		// Fallback to default Cargo home
		switch runtime.GOOS {
		case "windows":
			cargoHome = filepath.Join(os.Getenv("USERPROFILE"), ".cargo")
		default:
			cargoHome = filepath.Join(os.Getenv("HOME"), ".cargo")
		}
	}

	return filepath.Join(cargoHome, "bin")
}
