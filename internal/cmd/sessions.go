package cmd

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// SessionWithChildren represents a session with its nested children
type SessionWithChildren struct {
	session.Session
	Children []SessionWithChildren `json:"children,omitempty" yaml:"children,omitempty"`
}

// ImportSession represents a session with proper JSON tags for import
type ImportSession struct {
	ID               string          `json:"id"`
	ParentSessionID  string          `json:"parent_session_id"`
	Title            string          `json:"title"`
	MessageCount     int64           `json:"message_count"`
	PromptTokens     int64           `json:"prompt_tokens"`
	CompletionTokens int64           `json:"completion_tokens"`
	Cost             float64         `json:"cost"`
	CreatedAt        int64           `json:"created_at"`
	UpdatedAt        int64           `json:"updated_at"`
	SummaryMessageID string          `json:"summary_message_id,omitempty"`
	Children         []ImportSession `json:"children,omitempty"`
}

// ImportData represents the full import structure for sessions
type ImportData struct {
	Version       string          `json:"version" yaml:"version"`
	ExportedAt    string          `json:"exported_at,omitempty" yaml:"exported_at,omitempty"`
	TotalSessions int             `json:"total_sessions,omitempty" yaml:"total_sessions,omitempty"`
	Sessions      []ImportSession `json:"sessions" yaml:"sessions"`
}

// ImportMessage represents a message with proper JSON tags for import
type ImportMessage struct {
	ID        string        `json:"id"`
	Role      string        `json:"role"`
	SessionID string        `json:"session_id"`
	Parts     []interface{} `json:"parts"`
	Model     string        `json:"model,omitempty"`
	Provider  string        `json:"provider,omitempty"`
	CreatedAt int64         `json:"created_at"`
}

// ImportSessionInfo represents session info with proper JSON tags for conversation import
type ImportSessionInfo struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	MessageCount     int64   `json:"message_count"`
	PromptTokens     int64   `json:"prompt_tokens,omitempty"`
	CompletionTokens int64   `json:"completion_tokens,omitempty"`
	Cost             float64 `json:"cost,omitempty"`
	CreatedAt        int64   `json:"created_at"`
}

// ConversationData represents a single conversation import structure
type ConversationData struct {
	Version  string            `json:"version" yaml:"version"`
	Session  ImportSessionInfo `json:"session" yaml:"session"`
	Messages []ImportMessage   `json:"messages" yaml:"messages"`
}

// ImportResult contains the results of an import operation
type ImportResult struct {
	TotalSessions    int               `json:"total_sessions"`
	ImportedSessions int               `json:"imported_sessions"`
	SkippedSessions  int               `json:"skipped_sessions"`
	ImportedMessages int               `json:"imported_messages"`
	Errors           []string          `json:"errors,omitempty"`
	SessionMapping   map[string]string `json:"session_mapping"` // old_id -> new_id
}

// SessionStats represents aggregated session statistics
type SessionStats struct {
	TotalSessions         int64   `json:"total_sessions"`
	TotalMessages         int64   `json:"total_messages"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	TotalCost             float64 `json:"total_cost"`
	AvgCostPerSession     float64 `json:"avg_cost_per_session"`
}

// GroupedSessionStats represents statistics grouped by time period
type GroupedSessionStats struct {
	Period           string  `json:"period"`
	SessionCount     int64   `json:"session_count"`
	MessageCount     int64   `json:"message_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalCost        float64 `json:"total_cost"`
	AvgCost          float64 `json:"avg_cost"`
}

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Manage sessions",
	Long:  `List and export sessions and their nested subsessions`,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List sessions",
	Long:  `List all sessions in a hierarchical format`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		return runSessionsList(cmd.Context(), format)
	},
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export sessions",
	Long:  `Export all sessions and their nested subsessions to different formats`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		return runSessionsExport(cmd.Context(), format)
	},
}

var exportConversationCmd = &cobra.Command{
	Use:   "export-conversation <session-id>",
	Short: "Export a single conversation",
	Long:  `Export a single session with all its messages as markdown for sharing`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		format, _ := cmd.Flags().GetString("format")
		return runExportConversation(cmd.Context(), sessionID, format)
	},
}

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import sessions from a file",
	Long:  `Import sessions from a JSON or YAML file with hierarchical structure`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		format, _ := cmd.Flags().GetString("format")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		return runImport(cmd.Context(), file, format, dryRun)
	},
}

