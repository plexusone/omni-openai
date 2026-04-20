package omnillm

import (
	"context"
	"io"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/packages/ssestream"
	"github.com/openai/openai-go/shared"
	core "github.com/plexusone/omnillm-core"
)

func init() {
	// Register OpenAI as a thick provider (priority 10, overrides thin provider)
	core.RegisterProvider(core.ProviderNameOpenAI, newProviderFromConfig, core.PriorityThick)
}

// newProviderFromConfig creates a new OpenAI provider from omnillm config (for registry).
func newProviderFromConfig(config core.ProviderConfig) (core.Provider, error) {
	return New(Config{
		APIKey:  config.APIKey,
		BaseURL: config.BaseURL,
	})
}

// Config holds configuration for the OpenAI provider.
type Config struct {
	// APIKey is the OpenAI API key (required).
	APIKey string

	// BaseURL is an optional custom API endpoint.
	// Use this for Azure OpenAI or proxy servers.
	BaseURL string

	// Organization is the optional OpenAI organization ID.
	Organization string
}

// Provider implements core.Provider using the official OpenAI SDK.
type Provider struct {
	client openai.Client
	config Config
}

// Ensure Provider implements core.Provider at compile time.
var _ core.Provider = (*Provider)(nil)

// New creates a new OpenAI provider with the given configuration.
func New(cfg Config) (*Provider, error) {
	if cfg.APIKey == "" {
		return nil, core.ErrInvalidAPIKey
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}

	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	if cfg.Organization != "" {
		opts = append(opts, option.WithOrganization(cfg.Organization))
	}

	client := openai.NewClient(opts...)

	return &Provider{
		client: client,
		config: cfg,
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "openai"
}

// Capabilities returns the provider's supported features.
func (p *Provider) Capabilities() core.Capabilities {
	return core.Capabilities{
		Tools:             true,
		Streaming:         true,
		Vision:            true,
		JSON:              true,
		SystemRole:        true,
		MaxContextWindow:  128000, // GPT-4 Turbo
		SupportsMaxTokens: true,
	}
}

// Close releases resources held by the provider.
func (p *Provider) Close() error {
	// The OpenAI SDK doesn't require explicit cleanup
	return nil
}

// CreateChatCompletion sends a chat completion request and returns the response.
func (p *Provider) CreateChatCompletion(ctx context.Context, req *core.ChatCompletionRequest) (*core.ChatCompletionResponse, error) {
	params := p.buildParams(req)

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, p.wrapError(err)
	}

	return p.convertResponse(resp), nil
}

// CreateChatCompletionStream creates a streaming chat completion.
func (p *Provider) CreateChatCompletionStream(ctx context.Context, req *core.ChatCompletionRequest) (core.ChatCompletionStream, error) {
	params := p.buildParams(req)

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)

	return &streamAdapter{stream: stream}, nil
}

// buildParams converts a core request to OpenAI SDK params.
func (p *Provider) buildParams(req *core.ChatCompletionRequest) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model:    req.Model,
		Messages: p.convertMessages(req.Messages),
	}

	if req.MaxTokens != nil {
		params.MaxTokens = param.NewOpt(int64(*req.MaxTokens))
	}
	if req.Temperature != nil {
		params.Temperature = param.NewOpt(*req.Temperature)
	}
	if req.TopP != nil {
		params.TopP = param.NewOpt(*req.TopP)
	}
	if len(req.Stop) > 0 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfStringArray: req.Stop,
		}
	}
	if req.PresencePenalty != nil {
		params.PresencePenalty = param.NewOpt(*req.PresencePenalty)
	}
	if req.FrequencyPenalty != nil {
		params.FrequencyPenalty = param.NewOpt(*req.FrequencyPenalty)
	}
	if req.User != nil {
		params.User = param.NewOpt(*req.User)
	}
	if req.Seed != nil {
		params.Seed = param.NewOpt(int64(*req.Seed))
	}
	if req.N != nil {
		params.N = param.NewOpt(int64(*req.N))
	}
	if req.Logprobs != nil && *req.Logprobs {
		params.Logprobs = param.NewOpt(true)
		if req.TopLogprobs != nil {
			params.TopLogprobs = param.NewOpt(int64(*req.TopLogprobs))
		}
	}

	// Response format
	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_object" {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
		}
	}

	// Tools
	if len(req.Tools) > 0 {
		params.Tools = p.convertTools(req.Tools)
	}

	// Tool choice
	if req.ToolChoice != nil {
		params.ToolChoice = p.convertToolChoice(req.ToolChoice)
	}

	return params
}

// convertMessages converts core messages to OpenAI message params.
func (p *Provider) convertMessages(messages []core.Message) []openai.ChatCompletionMessageParamUnion {
	result := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case core.RoleSystem:
			result = append(result, openai.SystemMessage(msg.Content))

		case core.RoleUser:
			result = append(result, openai.UserMessage(msg.Content))

		case core.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				// Assistant message with tool calls
				toolCalls := make([]openai.ChatCompletionMessageToolCallParam, 0, len(msg.ToolCalls))
				for _, tc := range msg.ToolCalls {
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
						ID: tc.ID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					})
				}
				result = append(result, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &openai.ChatCompletionAssistantMessageParam{
						Content: openai.ChatCompletionAssistantMessageParamContentUnion{
							OfString: param.NewOpt(msg.Content),
						},
						ToolCalls: toolCalls,
					},
				})
			} else {
				result = append(result, openai.AssistantMessage(msg.Content))
			}

		case core.RoleTool:
			toolCallID := ""
			if msg.ToolCallID != nil {
				toolCallID = *msg.ToolCallID
			}
			result = append(result, openai.ToolMessage(msg.Content, toolCallID))
		}
	}

	return result
}

