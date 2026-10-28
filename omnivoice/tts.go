// Package omnivoice provides omnivoice STT and TTS provider implementations for OpenAI.
package omnivoice

import (
	"context"
	"io"

	"github.com/plexusone/omnivoice-core/tts"

	oaiclient "github.com/plexusone/omni-openai"
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
//
// Raw output formats are produced from OpenAI's 24 kHz PCM: "pcm" or
// "linear16" at another SampleRate is resampled, and "mulaw" and "alaw" are
// encoded, at 8 kHz unless SampleRate says otherwise. The result reports the
// sample rate of raw audio.
func (p *TTSProvider) Synthesize(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
	plan := planOutput(config.OutputFormat, config.SampleRate)
	resp, err := p.client.Synthesize(ctx, ttsRequest(text, config, plan))
	if err != nil {
		return nil, err
	}

	audio, err := plan.convertAudio(resp.Audio)
	if err != nil {
		return nil, err
	}
	result := &tts.SynthesisResult{
		Audio:      audio,
		Format:     resp.Format,
		SampleRate: plan.sampleRate,
	}
	if plan.resultFormat != "" {
		result.Format = plan.resultFormat
	}
	return result, nil
}

// ttsRequest builds the API request for a synthesis.
func ttsRequest(text string, config tts.SynthesisConfig, plan outputPlan) oaiclient.TTSRequest {
	req := oaiclient.TTSRequest{
		Input:          text,
		Model:          config.Model,
		Voice:          config.VoiceID,
		ResponseFormat: plan.apiFormat,
	}
	if config.Speed > 0 {
		req.Speed = config.Speed
	}
	return req
}

// SynthesizeStream converts text to speech with streaming output, in the
// same output formats as Synthesize.
func (p *TTSProvider) SynthesizeStream(ctx context.Context, text string, config tts.SynthesisConfig) (<-chan tts.StreamChunk, error) {
	plan := planOutput(config.OutputFormat, config.SampleRate)
	conv := &streamConverter{plan: plan}
	reader, err := p.client.SynthesizeStream(ctx, ttsRequest(text, config, plan))
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

				audio, convErr := conv.convert(data)
				if convErr != nil {
					chunkCh <- tts.StreamChunk{Error: convErr}
					return
				}
				if len(audio) > 0 {
					select {
					case chunkCh <- tts.StreamChunk{Audio: audio}:
					case <-ctx.Done():
						chunkCh <- tts.StreamChunk{Error: ctx.Err()}
						return
					}
				}
			}
			if err != nil {
				if err == io.EOF {
					// Send what conversion held back, then the final chunk.
					rest, convErr := conv.flush()
					if convErr != nil {
						chunkCh <- tts.StreamChunk{Error: convErr}
						return
					}
					chunkCh <- tts.StreamChunk{Audio: rest, IsFinal: true}
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
