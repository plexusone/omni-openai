package realtime

import "time"

const (
	// DefaultModel is the default realtime model.
	DefaultModel = "gpt-4o-realtime-preview-2024-12-17"

	// DefaultVoice is the default voice.
	DefaultVoice = "alloy"

	// DefaultInputAudioFormat is the default input audio format.
	DefaultInputAudioFormat = "pcm16"

	// DefaultOutputAudioFormat is the default output audio format.
	DefaultOutputAudioFormat = "pcm16"

	// DefaultSampleRate is the native sample rate for PCM16 (24kHz).
	DefaultSampleRate = 24000

	// DefaultTemperature is the default temperature.
	DefaultTemperature = 0.8

	// DefaultConnectTimeout is the default connection timeout.
	DefaultConnectTimeout = 30 * time.Second
)

// Available voices for the Realtime API.
const (
	VoiceAlloy   = "alloy"
	VoiceEcho    = "echo"
	VoiceShimmer = "shimmer"
	VoiceAsh     = "ash"
	VoiceBallad  = "ballad"
	VoiceCoral   = "coral"
	VoiceSage    = "sage"
	VoiceVerse   = "verse"
)

// Audio formats supported by the Realtime API.
const (
	AudioFormatPCM16    = "pcm16"     // 24kHz mono PCM16 (raw)
	AudioFormatG711Ulaw = "g711_ulaw" // 8kHz mu-law
	AudioFormatG711Alaw = "g711_alaw" // 8kHz a-law
)

// Config holds client configuration.
type Config struct {
	// APIKey is the OpenAI API key.
	APIKey string

	// Model is the realtime model to use.
	Model string

	// Voice is the voice for audio output.
	Voice string

	// Instructions is the system prompt.
	Instructions string

	// Modalities specifies the modalities to use (["text", "audio"]).
	Modalities []string

	// InputAudioFormat is the format for input audio.
	InputAudioFormat string

	// OutputAudioFormat is the format for output audio.
	OutputAudioFormat string

	// TurnDetection configures voice activity detection.
	TurnDetection *TurnDetectionConfig

	// Tools are functions the model can call.
	Tools []Tool

	// Temperature controls randomness (0.6-1.2).
	Temperature float64

	// MaxResponseOutputTokens limits response length.
	MaxResponseOutputTokens any

	// ConnectTimeout is the timeout for connecting.
	ConnectTimeout time.Duration
}

// Option configures the client.
type Option func(*Config)

// WithModel sets the realtime model.
func WithModel(model string) Option {
	return func(c *Config) {
		c.Model = model
	}
}

// WithVoice sets the voice for audio output.
func WithVoice(voice string) Option {
	return func(c *Config) {
		c.Voice = voice
	}
}

// WithInstructions sets the system prompt.
func WithInstructions(instructions string) Option {
	return func(c *Config) {
		c.Instructions = instructions
	}
}

// WithModalities sets the modalities to use.
func WithModalities(modalities ...string) Option {
	return func(c *Config) {
		c.Modalities = modalities
	}
}

// WithInputAudioFormat sets the input audio format.
func WithInputAudioFormat(format string) Option {
	return func(c *Config) {
		c.InputAudioFormat = format
	}
}

// WithOutputAudioFormat sets the output audio format.
func WithOutputAudioFormat(format string) Option {
	return func(c *Config) {
		c.OutputAudioFormat = format
	}
}

// WithTurnDetection configures voice activity detection.
func WithTurnDetection(config *TurnDetectionConfig) Option {
	return func(c *Config) {
		c.TurnDetection = config
	}
}

// WithServerVAD enables server-side voice activity detection.
func WithServerVAD() Option {
	return func(c *Config) {
		c.TurnDetection = &TurnDetectionConfig{
			Type:              "server_vad",
			Threshold:         0.5,
			PrefixPaddingMs:   300,
			SilenceDurationMs: 500,
			CreateResponse:    true,
		}
	}
}

// WithNoTurnDetection disables automatic turn detection.
func WithNoTurnDetection() Option {
	return func(c *Config) {
		c.TurnDetection = &TurnDetectionConfig{
			Type: "none",
		}
	}
}

// WithTools sets the tools the model can call.
func WithTools(tools ...Tool) Option {
	return func(c *Config) {
		c.Tools = tools
	}
}

// WithTemperature sets the temperature (0.6-1.2).
func WithTemperature(temp float64) Option {
	return func(c *Config) {
		c.Temperature = temp
	}
}

// WithMaxTokens sets the maximum response tokens.
func WithMaxTokens(max int) Option {
	return func(c *Config) {
		c.MaxResponseOutputTokens = max
	}
}

// WithConnectTimeout sets the connection timeout.
func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ConnectTimeout = timeout
	}
}

// applyDefaults applies default values to the config.
func applyDefaults(c *Config) {
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.Voice == "" {
		c.Voice = DefaultVoice
	}
	if c.InputAudioFormat == "" {
		c.InputAudioFormat = DefaultInputAudioFormat
	}
	if c.OutputAudioFormat == "" {
		c.OutputAudioFormat = DefaultOutputAudioFormat
	}
	if len(c.Modalities) == 0 {
		c.Modalities = []string{"text", "audio"}
	}
	if c.Temperature == 0 {
		c.Temperature = DefaultTemperature
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = DefaultConnectTimeout
	}
}