// convertTools converts core tools to OpenAI tool params.
func (p *Provider) convertTools(tools []core.Tool) []openai.ChatCompletionToolParam {
	result := make([]openai.ChatCompletionToolParam, 0, len(tools))

	for _, tool := range tools {
		// Convert parameters to FunctionParameters
		var funcParams shared.FunctionParameters
		if tool.Function.Parameters != nil {
			// Parameters should be a JSON Schema object
			if params, ok := tool.Function.Parameters.(map[string]any); ok {
				funcParams = params
			}
		}

		result = append(result, openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        tool.Function.Name,
				Description: param.NewOpt(tool.Function.Description),
				Parameters:  funcParams,
			},
		})
	}

	return result
}

// convertToolChoice converts core tool choice to OpenAI format.
func (p *Provider) convertToolChoice(choice any) openai.ChatCompletionToolChoiceOptionUnionParam {
	switch v := choice.(type) {
	case string:
		switch v {
		case "auto":
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: param.NewOpt("auto"),
			}
		case "none":
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: param.NewOpt("none"),
			}
		case "required":
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: param.NewOpt("required"),
			}
		}
	case map[string]any:
		// Specific function choice
		if fn, ok := v["function"].(map[string]any); ok {
			if name, ok := fn["name"].(string); ok {
				return openai.ChatCompletionToolChoiceOptionParamOfChatCompletionNamedToolChoice(
					openai.ChatCompletionNamedToolChoiceFunctionParam{
						Name: name,
					},
				)
			}
		}
	}
	return openai.ChatCompletionToolChoiceOptionUnionParam{
		OfAuto: param.NewOpt("auto"),
	}
}

// convertResponse converts an OpenAI response to core format.
func (p *Provider) convertResponse(resp *openai.ChatCompletion) *core.ChatCompletionResponse {
	result := &core.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  string(resp.Object),
		Created: resp.Created,
		Model:   resp.Model,
		Usage: core.Usage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		},
	}

	if resp.SystemFingerprint != "" {
		result.SystemFingerprint = &resp.SystemFingerprint
	}

	for _, choice := range resp.Choices {
		coreChoice := core.ChatCompletionChoice{
			Index: int(choice.Index),
			Message: core.Message{
				Role:    core.Role(choice.Message.Role),
				Content: choice.Message.Content,
			},
		}

		// Convert finish reason
		if choice.FinishReason != "" {
			reason := choice.FinishReason
			coreChoice.FinishReason = &reason
		}

		// Convert tool calls
		for _, tc := range choice.Message.ToolCalls {
			coreChoice.Message.ToolCalls = append(coreChoice.Message.ToolCalls, core.ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: core.ToolFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}

		result.Choices = append(result.Choices, coreChoice)
	}

	return result
}

// wrapError converts OpenAI SDK errors to core errors.
func (p *Provider) wrapError(err error) error {
	if err == nil {
		return nil
	}

	// Try to extract API error details
	// The OpenAI SDK may return structured errors we can inspect
	return core.NewAPIError("openai", 0, "", err.Error())
}

// streamAdapter wraps an OpenAI stream to implement core.ChatCompletionStream.
type streamAdapter struct {
	stream *ssestream.Stream[openai.ChatCompletionChunk]
}

// Recv receives the next chunk from the stream.
func (s *streamAdapter) Recv() (*core.ChatCompletionChunk, error) {
	if !s.stream.Next() {
		err := s.stream.Err()
		if err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	chunk := s.stream.Current()
	return s.convertChunk(chunk), nil
}

// Close closes the stream.
func (s *streamAdapter) Close() error {
	return s.stream.Close()
}

// convertChunk converts an OpenAI chunk to core format.
func (s *streamAdapter) convertChunk(chunk openai.ChatCompletionChunk) *core.ChatCompletionChunk {
	result := &core.ChatCompletionChunk{
		ID:      chunk.ID,
		Object:  string(chunk.Object),
		Created: chunk.Created,
		Model:   chunk.Model,
	}

	if chunk.SystemFingerprint != "" {
		result.SystemFingerprint = &chunk.SystemFingerprint
	}

	for _, choice := range chunk.Choices {
		coreChoice := core.ChatCompletionChoice{
			Index: int(choice.Index),
		}

		// Set delta
		coreChoice.Delta = &core.Message{
			Role:    core.Role(choice.Delta.Role),
			Content: choice.Delta.Content,
		}

		// Convert tool calls in delta
		for _, tc := range choice.Delta.ToolCalls {
			coreChoice.Delta.ToolCalls = append(coreChoice.Delta.ToolCalls, core.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: core.ToolFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}

		// Convert finish reason
		if choice.FinishReason != "" {
			reason := choice.FinishReason
			coreChoice.FinishReason = &reason
		}

		result.Choices = append(result.Choices, coreChoice)
	}

	// Usage is typically only in the final chunk
	if chunk.Usage.TotalTokens > 0 {
		result.Usage = &core.Usage{
			PromptTokens:     int(chunk.Usage.PromptTokens),
			CompletionTokens: int(chunk.Usage.CompletionTokens),
			TotalTokens:      int(chunk.Usage.TotalTokens),
		}
	}

	return result
}
