package realtime

import (
	"github.com/plexusone/omnivoice-core/gateway"
	corereal "github.com/plexusone/omnivoice-core/realtime"
)

// Factory creates OpenAI Realtime providers from gateway configuration.
// It implements [gateway.RealtimeProviderFactory].
type Factory struct{}

// NewFactory creates a new OpenAI realtime provider factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Ensure Factory implements gateway.RealtimeProviderFactory.
var _ gateway.RealtimeProviderFactory = (*Factory)(nil)

// Create creates an OpenAI RealtimeProvider from the given configuration.
func (f *Factory) Create(config *gateway.RealtimeConfig) (corereal.Provider, error) {
	if config.APIKey == "" {
		return nil, ErrNoAPIKey
	}

	opts := []Option{}

	if config.Model != "" {
		opts = append(opts, WithModel(config.Model))
	}
	if config.Voice != "" {
		opts = append(opts, WithVoice(config.Voice))
	}
	if config.Instructions != "" {
		opts = append(opts, WithInstructions(config.Instructions))
	}
	if config.Temperature > 0 {
		opts = append(opts, WithTemperature(config.Temperature))
	}

	// Convert gateway functions to local tools
	if len(config.Functions) > 0 {
		tools := make([]Tool, len(config.Functions))
		for i, fn := range config.Functions {
			tools[i] = Tool{
				Type:        "function",
				Name:        fn.Name,
				Description: fn.Description,
				Parameters:  fn.Parameters,
			}
		}
		opts = append(opts, WithTools(tools...))
	}

	return NewProvider(config.APIKey, opts...), nil
}

// Name returns the provider name.
func (f *Factory) Name() string {
	return "openai"
}

// ProviderName is the name used to identify OpenAI realtime provider.
const ProviderName = "openai"
