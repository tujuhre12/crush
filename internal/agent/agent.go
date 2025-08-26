package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/agent/prompt"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/ai"
	"github.com/charmbracelet/crush/internal/ai/providers"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/history"
	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/charmbracelet/crush/internal/session"
)

// Common errors
var (
	ErrRequestCancelled = errors.New("request canceled by user")
	ErrSessionBusy      = errors.New("session is currently processing another request")
)

type AgentEventType string

const (
	AgentEventTypeError     AgentEventType = "error"
	AgentEventTypeResponse  AgentEventType = "response"
	AgentEventTypeSummarize AgentEventType = "summarize"
)

type AgentEvent struct {
	Type   AgentEventType
	Result ai.AgentResult
	Error  error

	// When summarizing
	SessionID string
	Progress  string
	Done      bool
}

type Service interface {
	pubsub.Suscriber[AgentEvent]
	Model() catwalk.Model
	Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error)
	Cancel(sessionID string)
	CancelAll()
	IsSessionBusy(sessionID string) bool
	IsBusy() bool
	Summarize(ctx context.Context, sessionID string) error
	UpdateModel() error
	QueuedPrompts(sessionID string) int
	ClearQueue(sessionID string)
}

type agent struct {
	*pubsub.Broker[AgentEvent]
	cfg            *config.Config
	permissions    permission.Service
	sessions       session.Service
	messages       message.Service
	history        history.Service
	lspClients     map[string]*lsp.Client
	activeRequests *csync.Map[string, context.CancelFunc]

	promptQueue *csync.Map[string, []string]
}

type AgentOption = func(*agent)

// WIP this is a work in progress
func NewAgent(
	cfg *config.Config,
	permissions permission.Service,
	sessions session.Service,
	messages message.Service,
	history history.Service,
	lspClients map[string]*lsp.Client,
) Service {
	return &agent{
		cfg:            cfg,
		Broker:         pubsub.NewBroker[AgentEvent](),
		permissions:    permissions,
		sessions:       sessions,
		messages:       messages,
		history:        history,
		lspClients:     lspClients,
		activeRequests: csync.NewMap[string, context.CancelFunc](),
		promptQueue:    csync.NewMap[string, []string](),
	}
}

func (a *agent) getLanguageModel(providerName, modelID string) (ai.LanguageModel, error) {
	var provider ai.Provider
	providerCfg, ok := a.cfg.Providers.Get(providerName)
	if !ok {
		return nil, errors.New("provider not found")
	}

	models := providerCfg.Models
	foundModel := false
	for _, providerModel := range models {
		if providerModel.ID == modelID {
			foundModel = true
			break
		}
	}
	if !foundModel {
		return nil, fmt.Errorf("model `%s` not found in provider `%s`", modelID, providerName)
	}
	switch providerName {
	case "openai":
		apiKey, err := a.cfg.Resolve(providerCfg.APIKey)
		if err != nil {
			return nil, err
		}
		baseURL, err := a.cfg.Resolve(providerCfg.BaseURL)
		if err != nil {
			return nil, err
		}
		opts := []providers.OpenAiOption{
			providers.WithOpenAiAPIKey(apiKey),
		}
		if baseURL != "" {
			opts = append(opts, providers.WithOpenAiBaseURL(baseURL))
		}
		provider = providers.NewOpenAiProvider(opts...)
	default:
		return nil, errors.New("provider not found")
	}
	if provider == nil {
		return nil, errors.New("provider not found")
	}
	return provider.LanguageModel(modelID)
}

func (a *agent) tools(ctx context.Context) []ai.AgentTool {
	cwd := a.cfg.WorkingDir()
	allTools := []ai.AgentTool{
		tools.NewBashTool(a.permissions, cwd),
		tools.NewDownloadTool(a.permissions, cwd),
		tools.NewEditTool(a.lspClients, a.permissions, a.history, cwd),
		tools.NewMultiEditTool(a.lspClients, a.permissions, a.history, cwd),
		tools.NewFetchTool(a.permissions, cwd),
		tools.NewGlobTool(cwd),
		tools.NewGrepTool(cwd),
		tools.NewLSTool(a.permissions, cwd),
		tools.NewSourcegraphTool(),
		tools.NewViewTool(a.lspClients, a.permissions, cwd),
		tools.NewWriteTool(a.lspClients, a.permissions, a.history, cwd),
	}
	mcpTools := tools.GetMCPTools(ctx, a.permissions, a.cfg)

	allTools = append(allTools, mcpTools...)

	if len(a.lspClients) > 0 {
		allTools = append(allTools, tools.NewDiagnosticsTool(a.lspClients))
	}
	// TODO: add agent tool
	return allTools
}

