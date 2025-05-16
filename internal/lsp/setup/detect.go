package setup

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/lsp/protocol"
)

// LanguageScore represents a language with its importance score in the project
type LanguageScore struct {
	Language protocol.LanguageKind
	Score    int
}

// DetectProjectLanguages scans the workspace and returns a map of languages to their importance score
func DetectProjectLanguages(workspaceDir string) (map[protocol.LanguageKind]int, error) {
	languages := make(map[protocol.LanguageKind]int)

	skipDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
		".idea":        true,
		".vscode":      true,
		".github":      true,
		".gitlab":      true,
		".next":        true,
	}

	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip files larger than 1MB to avoid processing large binary files
		if info.Size() > 1024*1024 {
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Detect language based on file extension
		lang := detectLanguageFromPath(path)
		if lang != "" {
			languages[lang]++
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Check for special project files to boost language scores
	checkSpecialProjectFiles(workspaceDir, languages)

	return languages, nil
}

// detectLanguageFromPath detects the language based on the file path
func detectLanguageFromPath(path string) protocol.LanguageKind {
	ext := strings.ToLower(filepath.Ext(path))
	filename := strings.ToLower(filepath.Base(path))

	// Special case for Dockerfiles which don't have extensions
	if filename == "dockerfile" || strings.HasSuffix(filename, ".dockerfile") {
		return protocol.LangDockerfile
	}

	// Special case for Makefiles
	if filename == "makefile" || strings.HasSuffix(filename, ".mk") {
		return protocol.LangMakefile
	}

	// Special case for shell scripts without extensions
	if isShellScript(path) {
		return protocol.LangShellScript
	}

	// Map file extensions to languages
	switch ext {
	case ".go":
		return protocol.LangGo
	case ".js":
		return protocol.LangJavaScript
	case ".jsx":
		return protocol.LangJavaScriptReact
	case ".ts":
		return protocol.LangTypeScript
	case ".tsx":
		return protocol.LangTypeScriptReact
	case ".py":
		return protocol.LangPython
	case ".java":
		return protocol.LangJava
	case ".c":
		return protocol.LangC
	case ".cpp", ".cc", ".cxx", ".c++":
		return protocol.LangCPP
	case ".cs":
		return protocol.LangCSharp
	case ".php":
		return protocol.LangPHP
	case ".rb":
		return protocol.LangRuby
	case ".rs":
		return protocol.LangRust
	case ".swift":
		return protocol.LangSwift
	case ".kt", ".kts":
		return "kotlin"
	case ".scala":
		return protocol.LangScala
	case ".html", ".htm":
		return protocol.LangHTML
	case ".css":
		return protocol.LangCSS
	case ".scss":
		return protocol.LangSCSS
	case ".sass":
		return protocol.LangSASS
	case ".less":
		return protocol.LangLess
	case ".json":
		return protocol.LangJSON
	case ".xml":
		return protocol.LangXML
	case ".yaml", ".yml":
		return protocol.LangYAML
	case ".md", ".markdown":
		return protocol.LangMarkdown
	case ".sh", ".bash", ".zsh":
		return protocol.LangShellScript
	case ".sql":
		return protocol.LangSQL
	case ".dart":
		return protocol.LangDart
	case ".lua":
		return protocol.LangLua
	case ".ex", ".exs":
		return protocol.LangElixir
	case ".erl":
		return protocol.LangErlang
	case ".hs":
		return protocol.LangHaskell
	case ".pl", ".pm":
		return protocol.LangPerl
	case ".r":
		return protocol.LangR
	case ".vue":
		return "vue"
	case ".svelte":
		return "svelte"
	}

	return ""
}

// isShellScript checks if a file is a shell script by looking at its shebang
func isShellScript(path string) bool {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read the first line
	buf := make([]byte, 128)
	n, err := file.Read(buf)
	if err != nil || n < 2 {
		return false
	}

	// Check for shebang
	if buf[0] == '#' && buf[1] == '!' {
		line := string(buf[:n])
		return strings.Contains(line, "/bin/sh") ||
			strings.Contains(line, "/bin/bash") ||
			strings.Contains(line, "/bin/zsh") ||
			strings.Contains(line, "/usr/bin/env sh") ||
			strings.Contains(line, "/usr/bin/env bash") ||
			strings.Contains(line, "/usr/bin/env zsh")
	}

	return false
}

// checkSpecialProjectFiles looks for special project files to boost language scores
func checkSpecialProjectFiles(workspaceDir string, languages map[protocol.LanguageKind]int) {
	// Check for package.json (Node.js/JavaScript/TypeScript)
	if _, err := os.Stat(filepath.Join(workspaceDir, "package.json")); err == nil {
		languages[protocol.LangJavaScript] += 10

		// Check for TypeScript configuration
		if _, err := os.Stat(filepath.Join(workspaceDir, "tsconfig.json")); err == nil {
			languages[protocol.LangTypeScript] += 15
		}
	}

	// Check for go.mod (Go)
	if _, err := os.Stat(filepath.Join(workspaceDir, "go.mod")); err == nil {
		languages[protocol.LangGo] += 20
	}

	// Check for requirements.txt or setup.py (Python)
	if _, err := os.Stat(filepath.Join(workspaceDir, "requirements.txt")); err == nil {
		languages[protocol.LangPython] += 15
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "setup.py")); err == nil {
		languages[protocol.LangPython] += 15
	}

	// Check for pom.xml or build.gradle (Java)
	if _, err := os.Stat(filepath.Join(workspaceDir, "pom.xml")); err == nil {
		languages[protocol.LangJava] += 15
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "build.gradle")); err == nil {
		languages[protocol.LangJava] += 15
	}

	// Check for Cargo.toml (Rust)
	if _, err := os.Stat(filepath.Join(workspaceDir, "Cargo.toml")); err == nil {
		languages[protocol.LangRust] += 20
	}

	// Check for composer.json (PHP)
	if _, err := os.Stat(filepath.Join(workspaceDir, "composer.json")); err == nil {
		languages[protocol.LangPHP] += 15
	}

	// Check for Gemfile (Ruby)
	if _, err := os.Stat(filepath.Join(workspaceDir, "Gemfile")); err == nil {
		languages[protocol.LangRuby] += 15
	}

	// Check for CMakeLists.txt (C/C++)
	if _, err := os.Stat(filepath.Join(workspaceDir, "CMakeLists.txt")); err == nil {
		languages[protocol.LangCPP] += 10
		languages[protocol.LangC] += 5
	}

	// Check for pubspec.yaml (Dart/Flutter)
	if _, err := os.Stat(filepath.Join(workspaceDir, "pubspec.yaml")); err == nil {
		languages["dart"] += 20
	}

	// Check for mix.exs (Elixir)
	if _, err := os.Stat(filepath.Join(workspaceDir, "mix.exs")); err == nil {
		languages[protocol.LangElixir] += 20
	}
}

