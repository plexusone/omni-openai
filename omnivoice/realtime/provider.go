package realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"

	corereal "github.com/plexusone/omnivoice-core/realtime"
)

// RealtimeProvider provides native voice-to-voice processing via OpenAI Realtime API.
// It implements the [corereal.Provider] interface.
type RealtimeProvider struct {
	client *Client
	config Config
}

// Ensure RealtimeProvider implements corereal.Provider.
var _ corereal.Provider = (*RealtimeProvider)(nil)

// NewProvider creates a new RealtimeProvider.
func NewProvider(apiKey string, opts ...Option) *RealtimeProvider {
	return &RealtimeProvider{
		client: NewClient(apiKey, opts...),
		config: Config{APIKey: apiKey},
	}
}

// ProcessAudioStream processes audio input and returns audio output.
// This provides native voice-to-voice with ~100ms latency.
// Implements [corereal.Provider].
func (p *RealtimeProvider) ProcessAudioStream(
	ctx context.Context,
	audioIn <-chan []byte,
	config corereal.ProcessConfig,
) (<-chan corereal.AudioChunk, <-chan corereal.Transcript, error) {
	// Apply config overrides
	opts := []Option{}
	if config.Instructions != "" {
		opts = append(opts, WithInstructions(config.Instructions))
	}
	if config.Voice != "" {
		opts = append(opts, WithVoice(config.Voice))
	}
	if len(config.Functions) > 0 {
		// Convert corereal.FunctionDeclaration to local Tool
		tools := make([]Tool, len(config.Functions))
		for i, f := range config.Functions {
			tools[i] = Tool{
				Type:        "function",
				Name:        f.Name,
				Description: f.Description,
				Parameters:  f.Parameters,
			}
		}
		opts = append(opts, WithTools(tools...))
	}

	// Create client with overrides
	client := NewClient(p.config.APIKey, opts...)

	session, err := client.Connect(ctx)
	if err != nil {
		return nil, nil, err
	}

	audioCh := make(chan corereal.AudioChunk, 100)
	transcriptCh := make(chan corereal.Transcript, 100)

	var wg sync.WaitGroup
	wg.Add(2)

	// Send audio to session
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case audio, ok := <-audioIn:
				if !ok {
					return
				}
				_ = session.SendAudio(audio)
			}
		}
	}()

	// Process events from session
	go func() {
		defer wg.Done()
		defer close(audioCh)
		defer close(transcriptCh)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-session.Events():
				if !ok {
					return
				}

				switch e := event.(type) {
				case *ResponseAudioDeltaEvent:
					audio, err := base64.StdEncoding.DecodeString(e.Delta)
					if err == nil && len(audio) > 0 {
						select {
						case audioCh <- corereal.AudioChunk{Audio: audio}:
						case <-ctx.Done():
							return
						}
					}

				case *ResponseAudioDoneEvent:
					select {
					case audioCh <- corereal.AudioChunk{IsFinal: true}:
					case <-ctx.Done():
						return
					}

				case *ResponseAudioTranscriptDeltaEvent:
					select {
					case transcriptCh <- corereal.Transcript{Text: e.Delta, IsInput: false}:
					case <-ctx.Done():
						return
					}

				case *ResponseAudioTranscriptDoneEvent:
					select {
					case transcriptCh <- corereal.Transcript{Text: e.Transcript, IsFinal: true, IsInput: false}:
					case <-ctx.Done():
						return
					}

				case *ConversationItemInputAudioTranscriptionCompletedEvent:
					select {
					case transcriptCh <- corereal.Transcript{Text: e.Transcript, IsFinal: true, IsInput: true}:
					case <-ctx.Done():
						return
					}

				case *ResponseFunctionCallArgumentsDoneEvent:
					if config.OnFunctionCall != nil {
						result, err := config.OnFunctionCall(e.CallID, e.Name, e.Arguments)
						if err != nil {
							_ = session.SendFunctionOutput(e.CallID, `{"error":"`+err.Error()+`"}`)
						} else {
							output, _ := json.Marshal(result)
							_ = session.SendFunctionOutput(e.CallID, string(output))
						}
						// Request a new response after function output
						_ = session.CreateResponse(nil)
					}

				case *ErrorEvent:
					// Could send error through a dedicated channel
					continue
				}
			}
		}
	}()

	// Close session when done
	go func() {
		wg.Wait()
		session.Close()
	}()

	return audioCh, transcriptCh, nil
}

// Name returns the provider name.
// Implements [corereal.Provider].
func (p *RealtimeProvider) Name() string {
	return "openai-realtime"
}

// Close releases any resources held by the provider.
// Implements [corereal.Provider].
func (p *RealtimeProvider) Close() error {
	// No persistent resources to clean up
	return nil
}
