package realtime

import (
	"github.com/plexusone/omnivoice-core/registry"
)

// Provider-specific option functions for type-safe configuration via the registry.
// These functions return registry.ProviderOption and can be used with
// omnivoice.GetRealtimeProvider("openai", opts...).
//
// Note: These are named with "Registry" prefix to avoid conflicts with the
// provider-internal Option functions (which configure the local Config struct).

// WithRegistryTools sets the tools available to the realtime model via registry.
func WithRegistryTools(tools []Tool) registry.ProviderOption {
	return registry.WithExtension("tools", tools)
}

// WithRegistryTurnDetection sets the turn detection configuration via registry.
func WithRegistryTurnDetection(config *TurnDetectionConfig) registry.ProviderOption {
	return registry.WithExtension("turnDetection", config)
}

// WithRegistryInputAudioFormat sets the input audio format via registry.
func WithRegistryInputAudioFormat(format string) registry.ProviderOption {
	return registry.WithExtension("inputAudioFormat", format)
}

// WithRegistryOutputAudioFormat sets the output audio format via registry.
func WithRegistryOutputAudioFormat(format string) registry.ProviderOption {
	return registry.WithExtension("outputAudioFormat", format)
}

// WithRegistryModalities sets the modalities (e.g., ["text", "audio"]) via registry.
func WithRegistryModalities(modalities []string) registry.ProviderOption {
	return registry.WithExtension("modalities", modalities)
}

// WithRegistryTemperature sets the temperature for response generation via registry.
func WithRegistryTemperature(temp float64) registry.ProviderOption {
	return registry.WithExtension("temperature", temp)
}

// WithRegistryMaxResponseOutputTokens sets the maximum output tokens via registry.
func WithRegistryMaxResponseOutputTokens(tokens int) registry.ProviderOption {
	return registry.WithExtension("maxResponseOutputTokens", tokens)
}
