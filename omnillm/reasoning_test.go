package omnillm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/openai/openai-go/shared"
	core "github.com/plexusone/omnillm-core/provider"
)

func TestReasoningEffortConstants(t *testing.T) {
	// Verify that core constants can be converted to shared.ReasoningEffort
	tests := []struct {
		name   string
		effort string
	}{
		{"none", core.ReasoningEffortNone},
		{"low", core.ReasoningEffortLow},
		{"medium", core.ReasoningEffortMedium},
		{"high", core.ReasoningEffortHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify conversion works (shared.ReasoningEffort is a string type)
			got := shared.ReasoningEffort(tt.effort)
			if string(got) != tt.effort {
				t.Errorf("shared.ReasoningEffort(%q) = %q, want %q", tt.effort, got, tt.effort)
			}
		})
	}
}

func TestBuildParams_ReasoningEffort(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name   string
		effort string
	}{
		{"none", core.ReasoningEffortNone},
		{"low", core.ReasoningEffortLow},
		{"medium", core.ReasoningEffortMedium},
		{"high", core.ReasoningEffortHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effort := tt.effort
			req := &core.ChatCompletionRequest{
				Model: "o1-preview",
				Messages: []core.Message{
					{Role: core.RoleUser, Content: "test"},
				},
				ReasoningEffort: &effort,
			}

			params := p.buildParams(req)

			if params.ReasoningEffort != shared.ReasoningEffort(tt.effort) {
				t.Errorf("params.ReasoningEffort = %q, want %q", params.ReasoningEffort, tt.effort)
			}
		})
	}
}

func TestBuildParams_NoReasoningEffort(t *testing.T) {
	p := &Provider{}

	req := &core.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "test"},
		},
		// No ReasoningEffort set
	}

	params := p.buildParams(req)

	// ReasoningEffort should be empty/zero value when not set
	if params.ReasoningEffort != "" {
		t.Errorf("params.ReasoningEffort = %q, want empty string", params.ReasoningEffort)
	}
}

// Integration tests - require OPENAI_API_KEY

func TestIntegration_ReasoningEffort(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	p, err := New(Config{APIKey: apiKey})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test with o3-mini which supports reasoning_effort
	// Using low effort to minimize latency and cost
	// Note: o3-mini doesn't support max_tokens, only max_completion_tokens
	effort := core.ReasoningEffortLow
	req := &core.ChatCompletionRequest{
		Model:           "o3-mini",
		ReasoningEffort: &effort,
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "What is 2+2? Reply with just the number."},
		},
	}

	resp, err := p.CreateChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("CreateChatCompletion() error: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("CreateChatCompletion() returned no choices")
	}

	t.Logf("Response with reasoning_effort=%s: %s", effort, resp.Choices[0].Message.Content)
}

func TestIntegration_ReasoningEffort_Streaming(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	p, err := New(Config{APIKey: apiKey})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Note: o3-mini doesn't support max_tokens, only max_completion_tokens
	effort := core.ReasoningEffortLow
	req := &core.ChatCompletionRequest{
		Model:           "o3-mini",
		ReasoningEffort: &effort,
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "What is 2+2? Reply with just the number."},
		},
	}

	stream, err := p.CreateChatCompletionStream(ctx, req)
	if err != nil {
		t.Fatalf("CreateChatCompletionStream() error: %v", err)
	}
	defer stream.Close()

	var content string
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content += chunk.Choices[0].Delta.Content
		}
	}

	if content == "" {
		t.Error("Streaming returned no content")
	}

	t.Logf("Streaming response with reasoning_effort=%s: %s", effort, content)
}
