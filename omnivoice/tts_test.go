package omnivoice_test

import (
	"os"
	"testing"

	"github.com/plexusone/omnivoice-core/tts/providertest"

	"github.com/plexusone/omnivoice-openai/omnivoice"
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
