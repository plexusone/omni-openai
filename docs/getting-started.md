# Getting Started

This guide covers installation and basic usage of omni-openai.

## Prerequisites

- Go 1.21 or later
- OpenAI API key

## Installation

```bash
go get github.com/plexusone/omni-openai
```

## Configuration

Set your OpenAI API key as an environment variable:

```bash
export OPENAI_API_KEY="sk-..."
```

Or pass it directly when creating providers.

## Basic Usage

### OmniLLM - Chat Completions

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    core "github.com/plexusone/omnillm-core"
    _ "github.com/plexusone/omni-openai/omnillm" // Auto-registers provider
)

func main() {
    ctx := context.Background()

    // Create provider via registry
    provider, err := core.NewProvider("openai", core.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Close()

    // Send a chat completion request
    resp, err := provider.CreateChatCompletion(ctx, &core.ChatCompletionRequest{
        Model: "gpt-4o",
        Messages: []core.Message{
            {Role: core.RoleUser, Content: "What is the capital of France?"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

### OmniVoice - Speech-to-Text

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/plexusone/omnivoice-core/stt"
    "github.com/plexusone/omni-openai/omnivoice"
)

func main() {
    ctx := context.Background()

    // Create STT provider
    provider, err := omnivoice.NewSTTProviderFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    // Read audio file
    audio, _ := os.ReadFile("speech.mp3")

    // Transcribe
    result, err := provider.Transcribe(ctx, audio, stt.TranscriptionConfig{
        Encoding: "mp3",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Transcription:", result.Text)
}
```

### OmniVoice - Text-to-Speech

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/plexusone/omnivoice-core/tts"
    "github.com/plexusone/omni-openai/omnivoice"
)

func main() {
    ctx := context.Background()

    // Create TTS provider
    provider, err := omnivoice.NewTTSProviderFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    // Synthesize speech
    result, err := provider.Synthesize(ctx, "Hello, world!", tts.SynthesisConfig{
        VoiceID: omnivoice.VoiceNova,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Save audio
    os.WriteFile("output.mp3", result.Audio, 0644)
}
```

## Next Steps

- [OmniLLM Provider](providers/omnillm.md) - Advanced chat completion features
- [OmniVoice Provider](providers/omnivoice.md) - STT/TTS configuration options
- [API Reference](api/client.md) - Complete API documentation
