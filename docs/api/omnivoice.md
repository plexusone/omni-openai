# OmniVoice API Reference

The `omnivoice` subpackage provides STT and TTS provider adapters.

## STT Provider

### NewSTTProvider

Creates a new STT provider with the given API key.

```go
func NewSTTProvider(apiKey string) *STTProvider
```

### NewSTTProviderFromEnv

Creates a new STT provider using `OPENAI_API_KEY`.

```go
func NewSTTProviderFromEnv() (*STTProvider, error)
```

### NewSTTProviderFromClient

Creates a provider from an existing OpenAI client.

```go
func NewSTTProviderFromClient(client *Client) *STTProvider
```

### Methods

#### Name

```go
func (p *STTProvider) Name() string // Returns "openai"
```

#### Transcribe

```go
func (p *STTProvider) Transcribe(ctx context.Context, audio []byte, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error)
```

#### TranscribeFile

```go
func (p *STTProvider) TranscribeFile(ctx context.Context, filePath string, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error)
```

#### TranscribeURL

Returns `ErrURLTranscriptionNotSupported` (OpenAI doesn't support URL transcription).

```go
func (p *STTProvider) TranscribeURL(ctx context.Context, url string, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error)
```

## TTS Provider

### NewTTSProvider

Creates a new TTS provider with the given API key.

```go
func NewTTSProvider(apiKey string) *TTSProvider
```

### NewTTSProviderFromEnv

Creates a new TTS provider using `OPENAI_API_KEY`.

```go
func NewTTSProviderFromEnv() (*TTSProvider, error)
```

### NewTTSProviderFromClient

Creates a provider from an existing OpenAI client.

```go
func NewTTSProviderFromClient(client *Client) *TTSProvider
```

### Methods

#### Name

```go
func (p *TTSProvider) Name() string // Returns "openai"
```

#### ListVoices

```go
func (p *TTSProvider) ListVoices(ctx context.Context) ([]tts.Voice, error)
```

#### GetVoice

```go
func (p *TTSProvider) GetVoice(ctx context.Context, voiceID string) (*tts.Voice, error)
```

#### Synthesize

```go
func (p *TTSProvider) Synthesize(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error)
```

#### SynthesizeStream

```go
func (p *TTSProvider) SynthesizeStream(ctx context.Context, text string, config tts.SynthesisConfig) (<-chan tts.StreamChunk, error)
```

## Constants

### Voice Constants

```go
const (
    VoiceAlloy   = "alloy"
    VoiceAsh     = "ash"
    VoiceBallad  = "ballad"
    VoiceCoral   = "coral"
    VoiceEcho    = "echo"
    VoiceFable   = "fable"
    VoiceOnyx    = "onyx"
    VoiceNova    = "nova"
    VoiceSage    = "sage"
    VoiceShimmer = "shimmer"
    VoiceVerse   = "verse"
    VoiceMarin   = "marin"
    VoiceCedar   = "cedar"
)
```

### Model Constants

```go
const (
    ModelTTS1   = "tts-1"
    ModelTTS1HD = "tts-1-hd"
)
```

## Errors

```go
var ErrURLTranscriptionNotSupported = errors.New("openai: URL transcription not supported")
```
