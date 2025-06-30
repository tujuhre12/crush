package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/lsp/protocol"
)

type DefinitionsTool struct {
	lspClients map[string]*lsp.Client
}

const (
	DefinitionsToolName = "definitions"
)

func NewDefinitionsTool(lspClients map[string]*lsp.Client) BaseTool {
	return &DefinitionsTool{
		lspClients: lspClients,
	}
}

func (t *DefinitionsTool) Name() string {
	return DefinitionsToolName
}

func (t *DefinitionsTool) Info() ToolInfo {
	return ToolInfo{
		Name: DefinitionsToolName,
		Description: `Gets all symbol definitions from a file using the appropriate LSP server.

WHEN TO USE THIS TOOL:
- Use when you need to understand the structure and symbols in a code file
- Helpful for exploring unfamiliar codebases and understanding what's defined in a file
- Good for finding functions, classes, variables, types, interfaces, and other symbols
- Useful before making changes to understand existing code structure

HOW TO USE:
- Provide the path to a source code file
- The tool will automatically select the appropriate LSP server based on file extension
- Results show symbol names, types, locations, and hierarchical relationships

FEATURES:
- Supports multiple programming languages (Go, TypeScript, JavaScript, Rust, Python, etc.)
- Shows hierarchical symbol relationships (classes with their methods, etc.)
- Provides precise line number locations for each symbol
- Includes symbol details and documentation when available

LIMITATIONS:
- Requires an active LSP server for the file's language
- Only works with files that the LSP server can parse
- Results depend on LSP server capabilities and file syntax correctness

TIPS:
- Use this tool to get an overview of a file's structure before editing
- Combine with other tools like View to see the actual code implementation
- Helpful for understanding large files with many symbols`,
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The path to the file to get definitions from",
			},
		},
		Required: []string{"file_path"},
	}
}

type DefinitionsParams struct {
	FilePath string `json:"file_path"`
}

type DefinitionResult struct {
	Name     string             `json:"name"`
	Kind     string             `json:"kind"`
	Detail   string             `json:"detail,omitempty"`
	Range    string             `json:"range"`
	Children []DefinitionResult `json:"children,omitempty"`
}

func (t *DefinitionsTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var definitionsParams DefinitionsParams
	if err := json.Unmarshal([]byte(params.Input), &definitionsParams); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	if definitionsParams.FilePath == "" {
		return NewTextErrorResponse("file_path parameter is required"), nil
	}

	// Check if file exists
	if _, err := os.Stat(definitionsParams.FilePath); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("File does not exist: %s", definitionsParams.FilePath)), nil
	}

	// Find the appropriate LSP client for this file
	client, clientName, err := t.findLSPClient(definitionsParams.FilePath)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("No suitable LSP client found for file %s: %v\n\nAvailable LSP clients: %s",
			definitionsParams.FilePath, err, t.getAvailableClients())), nil
	}

	// Check if the server is ready
	if client.GetServerState() != lsp.StateReady {
		return NewTextErrorResponse(fmt.Sprintf("LSP server %s is not ready (state: %v). Please wait for the server to initialize or check the server configuration.",
			clientName, client.GetServerState())), nil
	}

	// Ensure the file is open in the LSP
	if err := client.OpenFileOnDemand(ctx, definitionsParams.FilePath); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to open file in LSP: %v", err)), nil
	}

	// Get document symbols
	documentURI := protocol.URIFromPath(definitionsParams.FilePath)
	symbolParams := protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: documentURI,
		},
	}

	result, err := client.DocumentSymbol(ctx, symbolParams)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to get document symbols from %s LSP: %v\n\nThis might happen if:\n- The file has syntax errors\n- The LSP server doesn't support this file type\n- The file is not part of a recognized project structure",
			clientName, err)), nil
	}

	// Parse the result and format it
	definitions, err := t.parseSymbolResult(result)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to parse symbols: %v", err)), nil
	}

	if len(definitions) == 0 {
		return NewTextResponse(fmt.Sprintf("No definitions found in file %s.\n\nThis might happen if:\n- The file is empty or contains only comments\n- The file has syntax errors preventing symbol extraction\n- The LSP server doesn't recognize symbols in this file type",
			definitionsParams.FilePath)), nil
	}

	// Format the output
	output := t.formatDefinitions(definitions, definitionsParams.FilePath, clientName)
	return NewTextResponse(output), nil
}

