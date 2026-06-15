# OmniVoice Realtime

The `omnivoice/realtime` package provides native voice-to-voice capabilities via the OpenAI Realtime API.

## Overview

OpenAI Realtime API enables:

- **Ultra-low latency** - ~100ms voice-to-voice response time
- **Native audio** - No separate STT/TTS, model handles audio directly
- **Function calling** - Execute tools during conversation
- **Turn detection** - Automatic voice activity detection (VAD)

## Quick Start

```go
import (
    "context"
    "os"

    "github.com/plexusone/omni-openai/omnivoice/realtime"
)

func main() {
    ctx := context.Background()

    // Create provider
    provider := realtime.NewProvider(os.Getenv("OPENAI_API_KEY"),
        realtime.WithVoice("alloy"),
        realtime.WithInstructions("You are a helpful assistant."),
    )

    // Create audio input channel
    audioIn := make(chan []byte, 100)

    // Start streaming
    audioCh, transcriptCh, err := provider.ProcessAudioStream(ctx, audioIn, realtime.ProcessConfig{})
    if err != nil {
        log.Fatal(err)
    }

    // Send audio from microphone (PCM16 24kHz mono)
    go streamMicrophoneAudio(audioIn)

    // Process responses
    for {
        select {
        case audio, ok := <-audioCh:
            if !ok {
                return
            }
            if len(audio.Audio) > 0 {
                playAudio(audio.Audio) // PCM16 24kHz mono
            }
            if audio.IsFinal {
                log.Println("Turn complete")
            }

        case transcript, ok := <-transcriptCh:
            if !ok {
                return
            }
            if transcript.IsInput {
                log.Printf("User: %s", transcript.Text)
            } else {
                log.Printf("Assistant: %s", transcript.Text)
            }
        }
    }
}
```

## Registry Integration

*Added in v0.4.0*

Use the omnivoice-core registry for automatic provider discovery:

```go
import (
    omnivoice "github.com/plexusone/omnivoice-core"
    "github.com/plexusone/omnivoice-core/registry"
    _ "github.com/plexusone/omni-openai/omnivoice/realtime" // Auto-register
)

// Get realtime provider via registry
provider, err := omnivoice.GetRealtimeProvider("openai",
    registry.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    registry.WithModel("gpt-4o-realtime-preview-2024-12-17"),
    registry.WithVoice("alloy"),
    registry.WithInstructions("You are a helpful assistant."),
)
if err != nil {
    log.Fatal(err)
}

// Process audio streams
audioCh, transcriptCh, err := provider.ProcessAudioStream(ctx, audioIn, nil)
```

### Type-Safe Registry Options

Provider-specific options for OpenAI Realtime configuration:

```go
import "github.com/plexusone/omni-openai/omnivoice/realtime"

provider, err := omnivoice.GetRealtimeProvider("openai",
    registry.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    // Type-safe OpenAI-specific options
    realtime.WithRegistryTools(tools),
    realtime.WithRegistryTurnDetection(turnDetectionConfig),
    realtime.WithRegistryInputAudioFormat("pcm16"),
    realtime.WithRegistryOutputAudioFormat("pcm16"),
    realtime.WithRegistryModalities([]string{"text", "audio"}),
    realtime.WithRegistryTemperature(0.8),
    realtime.WithRegistryMaxResponseOutputTokens(4096),
)
```

### Accessing Underlying Provider

Access the underlying OpenAI provider for full API access:

```go
wrapper := provider.(*realtime.RealtimeWrapper)
openaiProvider := wrapper.Provider()

// Use OpenAI-specific methods
```

## Configuration

### Provider Options

```go
provider := realtime.NewProvider(apiKey,
    // Voice selection
    realtime.WithVoice("nova"),

    // System instructions
    realtime.WithInstructions("You are a customer service agent."),

    // Model selection (default: gpt-4o-realtime-preview)
    realtime.WithModel("gpt-4o-realtime-preview-2024-12-17"),

    // Response modalities
    realtime.WithModalities("text", "audio"),

    // Turn detection mode
    realtime.WithTurnDetection(realtime.TurnDetectionServerVAD),

    // Function tools
    realtime.WithTools(
        realtime.Tool{
            Type: "function",
            Name: "get_weather",
            Description: "Get current weather",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "location": map[string]any{"type": "string"},
                },
            },
        },
    ),
)
```

### Available Voices

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

## Audio Format

### Input Audio

- **Format**: PCM16 (signed 16-bit little-endian)
- **Sample Rate**: 24kHz
- **Channels**: Mono
- **Chunk Size**: Recommended 20-100ms chunks

### Output Audio

- **Format**: PCM16 (signed 16-bit little-endian)
- **Sample Rate**: 24kHz
- **Channels**: Mono

## Function Calling

Handle function calls during the conversation:

