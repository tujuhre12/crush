package provider

import (
	"context"
	"encoding/json"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/llm/tools"
	"github.com/charmbracelet/crush/internal/logging"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

type LlamaClient ProviderClient

func newLlamaClient(opts providerClientOptions) LlamaClient {
	openaiClientOptions := []option.RequestOption{}
	if opts.apiKey != "" {
		openaiClientOptions = append(openaiClientOptions, option.WithAPIKey(opts.apiKey))
	}
	openaiClientOptions = append(openaiClientOptions, option.WithBaseURL("https://api.llama.com/compat/v1/"))
	if opts.extraHeaders != nil {
		for key, value := range opts.extraHeaders {
			openaiClientOptions = append(openaiClientOptions, option.WithHeader(key, value))
		}
	}
	return &llamaClient{
		providerOptions: opts,
		client:          openai.NewClient(openaiClientOptions...),
	}
}

type llamaClient struct {
	providerOptions providerClientOptions
	client          openai.Client
}

func (l *llamaClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	openaiMessages := l.convertMessages(messages)
	openaiTools := l.convertTools(tools)
	params := l.preparedParams(openaiMessages, openaiTools)
	cfg := config.Get()
	if cfg.Options.Debug {
		jsonData, _ := json.Marshal(params)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}
	attempts := 0
	for {
		attempts++
		openaiResponse, err := l.client.Chat.Completions.New(ctx, params)
		if err != nil {
			return nil, err
		}
		content := ""
		if openaiResponse.Choices[0].Message.Content != "" {
			content = openaiResponse.Choices[0].Message.Content
		}
		toolCalls := l.toolCalls(*openaiResponse)
		finishReason := l.finishReason(string(openaiResponse.Choices[0].FinishReason))
		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}
		return &ProviderResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        l.usage(*openaiResponse),
			FinishReason: finishReason,
		}, nil
	}
}

func (l *llamaClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	openaiMessages := l.convertMessages(messages)
	openaiTools := l.convertTools(tools)
	params := l.preparedParams(openaiMessages, openaiTools)
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}
	cfg := config.Get()
	if cfg.Options.Debug {
		jsonData, _ := json.Marshal(params)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}
	eventChan := make(chan ProviderEvent)
	go func() {
		attempts := 0
		acc := openai.ChatCompletionAccumulator{}
		currentContent := ""
		toolCalls := make([]message.ToolCall, 0)
		for {
			attempts++
			openaiStream := l.client.Chat.Completions.NewStreaming(ctx, params)
			for openaiStream.Next() {
				chunk := openaiStream.Current()
				acc.AddChunk(chunk)
				for _, choice := range chunk.Choices {
					if choice.Delta.Content != "" {
						currentContent += choice.Delta.Content
					}
				}
				eventChan <- ProviderEvent{Type: EventContentDelta, Content: currentContent}
			}
			if err := openaiStream.Err(); err != nil {
				eventChan <- ProviderEvent{Type: EventError, Error: err}
				return
			}
			toolCalls = l.toolCalls(acc.ChatCompletion)
			finishReason := l.finishReason(string(acc.ChatCompletion.Choices[0].FinishReason))
			if len(toolCalls) > 0 {
				finishReason = message.FinishReasonToolUse
			}
			eventChan <- ProviderEvent{
				Type: EventComplete,
				Response: &ProviderResponse{
					Content:      currentContent,
					ToolCalls:    toolCalls,
					Usage:        l.usage(acc.ChatCompletion),
					FinishReason: finishReason,
				},
			}
			return
		}
	}()
	return eventChan
}

