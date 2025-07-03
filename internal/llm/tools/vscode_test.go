package tools

import (
	"context"
	"testing"

	"github.com/charmbracelet/crush/internal/permission"
)

func TestVSCodeDiffTool(t *testing.T) {
	// Create a real permission service for testing
	permissions := permission.NewPermissionService()
	
	tool := NewVSCodeDiffTool(permissions)
	
	// Test tool info
	info := tool.Info()
	if info.Name != VSCodeDiffToolName {
		t.Errorf("Expected tool name %s, got %s", VSCodeDiffToolName, info.Name)
	}
	
	// Test tool name
	if tool.Name() != VSCodeDiffToolName {
		t.Errorf("Expected tool name %s, got %s", VSCodeDiffToolName, tool.Name())
	}
	
	// Test parameter validation
	params := `{
		"left_content": "Hello World",
		"right_content": "Hello Universe",
		"left_title": "before.txt",
		"right_title": "after.txt",
		"language": "text"
	}`
	
	call := ToolCall{
		ID:    "test-id",
		Name:  VSCodeDiffToolName,
		Input: params,
	}
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	// Auto-approve the session to avoid permission prompts during testing
	permissions.AutoApproveSession("test-session")
	
	// This will fail if VS Code is not installed, but should not error on parameter parsing
	response, err := tool.Run(ctx, call)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Should either succeed (if VS Code is available) or fail with a specific error message
	if response.IsError && response.Content != "VS Code is not available. Please install VS Code and ensure 'code' command is in PATH." {
		t.Errorf("Unexpected error response: %s", response.Content)
	}
}