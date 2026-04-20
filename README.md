# Omni-OpenAI

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

OpenAI provider adapters for the omni-* ecosystem, wrapping the official [openai-go](https://github.com/openai/openai-go) SDK.

## Features

- **OmniLLM**: Chat completions provider with streaming, tool calling, and vision support
- **OmniVoice STT**: Whisper transcription with word and segment timestamps
- **OmniVoice TTS**: OpenAI audio synthesis with multiple voices

## Installation

```bash
go get github.com/plexusone/omni-openai
```

## Usage

### OmniLLM Provider

```go
package main

import (
    "context"
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

    resp, err := provider.CreateChatCompletion(ctx, &core.ChatCompletionRequest{
        Model: "gpt-4o",
        Messages: []core.Message{
            {Role: core.RoleUser, Content: "Hello!"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Response: %s", resp.Choices[0].Message.Content)
}
```

### OmniVoice STT Provider

```go
package main

import (
    "context"
    "log"

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

    result, err := provider.Transcribe(ctx, audioData, stt.TranscriptionConfig{
        Encoding: "mp3",
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Transcription: %s", result.Text)
}
```

### OmniVoice TTS Provider

```go
package main

import (
    "context"
    "log"

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

    result, err := provider.Synthesize(ctx, "Hello, world!", tts.SynthesisConfig{
        VoiceID: omnivoice.VoiceNova,
    })
    if err != nil {
        log.Fatal(err)
    }
    // result.Audio contains the MP3 audio data
}
```

### Direct OpenAI Client Usage

```go
package main

import (
    "context"
    "log"

    openai "github.com/plexusone/omni-openai"
)

func main() {
    client, err := openai.NewClientFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Transcribe audio
    resp, err := client.Transcribe(ctx, openai.TranscriptionRequest{
        Audio:    audioData,
        Filename: "audio.mp3",
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Transcription: %s", resp.Text)

    // Synthesize speech
    ttsResp, err := client.Synthesize(ctx, openai.TTSRequest{
        Input: "Hello, world!",
        Voice: openai.VoiceAlloy,
    })
    if err != nil {
        log.Fatal(err)
    }
    // ttsResp.Audio contains the MP3 audio data
}
```

## Configuration

Set the `OPENAI_API_KEY` environment variable or pass the API key directly:

```go
client := openai.NewClient("your-api-key")
```

## Available TTS Voices

| Voice | Description |
|-------|-------------|
| alloy | Neutral, balanced |
| ash | Warm, engaging |
| ballad | Melodic, expressive |
| coral | Clear, articulate |
| echo | Smooth, natural |
| fable | Storytelling, dramatic |
| nova | Bright, energetic |
| onyx | Deep, resonant |
| sage | Calm, wise |
| shimmer | Light, cheerful |
| verse | Poetic, rhythmic |
| marin | Friendly, approachable |
| cedar | Grounded, trustworthy |

## Package Structure

```
omni-openai/
├── openai.go           # Direct OpenAI client (STT/TTS)
├── omnillm/            # OmniLLM provider adapter
│   └── adapter.go
└── omnivoice/          # OmniVoice provider adapters
    ├── stt.go
    └── tts.go
```

## License

MIT License - see [LICENSE](LICENSE) for details.

 [go-ci-svg]: https://github.com/plexusone/omni-openai/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omni-openai/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omni-openai/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omni-openai/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omni-openai/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omni-openai/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omni-openai
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omni-openai
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omni-openai
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omni-openai
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomni-openai
 [loc-svg]: https://tokei.rs/b1/github/plexusone/omni-openai
 [repo-url]: https://github.com/plexusone/omni-openai
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omni-openai/blob/master/LICENSE
