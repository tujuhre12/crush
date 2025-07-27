# RFC: Session Import and Export

## Summary

This RFC proposes a comprehensive system for importing and exporting conversation sessions in Crush.

## Background

Crush manages conversations through a hierarchical session system where:
- Sessions contain metadata (title, token counts, cost, timestamps) 
- Sessions can have parent-child relationships (nested conversations)
- Messages within sessions have structured content parts (text, tool calls, reasoning, etc.)
- The current implementation provides export functionality but lacks import capabilities

The latest commit introduced three key commands:
- `crush sessions list` - List sessions in various formats
- `crush sessions export` - Export all sessions and metadata  
- `crush sessions export-conversation <session-id>` - Export a single conversation with messages

## Motivation

Users need to:
1. Share conversations with others
2. Use conversation logs for debugging
3. Archive and analyze conversation history
4. Export data for external tools

## Detailed Design

### Core Data Model

The session export format builds on the existing session structure:

```go
type Session struct {
    ID               string  `json:"id"`
    ParentSessionID  string  `json:"parent_session_id,omitempty"`
    Title            string  `json:"title"`
    MessageCount     int64   `json:"message_count"`
    PromptTokens     int64   `json:"prompt_tokens"`
    CompletionTokens int64   `json:"completion_tokens"`
    Cost             float64 `json:"cost"`
    CreatedAt        int64   `json:"created_at"`
    UpdatedAt        int64   `json:"updated_at"`
    SummaryMessageID string  `json:"summary_message_id,omitempty"`
}

type SessionWithChildren struct {
    Session
    Children []SessionWithChildren `json:"children,omitempty"`
}
```

### Proposed Command Interface

#### Export Commands (Already Implemented)
```bash
# List sessions in various formats
crush sessions list [--format text|json|yaml|markdown]

# Export all sessions with metadata
crush sessions export [--format json|yaml|markdown]

# Export single conversation with full message history
crush sessions export-conversation <session-id> [--format markdown|json|yaml]
```

#### New Import Commands

```bash
# Import sessions from a file
crush sessions import <file> [--format json|yaml] [--dry-run] 

# Import a single conversation
crush sessions import-conversation <file> [--format json|yaml|markdown] 

```

#### Enhanced Inspection Commands

```bash
# Search sessions by criteria
crush sessions search [--title <pattern>] [--text <text>] [--format text|json]

# Show session statistics
crush sessions stats [--format text|json] [--group-by day|week|month]

# Show statistics for a single session
crush sessions stats <session-id> [--format text|json] 
```

### Import/Export Formats

#### Full Export Format (JSON)
```json
{
  "version": "1.0",
  "exported_at": "2025-01-27T10:30:00Z",
  "total_sessions": 15,
  "sessions": [
    {
      "id": "session-123",
      "parent_session_id": "",
      "title": "API Design Discussion",
      "message_count": 8,
      "prompt_tokens": 1250,
      "completion_tokens": 890,
      "cost": 0.0234,
      "created_at": 1706356200,
      "updated_at": 1706359800,
      "children": [
        {
          "id": "session-124",
          "parent_session_id": "session-123",
          "title": "Implementation Details",
          "message_count": 4,
          "prompt_tokens": 650,
          "completion_tokens": 420,
          "cost": 0.0145,
          "created_at": 1706359900,
          "updated_at": 1706361200
        }
      ]
    }
  ]
}
```

#### Conversation Export Format (JSON)
```json
{
  "version": "1.0",
  "session": {
    "id": "session-123",
    "title": "API Design Discussion",
    "created_at": 1706356200,
    "message_count": 3
  },
  "messages": [
    {
      "id": "msg-001",
      "session_id": "session-123", 
      "role": "user",
      "parts": [
        {
          "type": "text",
          "data": {
            "text": "Help me design a REST API for user management"
          }
        }
      ],
      "created_at": 1706356200
    },
    {
      "id": "msg-002",
      "session_id": "session-123",
      "role": "assistant",
      "model": "gpt-4",
      "provider": "openai",
      "parts": [
        {
          "type": "text", 
          "data": {
            "text": "I'll help you design a REST API for user management..."
          }
        },
        {
          "type": "finish",
          "data": {
            "reason": "stop",
            "time": 1706356230
          }
        }
      ],
      "created_at": 1706356220
    }
  ]
}
```