var importConversationCmd = &cobra.Command{
	Use:   "import-conversation <file>",
	Short: "Import a single conversation from a file",
	Long:  `Import a single conversation with messages from a JSON, YAML, or Markdown file`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		format, _ := cmd.Flags().GetString("format")
		return runImportConversation(cmd.Context(), file, format)
	},
}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search sessions by title or message content",
	Long:  `Search sessions by title pattern (case-insensitive) or message text content`,
	RunE: func(cmd *cobra.Command, args []string) error {
		titlePattern, _ := cmd.Flags().GetString("title")
		textPattern, _ := cmd.Flags().GetString("text")
		format, _ := cmd.Flags().GetString("format")

		if titlePattern == "" && textPattern == "" {
			return fmt.Errorf("at least one of --title or --text must be provided")
		}

		return runSessionsSearch(cmd.Context(), titlePattern, textPattern, format)
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show session statistics",
	Long:  `Display aggregated statistics about sessions including total counts, tokens, and costs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		groupBy, _ := cmd.Flags().GetString("group-by")
		return runSessionsStats(cmd.Context(), format, groupBy)
	},
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
	sessionsCmd.AddCommand(listCmd)
	sessionsCmd.AddCommand(exportCmd)
	sessionsCmd.AddCommand(exportConversationCmd)
	sessionsCmd.AddCommand(importCmd)
	sessionsCmd.AddCommand(importConversationCmd)
	sessionsCmd.AddCommand(searchCmd)
	sessionsCmd.AddCommand(statsCmd)

	listCmd.Flags().StringP("format", "f", "text", "Output format (text, json, yaml, markdown)")
	exportCmd.Flags().StringP("format", "f", "json", "Export format (json, yaml, markdown)")
	exportConversationCmd.Flags().StringP("format", "f", "markdown", "Export format (markdown, json, yaml)")
	importCmd.Flags().StringP("format", "f", "", "Import format (json, yaml) - auto-detected if not specified")
	importCmd.Flags().Bool("dry-run", false, "Validate import data without persisting changes")
	importConversationCmd.Flags().StringP("format", "f", "", "Import format (json, yaml, markdown) - auto-detected if not specified")
	searchCmd.Flags().String("title", "", "Search by session title pattern (case-insensitive substring search)")
	searchCmd.Flags().String("text", "", "Search by message text content")
	searchCmd.Flags().StringP("format", "f", "text", "Output format (text, json)")
	statsCmd.Flags().StringP("format", "f", "text", "Output format (text, json)")
	statsCmd.Flags().String("group-by", "", "Group statistics by time period (day, week, month)")
}

func runSessionsList(ctx context.Context, format string) error {
	sessionService, err := createSessionService(ctx)
	if err != nil {
		return err
	}

	sessions, err := buildSessionTree(ctx, sessionService)
	if err != nil {
		return err
	}

	return formatOutput(sessions, format, false)
}

func runSessionsExport(ctx context.Context, format string) error {
	sessionService, err := createSessionService(ctx)
	if err != nil {
		return err
	}

	sessions, err := buildSessionTree(ctx, sessionService)
	if err != nil {
		return err
	}

	return formatOutput(sessions, format, true)
}

func runExportConversation(ctx context.Context, sessionID, format string) error {
	sessionService, messageService, err := createServices(ctx)
	if err != nil {
		return err
	}

	// Get the session
	sess, err := sessionService.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session %s: %w", sessionID, err)
	}

	// Get all messages for the session
	messages, err := messageService.List(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get messages for session %s: %w", sessionID, err)
	}

	return formatConversation(sess, messages, format)
}

func createSessionService(ctx context.Context) (session.Service, error) {
	cwd, err := getCwd()
	if err != nil {
		return nil, err
	}

	cfg, err := config.Init(cwd, false)
	if err != nil {
		return nil, err
	}

	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		return nil, err
	}

	queries := db.New(conn)
	return session.NewService(queries), nil
}

func createServices(ctx context.Context) (session.Service, message.Service, error) {
	cwd, err := getCwd()
	if err != nil {
		return nil, nil, err
	}

	cfg, err := config.Init(cwd, false)
	if err != nil {
		return nil, nil, err
	}

	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		return nil, nil, err
	}

	queries := db.New(conn)
	sessionService := session.NewService(queries)
	messageService := message.NewService(queries)
	return sessionService, messageService, nil
}

func getCwd() (string, error) {
	// This could be enhanced to use the same logic as root.go
	cwd, err := getCwdFromFlags()
	if err != nil {
		return "", err
	}
	return cwd, nil
}

func getCwdFromFlags() (string, error) {
	return os.Getwd()
}

func buildSessionTree(ctx context.Context, sessionService session.Service) ([]SessionWithChildren, error) {
	// Get all top-level sessions (no parent)
	topLevelSessions, err := sessionService.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var result []SessionWithChildren
	for _, sess := range topLevelSessions {
		sessionWithChildren, err := buildSessionWithChildren(ctx, sessionService, sess)
		if err != nil {
			return nil, err
		}
		result = append(result, sessionWithChildren)
	}

	return result, nil
}

func buildSessionWithChildren(ctx context.Context, sessionService session.Service, sess session.Session) (SessionWithChildren, error) {
	children, err := sessionService.ListChildren(ctx, sess.ID)
	if err != nil {
		return SessionWithChildren{}, fmt.Errorf("failed to list children for session %s: %w", sess.ID, err)
	}

	var childrenWithChildren []SessionWithChildren
	for _, child := range children {
		childWithChildren, err := buildSessionWithChildren(ctx, sessionService, child)
		if err != nil {
			return SessionWithChildren{}, err
		}
		childrenWithChildren = append(childrenWithChildren, childWithChildren)
	}

	return SessionWithChildren{
		Session:  sess,
		Children: childrenWithChildren,
	}, nil
}

func formatOutput(sessions []SessionWithChildren, format string, includeMetadata bool) error {
	switch strings.ToLower(format) {
	case "json":
		return formatJSON(sessions)
	case "yaml":
		return formatYAML(sessions)
	case "markdown", "md":
		return formatMarkdown(sessions, includeMetadata)
	case "text":
		return formatText(sessions)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func formatJSON(sessions []SessionWithChildren) error {
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func formatYAML(sessions []SessionWithChildren) error {
	data, err := yaml.Marshal(sessions)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func formatMarkdown(sessions []SessionWithChildren, includeMetadata bool) error {
	fmt.Println("# Sessions")
	fmt.Println()

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	for _, sess := range sessions {
		printSessionMarkdown(sess, 0, includeMetadata)
	}

	return nil
}

func formatText(sessions []SessionWithChildren) error {
	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	for _, sess := range sessions {
		printSessionText(sess, 0)
	}

	return nil
}

func printSessionMarkdown(sess SessionWithChildren, level int, includeMetadata bool) {
	indent := strings.Repeat("#", level+2)
	fmt.Printf("%s %s\n", indent, sess.Title)
	fmt.Println()

	if includeMetadata {
		fmt.Printf("- **ID**: %s\n", sess.ID)
		if sess.ParentSessionID != "" {
			fmt.Printf("- **Parent**: %s\n", sess.ParentSessionID)
		}
		fmt.Printf("- **Messages**: %d\n", sess.MessageCount)
		fmt.Printf("- **Tokens**: %d prompt, %d completion\n", sess.PromptTokens, sess.CompletionTokens)
		fmt.Printf("- **Cost**: $%.4f\n", sess.Cost)
		fmt.Printf("- **Created**: %s\n", formatTimestamp(sess.CreatedAt))
		fmt.Printf("- **Updated**: %s\n", formatTimestamp(sess.UpdatedAt))
		fmt.Println()
	}

	for _, child := range sess.Children {
		printSessionMarkdown(child, level+1, includeMetadata)
	}
}

func printSessionText(sess SessionWithChildren, level int) {
	indent := strings.Repeat("  ", level)
	fmt.Printf("%s• %s (ID: %s, Messages: %d, Cost: $%.4f)\n",
		indent, sess.Title, sess.ID, sess.MessageCount, sess.Cost)

	for _, child := range sess.Children {
		printSessionText(child, level+1)
	}
}

func formatTimestamp(timestamp int64) string {
	// Assuming timestamp is Unix seconds
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}

func formatConversation(sess session.Session, messages []message.Message, format string) error {
	switch strings.ToLower(format) {
	case "markdown", "md":
		return formatConversationMarkdown(sess, messages)
	case "json":
		return formatConversationJSON(sess, messages)
	case "yaml":
		return formatConversationYAML(sess, messages)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func formatConversationMarkdown(sess session.Session, messages []message.Message) error {
	fmt.Printf("# %s\n\n", sess.Title)

	// Session metadata
	fmt.Printf("**Session ID:** %s  \n", sess.ID)
	fmt.Printf("**Created:** %s  \n", formatTimestamp(sess.CreatedAt))
	fmt.Printf("**Messages:** %d  \n", sess.MessageCount)
	fmt.Printf("**Tokens:** %d prompt, %d completion  \n", sess.PromptTokens, sess.CompletionTokens)
	if sess.Cost > 0 {
		fmt.Printf("**Cost:** $%.4f  \n", sess.Cost)
	}
	fmt.Println()
	fmt.Println("---")
	fmt.Println()

	for i, msg := range messages {
		formatMessageMarkdown(msg, i+1)
	}

	return nil
}

func formatMessageMarkdown(msg message.Message, index int) {
	// Role header
	switch msg.Role {
	case message.User:
		fmt.Printf("## 👤 User\n\n")
	case message.Assistant:
		fmt.Printf("## 🤖 Assistant")
		if msg.Model != "" {
			fmt.Printf(" (%s)", msg.Model)
		}
		fmt.Printf("\n\n")
	case message.System:
		fmt.Printf("## ⚙️ System\n\n")
	case message.Tool:
		fmt.Printf("## 🔧 Tool\n\n")
	}

	// Process each part
	for _, part := range msg.Parts {
		switch p := part.(type) {
		case message.TextContent:
			fmt.Printf("%s\n\n", p.Text)
		case message.ReasoningContent:
			if p.Thinking != "" {
				fmt.Printf("### 🧠 Reasoning\n\n")
				fmt.Printf("```\n%s\n```\n\n", p.Thinking)
			}
		case message.ToolCall:
			fmt.Printf("### 🔧 Tool Call: %s\n\n", p.Name)
			fmt.Printf("**ID:** %s  \n", p.ID)
			if p.Input != "" {
				fmt.Printf("**Input:**\n```json\n%s\n```\n\n", p.Input)
			}
		case message.ToolResult:
			fmt.Printf("### 📝 Tool Result: %s\n\n", p.Name)
			if p.IsError {
				fmt.Printf("**❌ Error:**\n```\n%s\n```\n\n", p.Content)
			} else {
				fmt.Printf("**✅ Result:**\n```\n%s\n```\n\n", p.Content)
			}
		case message.ImageURLContent:
			fmt.Printf("![Image](%s)\n\n", p.URL)
		case message.BinaryContent:
			fmt.Printf("**File:** %s (%s)\n\n", p.Path, p.MIMEType)
		case message.Finish:
			if p.Reason != message.FinishReasonEndTurn {
				fmt.Printf("**Finish Reason:** %s\n", p.Reason)
				if p.Message != "" {
					fmt.Printf("**Message:** %s\n", p.Message)
				}
				fmt.Println()
			}
		}
	}

	fmt.Println("---")
	fmt.Println()
}

func formatConversationJSON(sess session.Session, messages []message.Message) error {
	data := struct {
		Session  session.Session   `json:"session"`
		Messages []message.Message `json:"messages"`
	}{
		Session:  sess,
		Messages: messages,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func formatConversationYAML(sess session.Session, messages []message.Message) error {
	data := struct {
		Session  session.Session   `yaml:"session"`
		Messages []message.Message `yaml:"messages"`
	}{
		Session:  sess,
		Messages: messages,
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Println(string(yamlData))
	return nil
}

func runImport(ctx context.Context, file, format string, dryRun bool) error {
	// Read the file
	data, err := readImportFile(file, format)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Validate the data structure
	if err := validateImportData(data); err != nil {
		return fmt.Errorf("invalid import data: %w", err)
	}

	if dryRun {
		result := ImportResult{
			TotalSessions:    countTotalImportSessions(data.Sessions),
			ImportedSessions: 0,
			SkippedSessions:  0,
			ImportedMessages: 0,
			SessionMapping:   make(map[string]string),
		}
		fmt.Printf("Dry run: Would import %d sessions\n", result.TotalSessions)
		return nil
	}

	// Perform the actual import
	sessionService, messageService, err := createServices(ctx)
	if err != nil {
		return err
	}

	result, err := importSessions(ctx, sessionService, messageService, data)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Print summary
	fmt.Printf("Import completed successfully:\n")
	fmt.Printf("  Total sessions processed: %d\n", result.TotalSessions)
	fmt.Printf("  Sessions imported: %d\n", result.ImportedSessions)
	fmt.Printf("  Sessions skipped: %d\n", result.SkippedSessions)
	fmt.Printf("  Messages imported: %d\n", result.ImportedMessages)

	if len(result.Errors) > 0 {
		fmt.Printf("  Errors encountered: %d\n", len(result.Errors))
		for _, errStr := range result.Errors {
			fmt.Printf("    - %s\n", errStr)
		}
	}

	return nil
}

func runImportConversation(ctx context.Context, file, format string) error {
	// Read the conversation file
	convData, err := readConversationFile(file, format)
	if err != nil {
		return fmt.Errorf("failed to read conversation file: %w", err)
	}

	// Validate the conversation data
	if err := validateConversationData(convData); err != nil {
		return fmt.Errorf("invalid conversation data: %w", err)
	}

	// Import the conversation
	sessionService, messageService, err := createServices(ctx)
	if err != nil {
		return err
	}

	newSessionID, messageCount, err := importConversation(ctx, sessionService, messageService, convData)
	if err != nil {
		return fmt.Errorf("conversation import failed: %w", err)
	}

	fmt.Printf("Conversation imported successfully:\n")
	fmt.Printf("  Session ID: %s\n", newSessionID)
	fmt.Printf("  Title: %s\n", convData.Session.Title)
	fmt.Printf("  Messages imported: %d\n", messageCount)

	return nil
}

func readImportFile(file, format string) (*ImportData, error) {
	fileData, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", file, err)
	}

	// Auto-detect format if not specified
	if format == "" {
		format = detectFormat(file, fileData)
	}

	var data ImportData
	switch strings.ToLower(format) {
	case "json":
		if err := json.Unmarshal(fileData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case "yaml", "yml":
		if err := yaml.Unmarshal(fileData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return &data, nil
}

func readConversationFile(file, format string) (*ConversationData, error) {
	fileData, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", file, err)
	}

	// Auto-detect format if not specified
	if format == "" {
		format = detectFormat(file, fileData)
	}

	var data ConversationData
	switch strings.ToLower(format) {
	case "json":
		if err := json.Unmarshal(fileData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case "yaml", "yml":
		if err := yaml.Unmarshal(fileData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case "markdown", "md":
		return nil, fmt.Errorf("markdown import for conversations is not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return &data, nil
}

func detectFormat(filename string, data []byte) string {
	// First try file extension
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md", ".markdown":
		return "markdown"
	}

	// Try to detect from content
	data = bytes.TrimSpace(data)
	if len(data) > 0 {
		if data[0] == '{' || data[0] == '[' {
			return "json"
		}
		if strings.HasPrefix(string(data), "---") || strings.Contains(string(data), ":") {
			return "yaml"
		}
	}

	return "json" // default fallback
}

func validateImportData(data *ImportData) error {
	if data == nil {
		return fmt.Errorf("import data is nil")
	}

	if len(data.Sessions) == 0 {
		return fmt.Errorf("no sessions to import")
	}

	// Validate session structure
	for i, sess := range data.Sessions {
		if err := validateImportSessionHierarchy(sess, ""); err != nil {
			return fmt.Errorf("session %d validation failed: %w", i, err)
		}
	}

	return nil
}

func validateConversationData(data *ConversationData) error {
	if data == nil {
		return fmt.Errorf("conversation data is nil")
	}

	if data.Session.Title == "" {
		return fmt.Errorf("session title is required")
	}

	if len(data.Messages) == 0 {
		return fmt.Errorf("no messages to import")
	}

	return nil
}

func validateImportSessionHierarchy(sess ImportSession, expectedParent string) error {
	if sess.ID == "" {
		return fmt.Errorf("session ID is required")
	}

	if sess.Title == "" {
		return fmt.Errorf("session title is required")
	}

	// For top-level sessions, expectedParent should be empty and session should have no parent or empty parent
	if expectedParent == "" {
		if sess.ParentSessionID != "" {
			return fmt.Errorf("top-level session should not have a parent, got %s", sess.ParentSessionID)
		}
	} else {
		// For child sessions, parent should match expected parent
		if sess.ParentSessionID != expectedParent {
			return fmt.Errorf("parent session ID mismatch: expected %s, got %s (session ID: %s)", expectedParent, sess.ParentSessionID, sess.ID)
		}
	}

	// Validate children
	for _, child := range sess.Children {
		if err := validateImportSessionHierarchy(child, sess.ID); err != nil {
			return err
		}
	}

	return nil
}

func validateSessionHierarchy(sess SessionWithChildren, expectedParent string) error {
	if sess.ID == "" {
		return fmt.Errorf("session ID is required")
	}

	if sess.Title == "" {
		return fmt.Errorf("session title is required")
	}

	// For top-level sessions, expectedParent should be empty and session should have no parent or empty parent
	if expectedParent == "" {
		if sess.ParentSessionID != "" {
			return fmt.Errorf("top-level session should not have a parent, got %s", sess.ParentSessionID)
		}
	} else {
		// For child sessions, parent should match expected parent
		if sess.ParentSessionID != expectedParent {
			return fmt.Errorf("parent session ID mismatch: expected %s, got %s (session ID: %s)", expectedParent, sess.ParentSessionID, sess.ID)
		}
	}

	// Validate children
	for _, child := range sess.Children {
		if err := validateSessionHierarchy(child, sess.ID); err != nil {
			return err
		}
	}

	return nil
}

func countTotalImportSessions(sessions []ImportSession) int {
	count := len(sessions)
	for _, sess := range sessions {
		count += countTotalImportSessions(sess.Children)
	}
	return count
}

func countTotalSessions(sessions []SessionWithChildren) int {
	count := len(sessions)
	for _, sess := range sessions {
		count += countTotalSessions(sess.Children)
	}
	return count
}

func importSessions(ctx context.Context, sessionService session.Service, messageService message.Service, data *ImportData) (ImportResult, error) {
	result := ImportResult{
		TotalSessions:  countTotalImportSessions(data.Sessions),
		SessionMapping: make(map[string]string),
	}

	// Import sessions recursively, starting with top-level sessions
	for _, sess := range data.Sessions {
		err := importImportSessionWithChildren(ctx, sessionService, messageService, sess, "", &result)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to import session %s: %v", sess.ID, err))
		}
	}

	return result, nil
}

func importConversation(ctx context.Context, sessionService session.Service, messageService message.Service, data *ConversationData) (string, int, error) {
	// Generate new session ID
	newSessionID := uuid.New().String()

	// Create the session using the low-level database API
	cwd, err := getCwd()
	if err != nil {
		return "", 0, err
	}

	cfg, err := config.Init(cwd, false)
	if err != nil {
		return "", 0, err
	}

	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		return "", 0, err
	}

	queries := db.New(conn)

	// Create session with all original metadata
	_, err = queries.CreateSession(ctx, db.CreateSessionParams{
		ID:               newSessionID,
		ParentSessionID:  sql.NullString{Valid: false},
		Title:            data.Session.Title,
		MessageCount:     data.Session.MessageCount,
		PromptTokens:     data.Session.PromptTokens,
		CompletionTokens: data.Session.CompletionTokens,
		Cost:             data.Session.Cost,
	})
	if err != nil {
		return "", 0, fmt.Errorf("failed to create session: %w", err)
	}

	// Import messages
	messageCount := 0
	for _, msg := range data.Messages {
		// Generate new message ID
		newMessageID := uuid.New().String()

		// Marshal message parts
		partsJSON, err := json.Marshal(msg.Parts)
		if err != nil {
			return "", 0, fmt.Errorf("failed to marshal message parts: %w", err)
		}

		// Create message
		_, err = queries.CreateMessage(ctx, db.CreateMessageParams{
			ID:        newMessageID,
			SessionID: newSessionID,
			Role:      string(msg.Role),
			Parts:     string(partsJSON),
			Model:     sql.NullString{String: msg.Model, Valid: msg.Model != ""},
			Provider:  sql.NullString{String: msg.Provider, Valid: msg.Provider != ""},
		})
		if err != nil {
			return "", 0, fmt.Errorf("failed to create message: %w", err)
		}
		messageCount++
	}

	return newSessionID, messageCount, nil
}

func importImportSessionWithChildren(ctx context.Context, sessionService session.Service, messageService message.Service, sess ImportSession, parentID string, result *ImportResult) error {
	// Generate new session ID
	newSessionID := uuid.New().String()
	result.SessionMapping[sess.ID] = newSessionID

	// Create the session using the low-level database API to preserve metadata
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, err := config.Init(cwd, false)
	if err != nil {
		return err
	}

	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		return err
	}

	queries := db.New(conn)

	// Create session with all original metadata
	parentSessionID := sql.NullString{Valid: false}
	if parentID != "" {
		parentSessionID = sql.NullString{String: parentID, Valid: true}
	}

	_, err = queries.CreateSession(ctx, db.CreateSessionParams{
		ID:               newSessionID,
		ParentSessionID:  parentSessionID,
		Title:            sess.Title,
		MessageCount:     sess.MessageCount,
		PromptTokens:     sess.PromptTokens,
		CompletionTokens: sess.CompletionTokens,
		Cost:             sess.Cost,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	result.ImportedSessions++

	// Import children recursively
	for _, child := range sess.Children {
		err := importImportSessionWithChildren(ctx, sessionService, messageService, child, newSessionID, result)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to import child session %s: %v", child.ID, err))
		}
	}

	return nil
}

func importSessionWithChildren(ctx context.Context, sessionService session.Service, messageService message.Service, sess SessionWithChildren, parentID string, result *ImportResult) error {
	// Generate new session ID
	newSessionID := uuid.New().String()
	result.SessionMapping[sess.ID] = newSessionID

	// Create the session using the low-level database API to preserve metadata
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, err := config.Init(cwd, false)
	if err != nil {
		return err
	}

	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		return err
	}

	queries := db.New(conn)

	// Create session with all original metadata
	parentSessionID := sql.NullString{Valid: false}
	if parentID != "" {
		parentSessionID = sql.NullString{String: parentID, Valid: true}
	}

	_, err = queries.CreateSession(ctx, db.CreateSessionParams{
		ID:               newSessionID,
		ParentSessionID:  parentSessionID,
		Title:            sess.Title,
		MessageCount:     sess.MessageCount,
		PromptTokens:     sess.PromptTokens,
		CompletionTokens: sess.CompletionTokens,
		Cost:             sess.Cost,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	result.ImportedSessions++

	// Import children recursively
	for _, child := range sess.Children {
		err := importSessionWithChildren(ctx, sessionService, messageService, child, newSessionID, result)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to import child session %s: %v", child.ID, err))
		}
	}

	return nil
}

func runSessionsSearch(ctx context.Context, titlePattern, textPattern, format string) error {
	sessionService, err := createSessionService(ctx)
	if err != nil {
		return err
	}

	var sessions []session.Session

	// Determine which search method to use based on provided patterns
	if titlePattern != "" && textPattern != "" {
		sessions, err = sessionService.SearchByTitleAndText(ctx, titlePattern, textPattern)
	} else if titlePattern != "" {
		sessions, err = sessionService.SearchByTitle(ctx, titlePattern)
	} else if textPattern != "" {
		sessions, err = sessionService.SearchByText(ctx, textPattern)
	}

	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	return formatSearchResults(sessions, format)
}

func formatSearchResults(sessions []session.Session, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return formatSearchResultsJSON(sessions)
	case "text":
		return formatSearchResultsText(sessions)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func formatSearchResultsJSON(sessions []session.Session) error {
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func formatSearchResultsText(sessions []session.Session) error {
	if len(sessions) == 0 {
		fmt.Println("No sessions found matching the search criteria.")
		return nil
	}

	fmt.Printf("Found %d session(s):\n\n", len(sessions))
	for _, sess := range sessions {
		fmt.Printf("• %s (ID: %s)\n", sess.Title, sess.ID)
		fmt.Printf("  Messages: %d, Cost: $%.4f\n", sess.MessageCount, sess.Cost)
		fmt.Printf("  Created: %s\n", formatTimestamp(sess.CreatedAt))
		if sess.ParentSessionID != "" {
			fmt.Printf("  Parent: %s\n", sess.ParentSessionID)
		}
		fmt.Println()
	}

	return nil
}

func runSessionsStats(ctx context.Context, format, groupBy string) error {
	// Get database connection
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, err := config.Init(cwd, false)
	if err != nil {
		return err
	}

	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		return err
	}

	queries := db.New(conn)

	// Handle grouped statistics
	if groupBy != "" {
		return runGroupedStats(ctx, queries, format, groupBy)
	}

	// Get overall statistics
	statsRow, err := queries.GetSessionStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session stats: %w", err)
	}

	// Convert to our struct, handling NULL values
	stats := SessionStats{
		TotalSessions:         statsRow.TotalSessions,
		TotalMessages:         convertNullFloat64ToInt64(statsRow.TotalMessages),
		TotalPromptTokens:     convertNullFloat64ToInt64(statsRow.TotalPromptTokens),
		TotalCompletionTokens: convertNullFloat64ToInt64(statsRow.TotalCompletionTokens),
		TotalCost:             convertNullFloat64(statsRow.TotalCost),
		AvgCostPerSession:     convertNullFloat64(statsRow.AvgCostPerSession),
	}

	return formatStats(stats, format)
}

func runGroupedStats(ctx context.Context, queries *db.Queries, format, groupBy string) error {
	var groupedStats []GroupedSessionStats

	switch strings.ToLower(groupBy) {
	case "day":
		rows, err := queries.GetSessionStatsByDay(ctx)
		if err != nil {
			return fmt.Errorf("failed to get daily stats: %w", err)
		}
		groupedStats = convertDayStatsRows(rows)
	case "week":
		rows, err := queries.GetSessionStatsByWeek(ctx)
		if err != nil {
			return fmt.Errorf("failed to get weekly stats: %w", err)
		}
		groupedStats = convertWeekStatsRows(rows)
	case "month":
		rows, err := queries.GetSessionStatsByMonth(ctx)
		if err != nil {
			return fmt.Errorf("failed to get monthly stats: %w", err)
		}
		groupedStats = convertMonthStatsRows(rows)
	default:
		return fmt.Errorf("unsupported group-by value: %s. Valid values are: day, week, month", groupBy)
	}

	return formatGroupedStats(groupedStats, format, groupBy)
}

func convertNullFloat64(val sql.NullFloat64) float64 {
	if val.Valid {
		return val.Float64
	}
	return 0.0
}

func convertNullFloat64ToInt64(val sql.NullFloat64) int64 {
	if val.Valid {
		return int64(val.Float64)
	}
	return 0
}

func convertDayStatsRows(rows []db.GetSessionStatsByDayRow) []GroupedSessionStats {
	result := make([]GroupedSessionStats, 0, len(rows))
	for _, row := range rows {
		stats := GroupedSessionStats{
			Period:           fmt.Sprintf("%v", row.Day),
			SessionCount:     row.SessionCount,
			MessageCount:     convertNullFloat64ToInt64(row.MessageCount),
			PromptTokens:     convertNullFloat64ToInt64(row.PromptTokens),
			CompletionTokens: convertNullFloat64ToInt64(row.CompletionTokens),
			TotalCost:        convertNullFloat64(row.TotalCost),
			AvgCost:          convertNullFloat64(row.AvgCost),
		}
		result = append(result, stats)
	}
	return result
}

func convertWeekStatsRows(rows []db.GetSessionStatsByWeekRow) []GroupedSessionStats {
	result := make([]GroupedSessionStats, 0, len(rows))
	for _, row := range rows {
		stats := GroupedSessionStats{
			Period:           fmt.Sprintf("%v", row.WeekStart),
			SessionCount:     row.SessionCount,
			MessageCount:     convertNullFloat64ToInt64(row.MessageCount),
			PromptTokens:     convertNullFloat64ToInt64(row.PromptTokens),
			CompletionTokens: convertNullFloat64ToInt64(row.CompletionTokens),
			TotalCost:        convertNullFloat64(row.TotalCost),
			AvgCost:          convertNullFloat64(row.AvgCost),
		}
		result = append(result, stats)
	}
	return result
}

func convertMonthStatsRows(rows []db.GetSessionStatsByMonthRow) []GroupedSessionStats {
	result := make([]GroupedSessionStats, 0, len(rows))
	for _, row := range rows {
		stats := GroupedSessionStats{
			Period:           fmt.Sprintf("%v", row.Month),
			SessionCount:     row.SessionCount,
			MessageCount:     convertNullFloat64ToInt64(row.MessageCount),
			PromptTokens:     convertNullFloat64ToInt64(row.PromptTokens),
			CompletionTokens: convertNullFloat64ToInt64(row.CompletionTokens),
			TotalCost:        convertNullFloat64(row.TotalCost),
			AvgCost:          convertNullFloat64(row.AvgCost),
		}
		result = append(result, stats)
	}
	return result
}

func formatStats(stats SessionStats, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return formatStatsJSON(stats)
	case "text":
		return formatStatsText(stats)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func formatGroupedStats(stats []GroupedSessionStats, format, groupBy string) error {
	switch strings.ToLower(format) {
	case "json":
		return formatGroupedStatsJSON(stats)
	case "text":
		return formatGroupedStatsText(stats, groupBy)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func formatStatsJSON(stats SessionStats) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func formatStatsText(stats SessionStats) error {
	if stats.TotalSessions == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Println("Session Statistics")
	fmt.Println("==================")
	fmt.Printf("Total Sessions:       %d\n", stats.TotalSessions)
	fmt.Printf("Total Messages:       %d\n", stats.TotalMessages)
	fmt.Printf("Total Prompt Tokens:  %d\n", stats.TotalPromptTokens)
	fmt.Printf("Total Completion Tokens: %d\n", stats.TotalCompletionTokens)
	fmt.Printf("Total Cost:           $%.4f\n", stats.TotalCost)
	fmt.Printf("Average Cost/Session: $%.4f\n", stats.AvgCostPerSession)

	totalTokens := stats.TotalPromptTokens + stats.TotalCompletionTokens
	if totalTokens > 0 {
		fmt.Printf("Total Tokens:         %d\n", totalTokens)
		fmt.Printf("Average Tokens/Session: %.1f\n", float64(totalTokens)/float64(stats.TotalSessions))
	}

	if stats.TotalSessions > 0 {
		fmt.Printf("Average Messages/Session: %.1f\n", float64(stats.TotalMessages)/float64(stats.TotalSessions))
	}

	return nil
}

func formatGroupedStatsJSON(stats []GroupedSessionStats) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func formatGroupedStatsText(stats []GroupedSessionStats, groupBy string) error {
	if len(stats) == 0 {
		fmt.Printf("No sessions found for grouping by %s.\n", groupBy)
		return nil
	}

	fmt.Printf("Session Statistics (Grouped by %s)\n", strings.ToUpper(groupBy[:1])+groupBy[1:])
	fmt.Println(strings.Repeat("=", 30+len(groupBy)))
	fmt.Println()

	for _, stat := range stats {
		fmt.Printf("Period: %s\n", stat.Period)
		fmt.Printf("  Sessions:       %d\n", stat.SessionCount)
		fmt.Printf("  Messages:       %d\n", stat.MessageCount)
		fmt.Printf("  Prompt Tokens:  %d\n", stat.PromptTokens)
		fmt.Printf("  Completion Tokens: %d\n", stat.CompletionTokens)
		fmt.Printf("  Total Cost:     $%.4f\n", stat.TotalCost)
		fmt.Printf("  Average Cost:   $%.4f\n", stat.AvgCost)
		totalTokens := stat.PromptTokens + stat.CompletionTokens
		if totalTokens > 0 {
			fmt.Printf("  Total Tokens:   %d\n", totalTokens)
		}
		fmt.Println()
	}

	return nil
}
