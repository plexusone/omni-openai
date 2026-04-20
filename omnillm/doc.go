// Package omnillm provides an OmniLLM adapter for OpenAI's API using the official openai-go SDK.
//
// This package implements the [core.Provider] interface from omnillm-core,
// wrapping the official OpenAI Go SDK (github.com/openai/openai-go).
//
// # Features
//
// The adapter supports all major OpenAI features:
//   - Chat completions (GPT-4, GPT-4 Turbo, GPT-3.5 Turbo, etc.)
//   - Streaming responses
//   - Tool/function calling
//   - Vision (image inputs)
//   - JSON mode
//
// # Basic Usage
//
//	import (
//	    core "github.com/plexusone/omnillm-core"
//	    "github.com/plexusone/omni-openai/omnillm"
//	)
//
//	func main() {
//	    provider, err := omnillm.New(omnillm.Config{
//	        APIKey: os.Getenv("OPENAI_API_KEY"),
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer provider.Close()
//
//	    resp, err := provider.CreateChatCompletion(ctx, &core.ChatCompletionRequest{
//	        Model: "gpt-4",
//	        Messages: []core.Message{
//	            {Role: core.RoleUser, Content: "Hello!"},
//	        },
//	    })
//	}
//
// # Configuration
//
// The [Config] struct supports:
//   - APIKey: Your OpenAI API key (required)
//   - BaseURL: Custom API endpoint (optional, for Azure or proxies)
//   - Organization: OpenAI organization ID (optional)
//
// # Streaming
//
//	stream, err := provider.CreateChatCompletionStream(ctx, &core.ChatCompletionRequest{
//	    Model: "gpt-4",
//	    Messages: []core.Message{
//	        {Role: core.RoleUser, Content: "Tell me a story"},
//	    },
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer stream.Close()
//
//	for {
//	    chunk, err := stream.Recv()
//	    if err == io.EOF {
//	        break
//	    }
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Print(chunk.Choices[0].Delta.Content)
//	}
package omnillm