func (l *llamaClient) convertMessages(messages []message.Message) (openaiMessages []openai.ChatCompletionMessageParamUnion) {
	// Copied from openaiClient
	openaiMessages = append(openaiMessages, openai.SystemMessage(l.providerOptions.systemMessage))
	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			var content []openai.ChatCompletionContentPartUnionParam
			textBlock := openai.ChatCompletionContentPartTextParam{Text: msg.Content().String()}
			content = append(content, openai.ChatCompletionContentPartUnionParam{OfText: &textBlock})
			for _, binaryContent := range msg.BinaryContent() {
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{URL: binaryContent.String("llama")}
				imageBlock := openai.ChatCompletionContentPartImageParam{ImageURL: imageURL}
				content = append(content, openai.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})
			}
			openaiMessages = append(openaiMessages, openai.UserMessage(content))
		case message.Assistant:
			assistantMsg := openai.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}
			if msg.Content().String() != "" {
				assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(msg.Content().String()),
				}
			}
			if len(msg.ToolCalls()) > 0 {
				assistantMsg.ToolCalls = make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls()))
				for i, call := range msg.ToolCalls() {
					assistantMsg.ToolCalls[i] = openai.ChatCompletionMessageToolCallParam{
						ID:   call.ID,
						Type: "function",
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      call.Name,
							Arguments: call.Input,
						},
					}
				}
			}
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{OfAssistant: &assistantMsg})
		case message.Tool:
			for _, result := range msg.ToolResults() {
				openaiMessages = append(openaiMessages, openai.ToolMessage(result.Content, result.ToolCallID))
			}
		}
	}
	return
}

func (l *llamaClient) convertTools(tools []tools.BaseTool) []openai.ChatCompletionToolParam {
	openaiTools := make([]openai.ChatCompletionToolParam, len(tools))
	for i, tool := range tools {
		info := tool.Info()
		openaiTools[i] = openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        info.Name,
				Description: openai.String(info.Description),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": info.Parameters,
					"required":   info.Required,
				},
			},
		}
	}
	return openaiTools
}

func (l *llamaClient) preparedParams(messages []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) openai.ChatCompletionNewParams {
	model := l.providerOptions.model(l.providerOptions.modelType)
	cfg := config.Get()
	modelConfig := cfg.Models.Large
	if l.providerOptions.modelType == config.SmallModel {
		modelConfig = cfg.Models.Small
	}
	reasoningEffort := model.ReasoningEffort
	if modelConfig.ReasoningEffort != "" {
		reasoningEffort = modelConfig.ReasoningEffort
	}
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(model.ID),
		Messages: messages,
		Tools:    tools,
	}
	maxTokens := model.DefaultMaxTokens
	if modelConfig.MaxTokens > 0 {
		maxTokens = modelConfig.MaxTokens
	}
	if l.providerOptions.maxTokens > 0 {
		maxTokens = l.providerOptions.maxTokens
	}
	if model.CanReason {
		params.MaxCompletionTokens = openai.Int(maxTokens)
		switch reasoningEffort {
		case "low":
			params.ReasoningEffort = shared.ReasoningEffortLow
		case "medium":
			params.ReasoningEffort = shared.ReasoningEffortMedium
		case "high":
			params.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			params.ReasoningEffort = shared.ReasoningEffort(reasoningEffort)
		}
	} else {
		params.MaxTokens = openai.Int(maxTokens)
	}
	return params
}

func (l *llamaClient) toolCalls(completion openai.ChatCompletion) []message.ToolCall {
	var toolCalls []message.ToolCall
	if len(completion.Choices) > 0 && len(completion.Choices[0].Message.ToolCalls) > 0 {
		for _, call := range completion.Choices[0].Message.ToolCalls {
			toolCall := message.ToolCall{
				ID:       call.ID,
				Name:     call.Function.Name,
				Input:    call.Function.Arguments,
				Type:     "function",
				Finished: true,
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}
	return toolCalls
}

func (l *llamaClient) finishReason(reason string) message.FinishReason {
	switch reason {
	case "stop":
		return message.FinishReasonEndTurn
	case "length":
		return message.FinishReasonMaxTokens
	case "tool_calls":
		return message.FinishReasonToolUse
	default:
		return message.FinishReasonUnknown
	}
}

func (l *llamaClient) usage(completion openai.ChatCompletion) TokenUsage {
	cachedTokens := completion.Usage.PromptTokensDetails.CachedTokens
	inputTokens := completion.Usage.PromptTokens - cachedTokens
	return TokenUsage{
		InputTokens:         inputTokens,
		OutputTokens:        completion.Usage.CompletionTokens,
		CacheCreationTokens: 0, // OpenAI doesn't provide this directly
		CacheReadTokens:     cachedTokens,
	}
}

func (l *llamaClient) Model() config.Model {
	return l.providerOptions.model(l.providerOptions.modelType)
}