```go
config := realtime.ProcessConfig{
    Tools: []realtime.Tool{
        {
            Type:        "function",
            Name:        "lookup_order",
            Description: "Look up an order by ID",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "order_id": map[string]any{"type": "string"},
                },
                "required": []string{"order_id"},
            },
        },
    },
    OnFunctionCall: func(name, arguments string) (string, error) {
        switch name {
        case "lookup_order":
            var args struct {
                OrderID string `json:"order_id"`
            }
            json.Unmarshal([]byte(arguments), &args)

            order := lookupOrder(args.OrderID)
            return json.Marshal(order)
        }
        return "", fmt.Errorf("unknown function: %s", name)
    },
}

audioCh, transcriptCh, err := provider.ProcessAudioStream(ctx, audioIn, config)
```

## Events

The WebSocket client sends and receives these event types:

### Client Events

| Event | Description |
|-------|-------------|
| `session.update` | Update session configuration |
| `input_audio_buffer.append` | Send audio chunk |
| `input_audio_buffer.commit` | Commit buffered audio |
| `input_audio_buffer.clear` | Clear audio buffer |
| `conversation.item.create` | Add conversation item |
| `response.create` | Request a response |
| `response.cancel` | Cancel in-progress response |

### Server Events

| Event | Description |
|-------|-------------|
| `session.created` | Session established |
| `session.updated` | Session config updated |
| `response.audio.delta` | Audio chunk |
| `response.audio.done` | Audio complete |
| `response.audio_transcript.delta` | Transcript chunk |
| `response.audio_transcript.done` | Transcript complete |
| `response.function_call_arguments.done` | Function call ready |
| `error` | Error occurred |

## Integration with Call Systems

### Twilio Media Streams

```go
// Twilio sends mulaw 8kHz, needs conversion
twilioAudio := make(chan []byte)  // mulaw 8kHz

// Convert to PCM16 24kHz for Realtime API
audioIn := make(chan []byte)
go func() {
    for chunk := range twilioAudio {
        pcm := convertMulawToPCM16(chunk)
        upsampled := resample8kTo24k(pcm)
        audioIn <- upsampled
    }
}()

// Realtime API
audioCh, _, _ := provider.ProcessAudioStream(ctx, audioIn, config)

// Convert back for Twilio
for audio := range audioCh {
    downsampled := resample24kTo8k(audio.Audio)
    mulaw := convertPCM16ToMulaw(downsampled)
    sendToTwilio(mulaw)
}
```

### LiveKit

```go
// LiveKit provides PCM16 48kHz stereo
livekitAudio := make(chan []byte)

// Resample and convert to mono 24kHz
audioIn := make(chan []byte)
go func() {
    for chunk := range livekitAudio {
        mono := stereoToMono(chunk)
        resampled := resample48kTo24k(mono)
        audioIn <- resampled
    }
}()

// Output back to LiveKit
audioCh, _, _ := provider.ProcessAudioStream(ctx, audioIn, config)
for audio := range audioCh {
    stereo := monoToStereo(audio.Audio)
    upsampled := resample24kTo48k(stereo)
    sendToLiveKit(upsampled)
}
```

## Error Handling

```go
audioCh, transcriptCh, err := provider.ProcessAudioStream(ctx, audioIn, config)
if err != nil {
    // Connection failed
    log.Fatal(err)
}

// Monitor for errors during streaming
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case audio, ok := <-audioCh:
        if !ok {
            // Channel closed, check for error
            return nil
        }
        // Process audio
    }
}
```

## vs Traditional Pipeline (STT+LLM+TTS)

OpenAI Realtime provides native voice-to-voice, eliminating the need for separate STT and TTS providers.

| Aspect | Traditional | OpenAI Realtime |
|--------|-------------|-----------------|
| **Latency** | 500-1500ms | ~100ms |
| **API Calls** | 3 (STT + LLM + TTS) | 1 WebSocket |
| **Barge-in** | Complex coordination | Native support |
| **Voice options** | 1000s (ElevenLabs, etc.) | 11 preset voices |
| **Custom voices** | Yes (cloning) | No |
| **Domain STT** | Yes (medical, legal) | No |

### When to Use OpenAI Realtime

- Low latency is critical
- Natural conversation with interruptions
- Simpler architecture preferred
- Preset voices are acceptable

### When to Use Traditional Pipeline

- Custom/cloned voices required
- Domain-specific STT accuracy needed
- Specific language support
- Cost optimization with caching

See [Voice Architecture Guide](https://plexusone.dev/omnivoice-core/voice-architecture) for detailed comparison.

## Best Practices

1. **Buffer audio input** - Use buffered channel (100+ capacity)
2. **Handle disconnects** - Implement reconnection logic
3. **Monitor latency** - Log round-trip times for debugging
4. **Test with real audio** - Synthetic tests miss edge cases
5. **Use appropriate voice** - Match voice to use case