// Run implements Service.
func (a *agent) Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error) {
	// INFO: for now we assume that the agent uses the large model
	configModel := a.cfg.Models[config.SelectedModelTypeLarge]
	model, err := a.getLanguageModel(configModel.Provider, configModel.Model)
	if err != nil {
		return nil, err
	}

	modelCfg := a.Model()
	maxTokens := configModel.MaxTokens
	if maxTokens == 0 {
		maxTokens = modelCfg.DefaultMaxTokens
	}

	if !modelCfg.SupportsImages && attachments != nil {
		attachments = nil
	}

	agent := ai.NewAgent(
		model,
		ai.WithSystemPrompt(
			prompt.CoderPrompt(configModel.Provider, a.cfg.Options.ContextPaths...),
		),
		ai.WithTools(a.tools(ctx)...),
		ai.WithMaxOutputTokens(maxTokens),
	)

	events := make(chan AgentEvent, 1)
	if a.IsSessionBusy(sessionID) {
		existing, ok := a.promptQueue.Get(sessionID)
		if !ok {
			existing = []string{}
		}
		existing = append(existing, content)
		a.promptQueue.Set(sessionID, existing)
		return nil, nil
	}

	genCtx, cancel := context.WithCancel(ctx)
	a.activeRequests.Set(sessionID, cancel)

	go func() {
		slog.Debug("Request started", "sessionID", sessionID)

		result, err := a.makeCall(genCtx, agent, sessionID, content, attachments)
		a.activeRequests.Del(sessionID)
		cancel()
		if err != nil {
			slog.Error(err.Error())
			events <- AgentEvent{
				Type:  AgentEventTypeError,
				Error: err,
			}
		} else {
			result := AgentEvent{
				Type:   AgentEventTypeResponse,
				Result: *result,
			}
			a.Publish(pubsub.CreatedEvent, result)
			events <- result
		}
		slog.Debug("Request completed", "sessionID", sessionID)
		// TODO: implement this
		close(events)
	}()
	return events, nil
}

