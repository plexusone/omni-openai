# Client API Reference

The root package provides a direct OpenAI client for audio operations.

## Client

### NewClient

Creates a new OpenAI client with the given API key.

```go
func NewClient(apiKey string) *Client
```

### NewClientFromEnv

Creates a new client using the `OPENAI_API_KEY` environment variable.

```go
func NewClientFromEnv() (*Client, error)
```

## Transcription

### Transcribe

Converts audio to text using Whisper.

```go
func (c *Client) Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResponse, error)
```

**TranscriptionRequest:**

| Field | Type | Description |
|-------|------|-------------|
| Audio | `[]byte` | Audio data to transcribe |
| Filename | `string` | Filename for format detection |
| Model | `string` | Whisper model (default: "whisper-1") |
| Language | `string` | ISO-639-1 language code |
| Prompt | `string` | Optional guiding prompt |
| ResponseFormat | `string` | Output format |
| Temperature | `float64` | Sampling temperature (0-1) |
| TimestampGranularities | `[]string` | "word", "segment", or both |

**TranscriptionResponse:**

| Field | Type | Description |
|-------|------|-------------|
| Text | `string` | Transcribed text |
| Language | `string` | Detected language |
| Duration | `float64` | Audio duration in seconds |
| Words | `[]WordTimestamp` | Word-level timestamps |
| Segments | `[]Segment` | Segment-level timestamps |

### TranscribeFile

Transcribes audio from a file path.

```go
func (c *Client) TranscribeFile(ctx context.Context, filePath string, req TranscriptionRequest) (*TranscriptionResponse, error)
```

## Text-to-Speech

### Synthesize

Converts text to speech.

```go
func (c *Client) Synthesize(ctx context.Context, req TTSRequest) (*TTSResponse, error)
```

**TTSRequest:**

| Field | Type | Description |
|-------|------|-------------|
| Input | `string` | Text to convert (max 4096 chars) |
| Model | `string` | "tts-1" or "tts-1-hd" |
| Voice | `string` | Voice ID |
| ResponseFormat | `string` | "mp3", "opus", "aac", etc. |
| Speed | `float64` | Speech speed (0.25-4.0) |

**TTSResponse:**

| Field | Type | Description |
|-------|------|-------------|
| Audio | `[]byte` | Generated audio data |
| Format | `string` | Audio format |

### SynthesizeStream

Converts text to speech with streaming output.

```go
func (c *Client) SynthesizeStream(ctx context.Context, req TTSRequest) (io.ReadCloser, error)
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
    ModelWhisper1 = "whisper-1"
    ModelTTS1     = "tts-1"
    ModelTTS1HD   = "tts-1-hd"
)
```
