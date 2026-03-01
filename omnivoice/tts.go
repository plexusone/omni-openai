// Package omnivoice provides omnivoice STT and TTS provider implementations for OpenAI.
package omnivoice

import (
	"context"
	"io"

	"github.com/plexusone/omnivoice-core/tts"

	oaiclient "github.com/plexusone/omnivoice-openai"
)

// OpenAI TTS voices with their metadata.
var openAIVoices = []tts.Voice{
	{ID: "alloy", Name: "Alloy", Language: "en", Gender: "neutral", Provider: "openai"},
	{ID: "ash", Name: "Ash", Language: "en", Gender: "male", Provider: "openai"},
	{ID: "ballad", Name: "Ballad", Language: "en", Gender: "male", Provider: "openai"},
	{ID: "coral", Name: "Coral", Language: "en", Gender: "female", Provider: "openai"},
	{ID: "echo", Name: "Echo", Language: "en", Gender: "male", Provider: "openai"},
	{ID: "fable", Name: "Fable", Language: "en", Gender: "neutral", Provider: "openai"},
	{ID: "onyx", Name: "Onyx", Language: "en", Gender: "male", Provider: "openai"},
	{ID: "nova", Name: "Nova", Language: "en", Gender: "female", Provider: "openai"},
	{ID: "sage", Name: "Sage", Language: "en", Gender: "female", Provider: "openai"},
	{ID: "shimmer", Name: "Shimmer", Language: "en", Gender: "female", Provider: "openai"},
	{ID: "verse", Name: "Verse", Language: "en", Gender: "male", Provider: "openai"},
	{ID: "marin", Name: "Marin", Language: "en", Gender: "female", Provider: "openai"},
	{ID: "cedar", Name: "Cedar", Language: "en", Gender: "male", Provider: "openai"},
}

// TTSProvider implements tts.Provider for OpenAI TTS.
type TTSProvider struct {
	client *oaiclient.Client
}

// NewTTSProvider creates a new OpenAI TTS provider.
func NewTTSProvider(apiKey string) *TTSProvider {
	return &TTSProvider{
		client: oaiclient.NewClient(apiKey),
	}
}

// NewTTSProviderFromEnv creates a new OpenAI TTS provider using OPENAI_API_KEY.
func NewTTSProviderFromEnv() (*TTSProvider, error) {
	client, err := oaiclient.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return &TTSProvider{client: client}, nil
}

// NewTTSProviderFromClient creates a provider from an existing OpenAI client.
func NewTTSProviderFromClient(client *oaiclient.Client) *TTSProvider {
	return &TTSProvider{client: client}
}

// Name returns the provider name.
func (p *TTSProvider) Name() string {
	return "openai"
}

// ListVoices returns available voices from OpenAI.
func (p *TTSProvider) ListVoices(ctx context.Context) ([]tts.Voice, error) {
	// Return a copy to prevent modification
	voices := make([]tts.Voice, len(openAIVoices))
	copy(voices, openAIVoices)
	return voices, nil
}

// GetVoice returns a specific voice by ID.
func (p *TTSProvider) GetVoice(ctx context.Context, voiceID string) (*tts.Voice, error) {
	for _, v := range openAIVoices {
		if v.ID == voiceID {
			voice := v // Copy
			return &voice, nil
		}
	}
	return nil, tts.ErrVoiceNotFound
}

// Synthesize converts text to speech.
func (p *TTSProvider) Synthesize(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
	req := oaiclient.TTSRequest{
		Input: text,
	}

	if config.Model != "" {
		req.Model = config.Model
	}
	if config.VoiceID != "" {
		req.Voice = config.VoiceID
	}
	if config.OutputFormat != "" {
		req.ResponseFormat = config.OutputFormat
	}
	if config.Speed > 0 {
		req.Speed = config.Speed
	}

	resp, err := p.client.Synthesize(ctx, req)
	if err != nil {
		return nil, err
	}

	return &tts.SynthesisResult{
		Audio:  resp.Audio,
		Format: resp.Format,
	}, nil
}

// SynthesizeStream converts text to speech with streaming output.
func (p *TTSProvider) SynthesizeStream(ctx context.Context, text string, config tts.SynthesisConfig) (<-chan tts.StreamChunk, error) {
	req := oaiclient.TTSRequest{
		Input: text,
	}

	if config.Model != "" {
		req.Model = config.Model
	}
	if config.VoiceID != "" {
		req.Voice = config.VoiceID
	}
	if config.OutputFormat != "" {
		req.ResponseFormat = config.OutputFormat
	}
	if config.Speed > 0 {
		req.Speed = config.Speed
	}

	reader, err := p.client.SynthesizeStream(ctx, req)
	if err != nil {
		return nil, err
	}

	// Create channel for streaming chunks
	chunkCh := make(chan tts.StreamChunk)

	// Read from the stream and send chunks
	go func() {
		defer close(chunkCh)
		defer func() {
			_ = reader.Close()
		}()

		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				chunkCh <- tts.StreamChunk{Error: ctx.Err()}
				return
			default:
			}

			n, err := reader.Read(buf)
			if n > 0 {
				// Copy the data to avoid buffer reuse issues
				data := make([]byte, n)
				copy(data, buf[:n])

				chunk := tts.StreamChunk{
					Audio: data,
				}

				select {
				case chunkCh <- chunk:
				case <-ctx.Done():
					chunkCh <- tts.StreamChunk{Error: ctx.Err()}
					return
				}
			}
			if err != nil {
				if err == io.EOF {
					// Send final chunk
					chunkCh <- tts.StreamChunk{IsFinal: true}
					return
				}
				chunkCh <- tts.StreamChunk{Error: err}
				return
			}
		}
	}()

	return chunkCh, nil
}

// Voice constants for OpenAI TTS.
const (
	VoiceAlloy   = oaiclient.VoiceAlloy
	VoiceAsh     = oaiclient.VoiceAsh
	VoiceBallad  = oaiclient.VoiceBallad
	VoiceCoral   = oaiclient.VoiceCoral
	VoiceEcho    = oaiclient.VoiceEcho
	VoiceFable   = oaiclient.VoiceFable
	VoiceOnyx    = oaiclient.VoiceOnyx
	VoiceNova    = oaiclient.VoiceNova
	VoiceSage    = oaiclient.VoiceSage
	VoiceShimmer = oaiclient.VoiceShimmer
	VoiceVerse   = oaiclient.VoiceVerse
	VoiceMarin   = oaiclient.VoiceMarin
	VoiceCedar   = oaiclient.VoiceCedar
)

// Model constants for OpenAI TTS.
const (
	ModelTTS1   = oaiclient.ModelTTS1
	ModelTTS1HD = oaiclient.ModelTTS1HD
)