func (a *agent) makeCall(ctx context.Context, agent ai.Agent, sessionID, prompt string, attachments []message.Attachment) (*ai.AgentResult, error) {
	msgs, err := a.messages.List(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	if len(msgs) == 0 {
		go func() {
			// TODO: generate title
			// titleErr := a.generateTitle(context.Background(), sessionID, content)
			// if titleErr != nil && !errors.Is(titleErr, context.Canceled) && !errors.Is(titleErr, context.DeadlineExceeded) {
			// 	slog.Error("failed to generate title", "error", titleErr)
			// }
		}()
	}
	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session.SummaryMessageID != "" {
		summaryMsgInex := -1
		for i, msg := range msgs {
			if msg.ID == session.SummaryMessageID {
				summaryMsgInex = i
				break
			}
		}
		if summaryMsgInex != -1 {
			msgs = msgs[summaryMsgInex:]
			msgs[0].Role = message.User
		}
	}

	// Create the user message
	var attachmentParts []message.ContentPart
	for _, attachment := range attachments {
		attachmentParts = append(attachmentParts, message.BinaryContent{Path: attachment.FilePath, MIMEType: attachment.MimeType, Data: attachment.Content})
	}
	parts := []message.ContentPart{message.TextContent{Text: prompt}}
	parts = append(parts, attachmentParts...)
	_, err = a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: parts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user message: %w", err)
	}

	var history []ai.Message
	for _, m := range msgs {
		if len(m.Parts) == 0 {
			continue
		}
		// Assistant message without content or tool calls (cancelled before it returned anything)
		if m.Role == message.Assistant && len(m.ToolCalls()) == 0 && m.Content().Text == "" && m.ReasoningContent().String() == "" {
			continue
		}
		history = append(history, m.ToAIMessage()...)
	}

	var files []ai.FilePart
	for _, attachment := range attachments {
		files = append(files, ai.FilePart{
			Filename:  attachment.FileName,
			Data:      attachment.Content,
			MediaType: attachment.MimeType,
		})
	}
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)
	// TODO: see if this is even needed
	ctx = context.WithValue(ctx, tools.MessageIDContextKey, "mock")

	var currentAssistant *message.Message
	result, err := agent.Stream(ctx, ai.AgentStreamCall{
		Prompt:   prompt,
		Files:    files,
		Messages: history,
		// Get's called before each step
		PrepareStep: func(options ai.PrepareStepFunctionOptions) (ai.PrepareStepResult, error) {
			prepared := ai.PrepareStepResult{}
			modelCfg := a.cfg.Models[config.SelectedModelTypeLarge]
			// Before each step create the new assistant message
			assistantMsg, createErr := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
				Role:     message.Assistant,
				Parts:    []message.ContentPart{},
				Model:    modelCfg.Model,
				Provider: modelCfg.Provider,
			})
			if createErr != nil {
				return prepared, createErr
			}
			currentAssistant = &assistantMsg
			return prepared, nil
		},
		OnChunk: func(chunk ai.StreamPart) error {
			data, _ := json.Marshal(chunk)
			slog.Info("\n" + string(data) + "\n")
			return nil
		},
		// TODO: see how to not swallow the errors on these handlers
		OnReasoningDelta: func(id string, text string) error {
			currentAssistant.AppendReasoningContent(text)
			return a.messages.Update(ctx, *currentAssistant)
		},
		OnReasoningEnd: func(id string, reasoning ai.ReasoningContent) error {
			// handle anthropic signature
			if anthropicData, ok := reasoning.ProviderMetadata["anthropic"]; ok {
				if signature, ok := anthropicData["signature"]; ok {
					if str, ok := signature.(string); ok {
						currentAssistant.AppendReasoningContent(str)
						return a.messages.Update(ctx, *currentAssistant)
					}
				}
			}
			return nil
		},
		OnTextDelta: func(id string, text string) error {
			currentAssistant.FinishThinking()
			currentAssistant.AppendContent(text)
			return a.messages.Update(ctx, *currentAssistant)
		},
		OnToolInputStart: func(id string, toolName string) error {
			currentAssistant.FinishThinking()
			toolCall := message.ToolCall{
				ID:               id,
				Name:             toolName,
				ProviderExecuted: false,
				Finished:         false,
			}
			slog.Info("Tool call started", "toolCall", toolName)
			currentAssistant.AddToolCall(toolCall)
			return a.messages.Update(ctx, *currentAssistant)
		},
		OnToolCall: func(tc ai.ToolCallContent) error {
			toolCall := message.ToolCall{
				ID:               tc.ToolCallID,
				Name:             tc.ToolName,
				Input:            tc.Input,
				ProviderExecuted: false,
				Finished:         true,
			}
			currentAssistant.AddToolCall(toolCall)
			return a.messages.Update(ctx, *currentAssistant)
		},
		OnToolResult: func(result ai.ToolResultContent) error {
			var resultContent string
			isError := false
			switch result.Result.GetType() {
			case ai.ToolResultContentTypeText:
				r, ok := ai.AsToolResultOutputType[ai.ToolResultOutputContentText](result.Result)
				if ok {
					resultContent = r.Text
				}
			case ai.ToolResultContentTypeError:
				r, ok := ai.AsToolResultOutputType[ai.ToolResultOutputContentError](result.Result)
				if ok {
					isError = true
					resultContent = r.Error.Error()
				}
			case ai.ToolResultContentTypeMedia:
				// TODO: handle this message type
			}
			toolResult := message.ToolResult{
				ToolCallID: result.ToolCallID,
				Name:       result.ToolName,
				Content:    resultContent,
				IsError:    isError,
				Metadata:   result.ClientMetadata,
			}
			currentAssistant.AddToolResult(toolResult)
			return a.messages.Update(ctx, *currentAssistant)
		},
		OnStepFinish: func(stepResult ai.StepResult) error {
			slog.Info("Step Finished", "result", stepResult)
			currentAssistant.FinishThinking()
			finishReason := message.FinishReasonUnknown
			switch stepResult.FinishReason {
			case ai.FinishReasonLength:
				finishReason = message.FinishReasonMaxTokens
			case ai.FinishReasonStop:
				finishReason = message.FinishReasonEndTurn
			case ai.FinishReasonToolCalls:
				finishReason = message.FinishReasonToolUse
			}
			currentAssistant.AddFinish(finishReason, "", "")
			return a.messages.Update(ctx, *currentAssistant)
		},
	})
	if err != nil {
		isCancelErr := errors.Is(err, context.Canceled)
		isPermissionErr := errors.Is(err, permission.ErrorPermissionDenied)
		if currentAssistant == nil {
			return result, err
		}
		toolCalls := currentAssistant.ToolCalls()
		toolResults := currentAssistant.ToolResults()
		for _, tc := range toolCalls {
			if !tc.Finished {
				tc.Finished = true
				tc.Input = "{}"
			}
			currentAssistant.AddToolCall(tc)
			found := false
			for _, tr := range toolResults {
				if tr.ToolCallID == tc.ID {
					found = true
					break
				}
			}
			if !found {
				content := "There was an error while executing the tool"
				if isCancelErr {
					content = "Tool execution canceled by user"
				} else if isPermissionErr {
					content = "Permission denied"
				}
				currentAssistant.AddToolResult(message.ToolResult{
					ToolCallID: tc.ID,
					Name:       tc.Name,
					Content:    content,
					IsError:    true,
				})
			}
		}
		if isCancelErr {
			currentAssistant.AddFinish(message.FinishReasonCanceled, "Request cancelled", "")
		} else if isPermissionErr {
			currentAssistant.AddFinish(message.FinishReasonPermissionDenied, "Permission denied", "")
		} else {
			currentAssistant.AddFinish(message.FinishReasonError, "API Error", err.Error())
		}
		// TODO: handle error?
		_ = a.messages.Update(context.Background(), *currentAssistant)
	}
	return result, err
}

