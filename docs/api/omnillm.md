# OmniLLM API Reference

The `omnillm` subpackage provides the OmniLLM provider adapter.

## Provider

### New

Creates a new OpenAI provider with the given configuration.

```go
func New(cfg Config) (*Provider, error)
```

### Config

| Field | Type | Description |
|-------|------|-------------|
| APIKey | `string` | OpenAI API key (required) |
| BaseURL | `string` | Custom API endpoint |
| Organization | `string` | OpenAI organization ID |

## Methods

### Name

Returns the provider identifier.

```go
func (p *Provider) Name() string // Returns "openai"
```

### Capabilities

Returns the provider's supported features.

```go
func (p *Provider) Capabilities() core.Capabilities
```

Returns:

```go
core.Capabilities{
    Tools:             true,
    Streaming:         true,
    Vision:            true,
    JSON:              true,
    SystemRole:        true,
    MaxContextWindow:  128000,
    SupportsMaxTokens: true,
}
```

### CreateChatCompletion

Sends a chat completion request.

```go
func (p *Provider) CreateChatCompletion(ctx context.Context, req *core.ChatCompletionRequest) (*core.ChatCompletionResponse, error)
```

### CreateChatCompletionStream

Creates a streaming chat completion.

```go
func (p *Provider) CreateChatCompletionStream(ctx context.Context, req *core.ChatCompletionRequest) (core.ChatCompletionStream, error)
```

### Close

Releases resources held by the provider.

```go
func (p *Provider) Close() error
```

## Auto-Registration

The package auto-registers with omnillm-core on import:

```go
import _ "github.com/plexusone/omni-openai/omnillm"
```

This registers the OpenAI provider with priority `PriorityThick` (10), which overrides any thin providers.
