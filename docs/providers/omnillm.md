# OmniLLM Provider

The OmniLLM provider implements the `core.Provider` interface from [omnillm-core](https://github.com/plexusone/omnillm-core), providing access to OpenAI's chat completion models.

## Features

- Chat completions (GPT-4, GPT-4o, GPT-3.5 Turbo, o1, etc.)
- Streaming responses
- Tool/function calling
- Vision (image inputs)
- JSON mode
- Auto-registration with priority (overrides thin providers)

## Installation

```go
import (
    core "github.com/plexusone/omnillm-core"
    _ "github.com/plexusone/omni-openai/omnillm" // Auto-registers
)
```

The blank import auto-registers the OpenAI provider with the omnillm-core registry.

## Basic Usage

### Via Registry (Recommended)

```go
provider, err := core.NewProvider("openai", core.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

resp, err := provider.CreateChatCompletion(ctx, &core.ChatCompletionRequest{
    Model: "gpt-4o",
    Messages: []core.Message{
        {Role: core.RoleUser, Content: "Hello!"},
    },
})
```

### Direct Instantiation

```go
import "github.com/plexusone/omni-openai/omnillm"

provider, err := omnillm.New(omnillm.Config{
    APIKey:       os.Getenv("OPENAI_API_KEY"),
    BaseURL:      "",  // Optional: custom endpoint
    Organization: "",  // Optional: organization ID
})
```

## Streaming

```go
stream, err := provider.CreateChatCompletionStream(ctx, &core.ChatCompletionRequest{
    Model: "gpt-4o",
    Messages: []core.Message{
        {Role: core.RoleUser, Content: "Tell me a story"},
    },
})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk.Choices[0].Delta.Content)
}
```

## Tool Calling

```go
resp, err := provider.CreateChatCompletion(ctx, &core.ChatCompletionRequest{
    Model: "gpt-4o",
    Messages: []core.Message{
        {Role: core.RoleUser, Content: "What's the weather in Tokyo?"},
    },
    Tools: []core.Tool{
        {
            Type: "function",
            Function: core.FunctionDefinition{
                Name:        "get_weather",
                Description: "Get weather for a location",
                Parameters: map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "location": map[string]any{
                            "type":        "string",
                            "description": "City name",
                        },
                    },
                    "required": []string{"location"},
                },
            },
        },
    },
})
```

## Configuration

| Field | Type | Description |
|-------|------|-------------|
| `APIKey` | string | OpenAI API key (required) |
| `BaseURL` | string | Custom API endpoint (for Azure or proxies) |
| `Organization` | string | OpenAI organization ID |

## Capabilities

The provider reports the following capabilities:

| Capability | Supported |
|------------|-----------|
| Tools | Yes |
| Streaming | Yes |
| Vision | Yes |
| JSON Mode | Yes |
| System Role | Yes |
| Max Context Window | 128,000 tokens |
| Max Tokens Param | Yes |