// Summarize implements Service.
func (a *agent) Summarize(ctx context.Context, sessionID string) error {
	// TODO: implement
	return nil
}

// UpdateModel implements Service.
func (a *agent) UpdateModel() error {
	return nil
}

func (a *agent) Cancel(sessionID string) {
	// Cancel regular requests
	if cancel, ok := a.activeRequests.Take(sessionID); ok && cancel != nil {
		slog.Info("Request cancellation initiated", "session_id", sessionID)
		cancel()
	}

	// Also check for summarize requests
	if cancel, ok := a.activeRequests.Take(sessionID + "-summarize"); ok && cancel != nil {
		slog.Info("Summarize cancellation initiated", "session_id", sessionID)
		cancel()
	}

	if a.QueuedPrompts(sessionID) > 0 {
		slog.Info("Clearing queued prompts", "session_id", sessionID)
		a.promptQueue.Del(sessionID)
	}
}

func (a *agent) ClearQueue(sessionID string) {
	if a.QueuedPrompts(sessionID) > 0 {
		slog.Info("Clearing queued prompts", "session_id", sessionID)
		a.promptQueue.Del(sessionID)
	}
}

func (a *agent) CancelAll() {
	if !a.IsBusy() {
		return
	}
	for key := range a.activeRequests.Seq2() {
		a.Cancel(key) // key is sessionID
	}

	timeout := time.After(5 * time.Second)
	for a.IsBusy() {
		select {
		case <-timeout:
			return
		default:
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (a *agent) IsBusy() bool {
	var busy bool
	for cancelFunc := range a.activeRequests.Seq() {
		if cancelFunc != nil {
			busy = true
			break
		}
	}
	return busy
}

func (a *agent) IsSessionBusy(sessionID string) bool {
	_, busy := a.activeRequests.Get(sessionID)
	return busy
}

func (a *agent) Model() catwalk.Model {
	return *a.cfg.GetModelByType(config.SelectedModelTypeLarge)
}

func (a *agent) QueuedPrompts(sessionID string) int {
	l, ok := a.promptQueue.Get(sessionID)
	if !ok {
		return 0
	}
	return len(l)
}
