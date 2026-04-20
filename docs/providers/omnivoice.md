# OmniVoice Providers

The OmniVoice providers implement the `stt.Provider` and `tts.Provider` interfaces from [omnivoice-core](https://github.com/plexusone/omnivoice-core).

## Features

### STT (Speech-to-Text)

- Whisper model transcription
- Word-level timestamps
- Segment-level timestamps
- Multiple audio format support (mp3, wav, flac, opus, m4a, webm)
- Language detection

### TTS (Text-to-Speech)

- 13 distinct voices
- Multiple output formats (mp3, opus, aac, flac, wav, pcm)
- Adjustable speed (0.25x to 4.0x)
- Streaming output
- HD quality model option

## Installation

```go
import "github.com/plexusone/omni-openai/omnivoice"
```

## STT Provider

### Basic Usage

```go
provider, err := omnivoice.NewSTTProviderFromEnv()
if err != nil {
    log.Fatal(err)
}

result, err := provider.Transcribe(ctx, audioData, stt.TranscriptionConfig{
    Encoding: "mp3",
    Language: "en",  // Optional: auto-detected if omitted
})
if err != nil {
    log.Fatal(err)
}

fmt.Println("Text:", result.Text)
fmt.Println("Language:", result.Language)
fmt.Println("Duration:", result.Duration)
```

### With Word Timestamps

```go
result, err := provider.Transcribe(ctx, audioData, stt.TranscriptionConfig{
    Encoding:             "mp3",
    EnableWordTimestamps: true,
})

for _, segment := range result.Segments {
    fmt.Printf("[%s - %s] %s\n", segment.StartTime, segment.EndTime, segment.Text)
    for _, word := range segment.Words {
        fmt.Printf("  %s (%s)\n", word.Text, word.StartTime)
    }
}
```

### From File

```go
result, err := provider.TranscribeFile(ctx, "speech.mp3", stt.TranscriptionConfig{})
```

## TTS Provider

### Basic Usage

```go
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

// result.Audio contains MP3 data
os.WriteFile("output.mp3", result.Audio, 0644)
```

### Streaming Output

```go
chunks, err := provider.SynthesizeStream(ctx, "Long text to synthesize...", tts.SynthesisConfig{
    VoiceID: omnivoice.VoiceAlloy,
})
if err != nil {
    log.Fatal(err)
}

for chunk := range chunks {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    if chunk.IsFinal {
        break
    }
    // Process chunk.Audio
}
```

### Configuration Options

```go
result, err := provider.Synthesize(ctx, "Hello!", tts.SynthesisConfig{
    Model:        "tts-1-hd",  // Higher quality
    VoiceID:      omnivoice.VoiceCoral,
    OutputFormat: "opus",
    Speed:        1.25,  // 25% faster
})
```

## Available Voices

| Voice | Constant | Description |
|-------|----------|-------------|
| alloy | `VoiceAlloy` | Neutral, balanced |
| ash | `VoiceAsh` | Warm, engaging |
| ballad | `VoiceBallad` | Melodic, expressive |
| coral | `VoiceCoral` | Clear, articulate |
| echo | `VoiceEcho` | Smooth, natural |
| fable | `VoiceFable` | Storytelling, dramatic |
| nova | `VoiceNova` | Bright, energetic |
| onyx | `VoiceOnyx` | Deep, resonant |
| sage | `VoiceSage` | Calm, wise |
| shimmer | `VoiceShimmer` | Light, cheerful |
| verse | `VoiceVerse` | Poetic, rhythmic |
| marin | `VoiceMarin` | Friendly, approachable |
| cedar | `VoiceCedar` | Grounded, trustworthy |

## Models

### STT Models

| Model | Constant | Description |
|-------|----------|-------------|
| whisper-1 | `ModelWhisper1` | Whisper v2 large |

### TTS Models

| Model | Constant | Description |
|-------|----------|-------------|
| tts-1 | `ModelTTS1` | Fast, lower latency |
| tts-1-hd | `ModelTTS1HD` | Higher quality audio |

## Output Formats

| Format | Description |
|--------|-------------|
| mp3 | Default, widely supported |
| opus | Efficient, good for streaming |
| aac | Apple ecosystem |
| flac | Lossless |
| wav | Uncompressed PCM |
| pcm | Raw PCM (24kHz, 16-bit, mono) |
