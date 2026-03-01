// Package omnivoice provides omnivoice STT and TTS provider implementations for OpenAI.
package omnivoice

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/plexusone/omnivoice-core/stt"

	oaiclient "github.com/plexusone/omnivoice-openai"
)

// ErrURLTranscriptionNotSupported is returned when URL transcription is not supported.
var ErrURLTranscriptionNotSupported = errors.New("openai: URL transcription not supported")

// STTProvider implements stt.Provider for OpenAI Whisper.
type STTProvider struct {
	client *oaiclient.Client
}

// NewSTTProvider creates a new OpenAI STT provider.
func NewSTTProvider(apiKey string) *STTProvider {
	return &STTProvider{
		client: oaiclient.NewClient(apiKey),
	}
}

// NewSTTProviderFromEnv creates a new OpenAI STT provider using OPENAI_API_KEY.
func NewSTTProviderFromEnv() (*STTProvider, error) {
	client, err := oaiclient.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return &STTProvider{client: client}, nil
}

// NewSTTProviderFromClient creates a provider from an existing OpenAI client.
func NewSTTProviderFromClient(client *oaiclient.Client) *STTProvider {
	return &STTProvider{client: client}
}

// Name returns the provider name.
func (p *STTProvider) Name() string {
	return "openai"
}

// Transcribe converts audio to text using Whisper.
func (p *STTProvider) Transcribe(ctx context.Context, audio []byte, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	req := oaiclient.TranscriptionRequest{
		Audio:    audio,
		Filename: getFilename(config.Encoding),
		Language: config.Language,
	}

	if config.Model != "" {
		req.Model = config.Model
	}

	// Request word timestamps if enabled
	if config.EnableWordTimestamps {
		req.TimestampGranularities = []string{"word", "segment"}
	}

	resp, err := p.client.Transcribe(ctx, req)
	if err != nil {
		return nil, err
	}

	return convertTranscriptionResult(resp), nil
}

// TranscribeFile transcribes audio from a file path.
func (p *STTProvider) TranscribeFile(ctx context.Context, filePath string, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	req := oaiclient.TranscriptionRequest{
		Filename: filepath.Base(filePath),
		Language: config.Language,
	}

	if config.Model != "" {
		req.Model = config.Model
	}

	if config.EnableWordTimestamps {
		req.TimestampGranularities = []string{"word", "segment"}
	}

	resp, err := p.client.TranscribeFile(ctx, filePath, req)
	if err != nil {
		return nil, err
	}

	return convertTranscriptionResult(resp), nil
}

// TranscribeURL transcribes audio from a URL.
// Note: OpenAI doesn't support URL transcription directly.
func (p *STTProvider) TranscribeURL(ctx context.Context, url string, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	return nil, ErrURLTranscriptionNotSupported
}

// convertTranscriptionResult converts OpenAI response to omnivoice format.
func convertTranscriptionResult(resp *oaiclient.TranscriptionResponse) *stt.TranscriptionResult {
	result := &stt.TranscriptionResult{
		Text:     resp.Text,
		Language: resp.Language,
		Duration: time.Duration(resp.Duration * float64(time.Second)),
	}

	// Convert segments
	for _, seg := range resp.Segments {
		segment := stt.Segment{
			Text:       seg.Text,
			StartTime:  time.Duration(seg.Start * float64(time.Second)),
			EndTime:    time.Duration(seg.End * float64(time.Second)),
			Confidence: 1.0 - seg.NoSpeechProb, // Approximate confidence
		}

		result.Segments = append(result.Segments, segment)
	}

	// Convert words
	if len(resp.Words) > 0 && len(result.Segments) > 0 {
		// Attach words to segments based on timing
		wordIdx := 0
		for i := range result.Segments {
			for wordIdx < len(resp.Words) {
				w := resp.Words[wordIdx]
				wordStart := time.Duration(w.Start * float64(time.Second))

				if wordStart >= result.Segments[i].StartTime && wordStart < result.Segments[i].EndTime {
					result.Segments[i].Words = append(result.Segments[i].Words, stt.Word{
						Text:       w.Word,
						StartTime:  time.Duration(w.Start * float64(time.Second)),
						EndTime:    time.Duration(w.End * float64(time.Second)),
						Confidence: 1.0, // Whisper doesn't provide word-level confidence
					})
					wordIdx++
				} else if wordStart >= result.Segments[i].EndTime {
					break
				} else {
					wordIdx++
				}
			}
		}
	}

	return result
}

// getFilename returns a filename with appropriate extension for the encoding.
func getFilename(encoding string) string {
	switch encoding {
	case "mp3":
		return "audio.mp3"
	case "wav":
		return "audio.wav"
	case "flac":
		return "audio.flac"
	case "opus":
		return "audio.opus"
	case "m4a":
		return "audio.m4a"
	case "webm":
		return "audio.webm"
	default:
		return "audio.mp3"
	}
}