// findLSPClient finds the most appropriate LSP client for the given file
func (t *DefinitionsTool) findLSPClient(filePath string) (*lsp.Client, string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Map file extensions to preferred LSP client types
	preferredClients := map[string][]string{
		".go":   {"gopls", "go"},
		".ts":   {"typescript", "vtsls", "tsserver"},
		".tsx":  {"typescript", "vtsls", "tsserver"},
		".js":   {"typescript", "vtsls", "tsserver"},
		".jsx":  {"typescript", "vtsls", "tsserver"},
		".rs":   {"rust-analyzer", "rust"},
		".py":   {"pyright", "pylsp", "python"},
		".java": {"jdtls", "java"},
		".c":    {"clangd", "ccls", "c"},
		".cpp":  {"clangd", "ccls", "cpp", "c++"},
		".cc":   {"clangd", "ccls", "cpp", "c++"},
		".cxx":  {"clangd", "ccls", "cpp", "c++"},
		".h":    {"clangd", "ccls", "c", "cpp"},
		".hpp":  {"clangd", "ccls", "cpp", "c++"},
		".cs":   {"omnisharp", "csharp"},
		".php":  {"intelephense", "php"},
		".rb":   {"solargraph", "ruby"},
		".lua":  {"lua-language-server", "lua"},
		".sh":   {"bash-language-server", "bash"},
		".bash": {"bash-language-server", "bash"},
		".zsh":  {"bash-language-server", "bash"},
	}

	// First, try to find a client that matches the preferred types for this file extension
	if preferred, exists := preferredClients[ext]; exists {
		for _, clientType := range preferred {
			for name, client := range t.lspClients {
				if strings.Contains(strings.ToLower(name), clientType) && client.GetServerState() == lsp.StateReady {
					return client, name, nil
				}
			}
		}
	}

	// If no preferred client found, try any available ready client
	// This is a fallback for generic LSP servers that might support multiple languages
	for name, client := range t.lspClients {
		if client.GetServerState() == lsp.StateReady {
			return client, name, nil
		}
	}

	return nil, "", fmt.Errorf("no suitable LSP client found for file extension %s", ext)
}

// getAvailableClients returns a string listing all available LSP clients
func (t *DefinitionsTool) getAvailableClients() string {
	if len(t.lspClients) == 0 {
		return "none"
	}

	var clients []string
	for name := range t.lspClients {
		clients = append(clients, name)
	}
	return strings.Join(clients, ", ")
}

// parseSymbolResult parses the LSP symbol result into our format
func (t *DefinitionsTool) parseSymbolResult(result protocol.Or_Result_textDocument_documentSymbol) ([]DefinitionResult, error) {
	var definitions []DefinitionResult

	// The result can be either []DocumentSymbol or []SymbolInformation
	// Try to unmarshal as DocumentSymbol first (newer format with hierarchy)
	if result.Value != nil {
		// Convert interface{} to []byte for unmarshaling
		resultBytes, err := json.Marshal(result.Value)
		if err != nil {
			return definitions, fmt.Errorf("failed to marshal result: %v", err)
		}

		// Try DocumentSymbol format first (hierarchical)
		var docSymbols []protocol.DocumentSymbol
		if err := json.Unmarshal(resultBytes, &docSymbols); err == nil && len(docSymbols) > 0 {
			for _, symbol := range docSymbols {
				definitions = append(definitions, t.convertDocumentSymbol(symbol))
			}
			return definitions, nil
		}

		// Try SymbolInformation format (flat list)
		var symbolInfos []protocol.SymbolInformation
		if err := json.Unmarshal(resultBytes, &symbolInfos); err == nil && len(symbolInfos) > 0 {
			for _, symbol := range symbolInfos {
				definitions = append(definitions, t.convertSymbolInformation(symbol))
			}
			return definitions, nil
		}

		// If both fail, try to handle as a single symbol (some servers return single objects)
		var singleDocSymbol protocol.DocumentSymbol
		if err := json.Unmarshal(resultBytes, &singleDocSymbol); err == nil && singleDocSymbol.Name != "" {
			definitions = append(definitions, t.convertDocumentSymbol(singleDocSymbol))
			return definitions, nil
		}

		var singleSymbolInfo protocol.SymbolInformation
		if err := json.Unmarshal(resultBytes, &singleSymbolInfo); err == nil && singleSymbolInfo.Name != "" {
			definitions = append(definitions, t.convertSymbolInformation(singleSymbolInfo))
			return definitions, nil
		}
	}

	return definitions, nil
}

// convertDocumentSymbol converts a DocumentSymbol to our format
func (t *DefinitionsTool) convertDocumentSymbol(symbol protocol.DocumentSymbol) DefinitionResult {
	result := DefinitionResult{
		Name:   symbol.Name,
		Kind:   t.symbolKindToString(symbol.Kind),
		Detail: symbol.Detail,
		Range:  t.formatRange(symbol.Range),
	}

	// Convert children recursively
	for _, child := range symbol.Children {
		result.Children = append(result.Children, t.convertDocumentSymbol(child))
	}

	return result
}

