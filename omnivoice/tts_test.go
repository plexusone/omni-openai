package omnivoice_test

import (
	"context"
	"os"
	"testing"

	"github.com/plexusone/omnivoice-core/tts"
	"github.com/plexusone/omnivoice-core/tts/providertest"

	"github.com/plexusone/omni-openai/omnivoice"
)

func TestTTSConformance(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")

	provider := omnivoice.NewTTSProvider(apiKey)

	providertest.RunAll(t, providertest.Config{
		Provider:        provider,
		SkipIntegration: apiKey == "",
		TestVoiceID:     omnivoice.VoiceNova,
		TestText:        "Hello, this is a test of the OpenAI text to speech system.",
	})
}

func TestSynthesizeMuLawIntegration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	provider := omnivoice.NewTTSProvider(apiKey)
	config := tts.SynthesisConfig{VoiceID: omnivoice.VoiceAlloy, OutputFormat: "mulaw", SampleRate: 8000}

	result, err := provider.Synthesize(context.Background(), "Hello from a phone call.", config)
	if err != nil {
		t.Fatalf("Synthesize() error: %v", err)
	}
	if result.Format != "mulaw" || result.SampleRate != 8000 || len(result.Audio) < 4000 {
		t.Errorf("result = %s at %d Hz, %d bytes; want at least half a second of 8 kHz mu-law",
			result.Format, result.SampleRate, len(result.Audio))
	}

	chunks, err := provider.SynthesizeStream(context.Background(), "Hello from a phone call.", config)
	if err != nil {
		t.Fatalf("SynthesizeStream() error: %v", err)
	}
	streamed := 0
	for chunk := range chunks {
		if chunk.Error != nil {
			t.Fatalf("stream error: %v", chunk.Error)
		}
		streamed += len(chunk.Audio)
	}
	if streamed < 4000 {
		t.Errorf("streamed %d bytes of mu-law, want at least half a second", streamed)
	}
	t.Logf("synthesized %d bytes, streamed %d bytes of 8 kHz mu-law", len(result.Audio), streamed)
}