### API Implementation

#### Import Service Interface
```go
type ImportService interface {
    // Import sessions from structured data
    ImportSessions(ctx context.Context, data ImportData, opts ImportOptions) (ImportResult, error)
    
    // Import single conversation
    ImportConversation(ctx context.Context, data ConversationData, opts ImportOptions) (Session, error)
    
    // Validate import data without persisting
    ValidateImport(ctx context.Context, data ImportData) (ValidationResult, error)
}

type ImportOptions struct {
    ConflictStrategy ConflictStrategy // skip, merge, replace
    DryRun          bool
    ParentSessionID string // For conversation imports
    PreserveIDs     bool   // Whether to preserve original IDs
}

type ConflictStrategy string

const (
    ConflictSkip    ConflictStrategy = "skip"    // Skip existing sessions
    ConflictMerge   ConflictStrategy = "merge"   // Merge with existing
    ConflictReplace ConflictStrategy = "replace" // Replace existing
)

type ImportResult struct {
    TotalSessions    int               `json:"total_sessions"`
    ImportedSessions int               `json:"imported_sessions"`
    SkippedSessions  int               `json:"skipped_sessions"`
    Errors          []ImportError     `json:"errors,omitempty"`
    SessionMapping  map[string]string `json:"session_mapping"` // old_id -> new_id
}
```

#### Enhanced Export Service
```go
type ExportService interface {
    // Export sessions with filtering
    ExportSessions(ctx context.Context, opts ExportOptions) ([]SessionWithChildren, error)
    
    // Export conversation with full message history  
    ExportConversation(ctx context.Context, sessionID string, opts ExportOptions) (ConversationExport, error)
    
    // Search and filter sessions
    SearchSessions(ctx context.Context, criteria SearchCriteria) ([]Session, error)
    
    // Get session statistics
    GetStats(ctx context.Context, opts StatsOptions) (SessionStats, error)
}

type ExportOptions struct {
    Format          string    // json, yaml, markdown
    IncludeMessages bool      // Include full message content
    DateRange       DateRange // Filter by date range
    SessionIDs      []string  // Export specific sessions
}

type SearchCriteria struct {
    TitlePattern    string
    DateRange       DateRange
    MinCost         float64
    MaxCost         float64
    ParentSessionID string
    HasChildren     *bool
}
```

## Implementation Status

The proposed session import/export functionality has been implemented as a prototype as of July 2025.

### Implemented Commands

All new commands have been added to `internal/cmd/sessions.go`:

- **Import**: `crush sessions import <file> [--format json|yaml] [--dry-run]`
  - Supports hierarchical session imports with parent-child relationships
  - Generates new UUIDs to avoid conflicts
  - Includes validation and dry-run capabilities

- **Import Conversation**: `crush sessions import-conversation <file> [--format json|yaml]`
  - Imports single conversations with full message history
  - Preserves all message content parts and metadata

- **Search**: `crush sessions search [--title <pattern>] [--text <text>] [--format text|json]`
  - Case-insensitive title search and message content search
  - Supports combined search criteria with AND logic

- **Stats**: `crush sessions stats [--format text|json] [--group-by day|week|month]`
  - Comprehensive usage statistics (sessions, messages, tokens, costs)
  - Time-based grouping with efficient database queries

### Database Changes

Added new SQL queries in `internal/db/sql/sessions.sql`:
- Search queries for title and message content filtering
- Statistics aggregation queries with time-based grouping
- All queries optimized for performance with proper indexing

### Database Schema Considerations

The current schema supports the import/export functionality. Additional indexes may be needed for search performance:

```sql
-- Optimize session searches by date and cost
CREATE INDEX idx_sessions_created_at ON sessions(created_at);
CREATE INDEX idx_sessions_cost ON sessions(cost);
CREATE INDEX idx_sessions_title ON sessions(title COLLATE NOCASE);

-- Optimize message searches by session
CREATE INDEX idx_messages_session_created ON messages(session_id, created_at);
```
