package realtime

import (
	"fmt"

	omnivoice "github.com/plexusone/omnivoice-core"
	"github.com/plexusone/omnivoice-core/registry"
)

func init() {
	omnivoice.RegisterRealtimeProvider("openai", NewRealtimeProvider, omnivoice.PriorityThick)
}

// NewRealtimeProvider creates an OpenAI realtime provider from registry config.
func NewRealtimeProvider(cfg registry.ProviderConfig) (registry.RealtimeProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai realtime: apiKey is required")
	}

	opts := []Option{}

	if v := getExtString(cfg.Extensions, "model"); v != "" {
		opts = append(opts, WithModel(v))
	}
	if v := getExtString(cfg.Extensions, "voice"); v != "" {
		opts = append(opts, WithVoice(v))
	}
	if v := getExtString(cfg.Extensions, "instructions"); v != "" {
		opts = append(opts, WithInstructions(v))
	}

	provider := NewProvider(cfg.APIKey, opts...)
	return &realtimeWrapper{provider}, nil
}

// realtimeWrapper wraps RealtimeProvider to implement registry.RealtimeProvider.
type realtimeWrapper struct {
	p *RealtimeProvider
}

func (w *realtimeWrapper) Name() string {
	return ProviderName
}

func (w *realtimeWrapper) Close() error {
	// RealtimeProvider doesn't have a Close method, but individual sessions do.
	// This is a no-op at the provider level.
	return nil
}

// Provider returns the underlying RealtimeProvider for full API access.
func (w *realtimeWrapper) Provider() *RealtimeProvider {
	return w.p
}

func getExtString(ext map[string]any, key string) string {
	if ext == nil {
		return ""
	}
	if v, ok := ext[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
