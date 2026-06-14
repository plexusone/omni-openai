package realtime

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("sk-test",
		WithModel("gpt-4o-realtime-preview-2024-12-17"),
		WithVoice(VoiceAlloy),
		WithInstructions("You are a helpful assistant."),
	)

	if client.config.APIKey != "sk-test" {
		t.Errorf("expected API key 'sk-test', got %q", client.config.APIKey)
	}
	if client.config.Model != "gpt-4o-realtime-preview-2024-12-17" {
		t.Errorf("expected model 'gpt-4o-realtime-preview-2024-12-17', got %q", client.config.Model)
	}
	if client.config.Voice != VoiceAlloy {
		t.Errorf("expected voice 'alloy', got %q", client.config.Voice)
	}
	if client.config.Instructions != "You are a helpful assistant." {
		t.Error("expected instructions to be set")
	}
}

func TestNewClientFromEnv(t *testing.T) {
	// Test with no env var
	os.Unsetenv("OPENAI_API_KEY")
	_, err := NewClientFromEnv()
	if err == nil {
		t.Error("expected error when OPENAI_API_KEY is not set")
	}

	// Test with env var
	os.Setenv("OPENAI_API_KEY", "sk-env-test")
	defer os.Unsetenv("OPENAI_API_KEY")

	client, err := NewClientFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.config.APIKey != "sk-env-test" {
		t.Errorf("expected API key 'sk-env-test', got %q", client.config.APIKey)
	}
}

func TestConfigDefaults(t *testing.T) {
	client := NewClient("sk-test")

	if client.config.Model != DefaultModel {
		t.Errorf("expected default model %q, got %q", DefaultModel, client.config.Model)
	}
	if client.config.Voice != DefaultVoice {
		t.Errorf("expected default voice %q, got %q", DefaultVoice, client.config.Voice)
	}
	if client.config.InputAudioFormat != DefaultInputAudioFormat {
		t.Errorf("expected default input format %q, got %q", DefaultInputAudioFormat, client.config.InputAudioFormat)
	}
	if client.config.OutputAudioFormat != DefaultOutputAudioFormat {
		t.Errorf("expected default output format %q, got %q", DefaultOutputAudioFormat, client.config.OutputAudioFormat)
	}
	if client.config.Temperature != DefaultTemperature {
		t.Errorf("expected default temperature %f, got %f", DefaultTemperature, client.config.Temperature)
	}
	if len(client.config.Modalities) != 2 {
		t.Errorf("expected 2 modalities, got %d", len(client.config.Modalities))
	}
}

func TestWithServerVAD(t *testing.T) {
	client := NewClient("sk-test", WithServerVAD())

	if client.config.TurnDetection == nil {
		t.Fatal("expected TurnDetection to be set")
	}
	if client.config.TurnDetection.Type != "server_vad" {
		t.Errorf("expected type 'server_vad', got %q", client.config.TurnDetection.Type)
	}
	if !client.config.TurnDetection.CreateResponse {
		t.Error("expected CreateResponse to be true")
	}
}

func TestWithNoTurnDetection(t *testing.T) {
	client := NewClient("sk-test", WithNoTurnDetection())

	if client.config.TurnDetection == nil {
		t.Fatal("expected TurnDetection to be set")
	}
	if client.config.TurnDetection.Type != "none" {
		t.Errorf("expected type 'none', got %q", client.config.TurnDetection.Type)
	}
}

func TestWithTools(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]string{"type": "string"},
		},
	})

	tool := Tool{
		Type:        "function",
		Name:        "search",
		Description: "Search for information",
		Parameters:  params,
	}

	client := NewClient("sk-test", WithTools(tool))

	if len(client.config.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(client.config.Tools))
	}
	if client.config.Tools[0].Name != "search" {
		t.Errorf("expected tool name 'search', got %q", client.config.Tools[0].Name)
	}
}

func TestParseServerEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantError bool
	}{
		{
			name:     "session.created",
			input:    `{"type":"session.created","event_id":"evt_123","session":{"id":"sess_123","model":"gpt-4o"}}`,
			wantType: TypeSessionCreated,
		},
		{
			name:     "error",
			input:    `{"type":"error","event_id":"evt_456","error":{"type":"invalid_request","message":"Bad request"}}`,
			wantType: TypeError,
		},
		{
			name:     "response.audio.delta",
			input:    `{"type":"response.audio.delta","event_id":"evt_789","delta":"AAAA","item_id":"item_1"}`,
			wantType: TypeResponseAudioDelta,
		},
		{
			name:     "response.text.delta",
			input:    `{"type":"response.text.delta","event_id":"evt_abc","delta":"Hello"}`,
			wantType: TypeResponseTextDelta,
		},
		{
			name:      "invalid json",
			input:     `{invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := parseServerEvent([]byte(tt.input))
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if event.GetType() != tt.wantType {
				t.Errorf("expected type %q, got %q", tt.wantType, event.GetType())
			}
		})
	}
}

func TestSessionCreatedEventParsing(t *testing.T) {
	input := `{
		"type": "session.created",
		"event_id": "evt_123",
		"session": {
			"id": "sess_ABC",
			"object": "realtime.session",
			"model": "gpt-4o-realtime-preview-2024-12-17",
			"modalities": ["text", "audio"],
			"voice": "alloy",
			"temperature": 0.8
		}
	}`

	event, err := parseServerEvent([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	created, ok := event.(*SessionCreatedEvent)
	if !ok {
		t.Fatalf("expected *SessionCreatedEvent, got %T", event)
	}

	if created.Session.ID != "sess_ABC" {
		t.Errorf("expected session ID 'sess_ABC', got %q", created.Session.ID)
	}
	if created.Session.Voice != "alloy" {
		t.Errorf("expected voice 'alloy', got %q", created.Session.Voice)
	}
	if created.Session.Temperature != 0.8 {
		t.Errorf("expected temperature 0.8, got %f", created.Session.Temperature)
	}
}

func TestErrorEventParsing(t *testing.T) {
	input := `{
		"type": "error",
		"event_id": "evt_error",
		"error": {
			"type": "invalid_request_error",
			"code": "invalid_value",
			"message": "Invalid voice specified",
			"param": "voice"
		}
	}`

	event, err := parseServerEvent([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errEvent, ok := event.(*ErrorEvent)
	if !ok {
		t.Fatalf("expected *ErrorEvent, got %T", event)
	}

	if errEvent.Error.Type != "invalid_request_error" {
		t.Errorf("expected error type 'invalid_request_error', got %q", errEvent.Error.Type)
	}
	if errEvent.Error.Message != "Invalid voice specified" {
		t.Errorf("expected message 'Invalid voice specified', got %q", errEvent.Error.Message)
	}
	if errEvent.Error.Param != "voice" {
		t.Errorf("expected param 'voice', got %q", errEvent.Error.Param)
	}
}

func TestAudioDeltaEventParsing(t *testing.T) {
	input := `{
		"type": "response.audio.delta",
		"event_id": "evt_audio",
		"response_id": "resp_123",
		"item_id": "item_456",
		"output_index": 0,
		"content_index": 0,
		"delta": "SGVsbG8gV29ybGQ="
	}`

	event, err := parseServerEvent([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	audioEvent, ok := event.(*ResponseAudioDeltaEvent)
	if !ok {
		t.Fatalf("expected *ResponseAudioDeltaEvent, got %T", event)
	}

	if audioEvent.ResponseID != "resp_123" {
		t.Errorf("expected response_id 'resp_123', got %q", audioEvent.ResponseID)
	}
	if audioEvent.ItemID != "item_456" {
		t.Errorf("expected item_id 'item_456', got %q", audioEvent.ItemID)
	}
	if audioEvent.Delta != "SGVsbG8gV29ybGQ=" {
		t.Errorf("expected delta 'SGVsbG8gV29ybGQ=', got %q", audioEvent.Delta)
	}
}

func TestFunctionCallEventParsing(t *testing.T) {
	input := `{
		"type": "response.function_call_arguments.done",
		"event_id": "evt_func",
		"response_id": "resp_123",
		"item_id": "item_789",
		"output_index": 0,
		"call_id": "call_ABC",
		"name": "get_weather",
		"arguments": "{\"location\":\"San Francisco\"}"
	}`

	event, err := parseServerEvent([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	funcEvent, ok := event.(*ResponseFunctionCallArgumentsDoneEvent)
	if !ok {
		t.Fatalf("expected *ResponseFunctionCallArgumentsDoneEvent, got %T", event)
	}

	if funcEvent.Name != "get_weather" {
		t.Errorf("expected name 'get_weather', got %q", funcEvent.Name)
	}
	if funcEvent.CallID != "call_ABC" {
		t.Errorf("expected call_id 'call_ABC', got %q", funcEvent.CallID)
	}
	if funcEvent.Arguments != `{"location":"San Francisco"}` {
		t.Errorf("unexpected arguments: %q", funcEvent.Arguments)
	}
}

func TestNewProvider(t *testing.T) {
	provider := NewProvider("sk-test",
		WithVoice(VoiceEcho),
		WithInstructions("Test instructions"),
	)

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if provider.Name() != "openai-realtime" {
		t.Errorf("expected name 'openai-realtime', got %q", provider.Name())
	}
}

// Integration test - only runs when OPENAI_API_KEY is set
func TestIntegration_Connect(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	// This test would actually connect to OpenAI
	// Skipped by default to avoid API charges
	t.Skip("Integration test skipped to avoid API charges")
}
