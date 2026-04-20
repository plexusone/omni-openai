# Omni-OpenAI

OpenAI provider adapters for the omni-* ecosystem, wrapping the official [openai-go](https://github.com/openai/openai-go) SDK.

## Overview

Omni-OpenAI provides unified adapters for integrating OpenAI's APIs with the omni-* ecosystem:

- **OmniLLM** - Chat completions provider for [omnillm-core](https://github.com/plexusone/omnillm-core)
- **OmniVoice** - STT/TTS providers for [omnivoice-core](https://github.com/plexusone/omnivoice-core)

## Features

### OmniLLM Provider

- Chat completions (GPT-4, GPT-4o, GPT-3.5 Turbo, etc.)
- Streaming responses
- Tool/function calling
- Vision (image inputs)
- JSON mode
- Auto-registration with omnillm-core registry

### OmniVoice Providers

- **STT (Speech-to-Text)**: Whisper transcription with word and segment timestamps
- **TTS (Text-to-Speech)**: Audio synthesis with 13 voices

## Installation

```bash
go get github.com/plexusone/omni-openai
```

## Quick Start

=== "OmniLLM"

    ```go
    import (
        core "github.com/plexusone/omnillm-core"
        _ "github.com/plexusone/omni-openai/omnillm"
    )

    provider, _ := core.NewProvider("openai", core.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    resp, _ := provider.CreateChatCompletion(ctx, &core.ChatCompletionRequest{
        Model:    "gpt-4o",
        Messages: []core.Message{{Role: core.RoleUser, Content: "Hello!"}},
    })
    ```

=== "OmniVoice STT"

    ```go
    import "github.com/plexusone/omni-openai/omnivoice"

    provider, _ := omnivoice.NewSTTProviderFromEnv()
    result, _ := provider.Transcribe(ctx, audioData, stt.TranscriptionConfig{})
    fmt.Println(result.Text)
    ```

=== "OmniVoice TTS"

    ```go
    import "github.com/plexusone/omni-openai/omnivoice"

    provider, _ := omnivoice.NewTTSProviderFromEnv()
    result, _ := provider.Synthesize(ctx, "Hello!", tts.SynthesisConfig{
        VoiceID: omnivoice.VoiceNova,
    })
    // result.Audio contains MP3 data
    ```

## Configuration

Set the `OPENAI_API_KEY` environment variable:

```bash
export OPENAI_API_KEY="sk-..."
```

## Package Structure

```
omni-openai/
├── openai.go           # Direct OpenAI client (STT/TTS)
├── omnillm/            # OmniLLM provider adapter
│   ├── adapter.go
│   └── doc.go
└── omnivoice/          # OmniVoice provider adapters
    ├── stt.go
    └── tts.go
```

## License

MIT License