// convertSymbolInformation converts a SymbolInformation to our format
func (t *DefinitionsTool) convertSymbolInformation(symbol protocol.SymbolInformation) DefinitionResult {
	return DefinitionResult{
		Name:  symbol.Name,
		Kind:  t.symbolKindToString(symbol.Kind),
		Range: t.formatLocation(symbol.Location),
	}
}

// symbolKindToString converts SymbolKind to a readable string
func (t *DefinitionsTool) symbolKindToString(kind protocol.SymbolKind) string {
	switch kind {
	case protocol.File:
		return "File"
	case protocol.Module:
		return "Module"
	case protocol.Namespace:
		return "Namespace"
	case protocol.Package:
		return "Package"
	case protocol.Class:
		return "Class"
	case protocol.Method:
		return "Method"
	case protocol.Property:
		return "Property"
	case protocol.Field:
		return "Field"
	case protocol.Constructor:
		return "Constructor"
	case protocol.Enum:
		return "Enum"
	case protocol.Interface:
		return "Interface"
	case protocol.Function:
		return "Function"
	case protocol.Variable:
		return "Variable"
	case protocol.Constant:
		return "Constant"
	case protocol.String:
		return "String"
	case protocol.Number:
		return "Number"
	case protocol.Boolean:
		return "Boolean"
	case protocol.Array:
		return "Array"
	case protocol.Object:
		return "Object"
	case protocol.Key:
		return "Key"
	case protocol.Null:
		return "Null"
	case protocol.EnumMember:
		return "EnumMember"
	case protocol.Struct:
		return "Struct"
	case protocol.Event:
		return "Event"
	case protocol.Operator:
		return "Operator"
	case protocol.TypeParameter:
		return "TypeParameter"
	default:
		return fmt.Sprintf("Unknown(%d)", kind)
	}
}

// formatRange formats a Range to a readable string
func (t *DefinitionsTool) formatRange(r protocol.Range) string {
	startLine := r.Start.Line + 1
	endLine := r.End.Line + 1
	
	if startLine == endLine {
		return fmt.Sprintf("line %d", startLine)
	}
	return fmt.Sprintf("lines %d-%d", startLine, endLine)
}

// formatLocation formats a Location to a readable string
func (t *DefinitionsTool) formatLocation(loc protocol.Location) string {
	startLine := loc.Range.Start.Line + 1
	endLine := loc.Range.End.Line + 1
	
	if startLine == endLine {
		return fmt.Sprintf("line %d", startLine)
	}
	return fmt.Sprintf("lines %d-%d", startLine, endLine)
}

// formatDefinitions formats the definitions into a readable output
func (t *DefinitionsTool) formatDefinitions(definitions []DefinitionResult, filePath, clientName string) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("# Definitions in %s\n", filePath))
	output.WriteString(fmt.Sprintf("*Using %s LSP server*\n\n", clientName))

	if len(definitions) == 0 {
		output.WriteString("No definitions found in this file.\n")
		return output.String()
	}

	// Group definitions by kind for better organization
	kindGroups := make(map[string][]DefinitionResult)
	for _, def := range definitions {
		kindGroups[def.Kind] = append(kindGroups[def.Kind], def)
	}

	// Define order of kinds for consistent output
	kindOrder := []string{"Class", "Interface", "Struct", "Enum", "Function", "Method", "Variable", "Constant", "Property", "Field"}

	// Write definitions grouped by kind
	for _, kind := range kindOrder {
		if defs, exists := kindGroups[kind]; exists {
			output.WriteString(fmt.Sprintf("## %ss\n", kind))
			for _, def := range defs {
				t.writeDefinition(&output, def, 0)
			}
			output.WriteString("\n")
			delete(kindGroups, kind)
		}
	}

	// Write any remaining kinds not in the predefined order
	for kind, defs := range kindGroups {
		output.WriteString(fmt.Sprintf("## %ss\n", kind))
		for _, def := range defs {
			t.writeDefinition(&output, def, 0)
		}
		output.WriteString("\n")
	}

	return output.String()
}

// writeDefinition recursively writes a definition and its children
func (t *DefinitionsTool) writeDefinition(output *strings.Builder, def DefinitionResult, indent int) {
	indentStr := strings.Repeat("  ", indent)

	// Format the main definition line
	output.WriteString(fmt.Sprintf("%s- **%s** (%s)", indentStr, def.Name, def.Range))

	if def.Detail != "" {
		output.WriteString(fmt.Sprintf(" - %s", def.Detail))
	}
	output.WriteString("\n")

	// Write children with increased indentation
	if len(def.Children) > 0 {
		for _, child := range def.Children {
			t.writeDefinition(output, child, indent+1)
		}
	}
}