// GetPrimaryLanguages returns the top N languages in the project
func GetPrimaryLanguages(languages map[protocol.LanguageKind]int, limit int) []LanguageScore {
	// Convert map to slice for sorting
	var langScores []LanguageScore
	for lang, score := range languages {
		if lang != "" && score > 0 {
			langScores = append(langScores, LanguageScore{
				Language: lang,
				Score:    score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(langScores, func(i, j int) bool {
		return langScores[i].Score > langScores[j].Score
	})

	// Return top N languages or all if less than N
	if len(langScores) <= limit {
		return langScores
	}
	return langScores[:limit]
}

// DetectMonorepo checks if the workspace is a monorepo by looking for multiple project files
func DetectMonorepo(workspaceDir string) (bool, []string) {
	var projectDirs []string

	// Common project files to look for
	projectFiles := []string{
		"package.json",
		"go.mod",
		"pom.xml",
		"build.gradle",
		"Cargo.toml",
		"requirements.txt",
		"setup.py",
		"composer.json",
		"Gemfile",
		"pubspec.yaml",
		"mix.exs",
	}

	// Skip directories that are typically not relevant
	skipDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
		".idea":        true,
		".vscode":      true,
		".github":      true,
		".gitlab":      true,
	}

	// Check for root project files
	rootIsProject := false
	for _, file := range projectFiles {
		if _, err := os.Stat(filepath.Join(workspaceDir, file)); err == nil {
			rootIsProject = true
			break
		}
	}

	// Walk through the workspace to find project files in subdirectories
	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip the root directory since we already checked it
		if path == workspaceDir {
			return nil
		}

		// Skip files
		if !info.IsDir() {
			return nil
		}

		// Skip directories in the skipDirs list
		if skipDirs[info.Name()] {
			return filepath.SkipDir
		}

		// Check for project files in this directory
		for _, file := range projectFiles {
			if _, err := os.Stat(filepath.Join(path, file)); err == nil {
				// Found a project file, add this directory to the list
				relPath, err := filepath.Rel(workspaceDir, path)
				if err == nil {
					projectDirs = append(projectDirs, relPath)
				}
				return filepath.SkipDir // Skip subdirectories of this project
			}
		}

		return nil
	})
	if err != nil {
		logging.Warn("Error detecting monorepo", "error", err)
	}

	// It's a monorepo if we found multiple project directories
	isMonorepo := len(projectDirs) > 0

	// If the root is also a project, add it to the list
	if rootIsProject {
		projectDirs = append([]string{"."}, projectDirs...)
	}

	return isMonorepo, projectDirs
}
