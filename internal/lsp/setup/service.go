package setup

import (
	"context"

	"github.com/opencode-ai/opencode/internal/lsp/protocol"
	"github.com/opencode-ai/opencode/internal/pubsub"
)

// LSPSetupEvent represents an event related to LSP setup
type LSPSetupEvent struct {
	Type        LSPSetupEventType
	Language    protocol.LanguageKind
	ServerName  string
	Success     bool
	Error       error
	Description string
}

// LSPSetupEventType defines the type of LSP setup event
type LSPSetupEventType string

const (
	// EventLanguageDetected is emitted when a language is detected in the workspace
	EventLanguageDetected LSPSetupEventType = "language_detected"
	// EventServerDiscovered is emitted when an LSP server is discovered
	EventServerDiscovered LSPSetupEventType = "server_discovered"
	// EventServerInstalled is emitted when an LSP server is installed
	EventServerInstalled LSPSetupEventType = "server_installed"
	// EventServerInstallFailed is emitted when an LSP server installation fails
	EventServerInstallFailed LSPSetupEventType = "server_install_failed"
	// EventSetupCompleted is emitted when the LSP setup is completed
	EventSetupCompleted LSPSetupEventType = "setup_completed"
)

// Service defines the interface for the LSP setup service
type Service interface {
	pubsub.Suscriber[LSPSetupEvent]
	
	// DetectLanguages detects languages in the workspace
	DetectLanguages(ctx context.Context, workspaceDir string) (map[protocol.LanguageKind]int, error)
	
	// GetPrimaryLanguages returns the top N languages in the project
	GetPrimaryLanguages(languages map[protocol.LanguageKind]int, limit int) []LanguageScore
	
	// DetectMonorepo checks if the workspace is a monorepo
	DetectMonorepo(ctx context.Context, workspaceDir string) (bool, []string)
	
	// DiscoverInstalledLSPs discovers installed LSP servers
	DiscoverInstalledLSPs(ctx context.Context) LSPServerMap
	
	// GetRecommendedLSPServers returns recommended LSP servers for languages
	GetRecommendedLSPServers(ctx context.Context, languages []LanguageScore) LSPServerMap
	
	// InstallLSPServer installs an LSP server
	InstallLSPServer(ctx context.Context, server LSPServerInfo) InstallationResult
	
	// VerifyInstallation verifies that an LSP server is correctly installed
	VerifyInstallation(ctx context.Context, serverName string) bool
	
	// SaveConfiguration saves the LSP configuration
	SaveConfiguration(ctx context.Context, servers map[protocol.LanguageKind]LSPServerInfo) error
}

type service struct {
	*pubsub.Broker[LSPSetupEvent]
}

// NewService creates a new LSP setup service
func NewService() Service {
	broker := pubsub.NewBroker[LSPSetupEvent]()
	return &service{
		Broker: broker,
	}
}

// DetectLanguages detects languages in the workspace
func (s *service) DetectLanguages(ctx context.Context, workspaceDir string) (map[protocol.LanguageKind]int, error) {
	languages, err := DetectProjectLanguages(workspaceDir)
	if err != nil {
		return nil, err
	}
	
	// Emit events for detected languages
	for lang, score := range languages {
		if lang != "" && score > 0 {
			s.Publish(pubsub.CreatedEvent, LSPSetupEvent{
				Type:        EventLanguageDetected,
				Language:    lang,
				Description: "Language detected in workspace",
			})
		}
	}
	
	return languages, nil
}

// GetPrimaryLanguages returns the top N languages in the project
func (s *service) GetPrimaryLanguages(languages map[protocol.LanguageKind]int, limit int) []LanguageScore {
	return GetPrimaryLanguages(languages, limit)
}

// DetectMonorepo checks if the workspace is a monorepo
func (s *service) DetectMonorepo(ctx context.Context, workspaceDir string) (bool, []string) {
	return DetectMonorepo(workspaceDir)
}

// DiscoverInstalledLSPs discovers installed LSP servers
func (s *service) DiscoverInstalledLSPs(ctx context.Context) LSPServerMap {
	servers := DiscoverInstalledLSPs()
	
	// Emit events for discovered servers
	for lang, serverList := range servers {
		for _, server := range serverList {
			s.Publish(pubsub.CreatedEvent, LSPSetupEvent{
				Type:        EventServerDiscovered,
				Language:    lang,
				ServerName:  server.Name,
				Description: "LSP server discovered",
			})
		}
	}
	
	return servers
}

// GetRecommendedLSPServers returns recommended LSP servers for languages
func (s *service) GetRecommendedLSPServers(ctx context.Context, languages []LanguageScore) LSPServerMap {
	return GetRecommendedLSPServers(languages)
}

// InstallLSPServer installs an LSP server
func (s *service) InstallLSPServer(ctx context.Context, server LSPServerInfo) InstallationResult {
	result := InstallLSPServer(ctx, server)
	
	// Emit event based on installation result
	eventType := EventServerInstalled
	if !result.Success {
		eventType = EventServerInstallFailed
	}
	
	s.Publish(pubsub.CreatedEvent, LSPSetupEvent{
		Type:        eventType,
		ServerName:  server.Name,
		Success:     result.Success,
		Error:       result.Error,
		Description: result.Output,
	})
	
	return result
}

// VerifyInstallation verifies that an LSP server is correctly installed
func (s *service) VerifyInstallation(ctx context.Context, serverName string) bool {
	return VerifyInstallation(serverName)
}

// SaveConfiguration saves the LSP configuration
func (s *service) SaveConfiguration(ctx context.Context, servers map[protocol.LanguageKind]LSPServerInfo) error {
	// Update the LSP configuration
	err := UpdateLSPConfig(servers)
	
	// Emit setup completed event
	s.Publish(pubsub.CreatedEvent, LSPSetupEvent{
		Type:        EventSetupCompleted,
		Success:     err == nil,
		Error:       err,
		Description: "LSP setup completed",
	})
	
	return err
}