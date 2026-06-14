package realtime

import "encoding/json"

// Event types for client-sent messages.
const (
	TypeSessionUpdate            = "session.update"
	TypeInputAudioBufferAppend   = "input_audio_buffer.append"
	TypeInputAudioBufferCommit   = "input_audio_buffer.commit"
	TypeInputAudioBufferClear    = "input_audio_buffer.clear"
	TypeConversationItemCreate   = "conversation.item.create"
	TypeConversationItemTruncate = "conversation.item.truncate"
	TypeConversationItemDelete   = "conversation.item.delete"
	TypeResponseCreate           = "response.create"
	TypeResponseCancel           = "response.cancel"
)

// Event types for server-sent messages.
const (
	TypeError                                            = "error"
	TypeSessionCreated                                   = "session.created"
	TypeSessionUpdated                                   = "session.updated"
	TypeConversationCreated                              = "conversation.created"
	TypeInputAudioBufferCommitted                        = "input_audio_buffer.committed"
	TypeInputAudioBufferCleared                          = "input_audio_buffer.cleared"
	TypeInputAudioBufferSpeechStarted                    = "input_audio_buffer.speech_started"
	TypeInputAudioBufferSpeechStopped                    = "input_audio_buffer.speech_stopped"
	TypeConversationItemCreated                          = "conversation.item.created"
	TypeConversationItemInputAudioTranscriptionCompleted = "conversation.item.input_audio_transcription.completed"
	TypeConversationItemInputAudioTranscriptionFailed    = "conversation.item.input_audio_transcription.failed"
	TypeConversationItemTruncated                        = "conversation.item.truncated"
	TypeConversationItemDeleted                          = "conversation.item.deleted"
	TypeResponseCreated                                  = "response.created"
	TypeResponseDone                                     = "response.done"
	TypeResponseOutputItemAdded                          = "response.output_item.added"
	TypeResponseOutputItemDone                           = "response.output_item.done"
	TypeResponseContentPartAdded                         = "response.content_part.added"
	TypeResponseContentPartDone                          = "response.content_part.done"
	TypeResponseTextDelta                                = "response.text.delta"
	TypeResponseTextDone                                 = "response.text.done"
	TypeResponseAudioTranscriptDelta                     = "response.audio_transcript.delta"
	TypeResponseAudioTranscriptDone                      = "response.audio_transcript.done"
	TypeResponseAudioDelta                               = "response.audio.delta"
	TypeResponseAudioDone                                = "response.audio.done"
	TypeResponseFunctionCallArgumentsDelta               = "response.function_call_arguments.delta"
	TypeResponseFunctionCallArgumentsDone                = "response.function_call_arguments.done"
	TypeRateLimitsUpdated                                = "rate_limits.updated"
)

// ServerEvent is the base interface for all server events.
type ServerEvent interface {
	GetType() string
	GetEventID() string
}

// baseEvent contains common fields for all events.
type baseEvent struct {
	EventID string `json:"event_id,omitempty"`
	Type    string `json:"type"`
}

func (e *baseEvent) GetEventID() string { return e.EventID }
func (e *baseEvent) GetType() string    { return e.Type }

// ErrorEvent is sent when an error occurs.
type ErrorEvent struct {
	baseEvent
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message"`
		Param   string `json:"param,omitempty"`
	} `json:"error"`
}

// SessionConfig configures the realtime session.
type SessionConfig struct {
	Modalities              []string             `json:"modalities,omitempty"`
	Instructions            string               `json:"instructions,omitempty"`
	Voice                   string               `json:"voice,omitempty"`
	InputAudioFormat        string               `json:"input_audio_format,omitempty"`
	OutputAudioFormat       string               `json:"output_audio_format,omitempty"`
	InputAudioTranscription *TranscriptionConfig `json:"input_audio_transcription,omitempty"`
	TurnDetection           *TurnDetectionConfig `json:"turn_detection,omitempty"`
	Tools                   []Tool               `json:"tools,omitempty"`
	ToolChoice              string               `json:"tool_choice,omitempty"`
	Temperature             float64              `json:"temperature,omitempty"`
	MaxResponseOutputTokens any                  `json:"max_response_output_tokens,omitempty"`
}

// TranscriptionConfig configures input audio transcription.
type TranscriptionConfig struct {
	Model string `json:"model,omitempty"`
}

// TurnDetectionConfig configures voice activity detection.
type TurnDetectionConfig struct {
	Type              string  `json:"type,omitempty"` // "server_vad" or "none"
	Threshold         float64 `json:"threshold,omitempty"`
	PrefixPaddingMs   int     `json:"prefix_padding_ms,omitempty"`
	SilenceDurationMs int     `json:"silence_duration_ms,omitempty"`
	CreateResponse    bool    `json:"create_response,omitempty"`
}

// Tool defines a function the model can call.
type Tool struct {
	Type        string          `json:"type"` // "function"
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// SessionCreatedEvent is sent when the session is created.
type SessionCreatedEvent struct {
	baseEvent
	Session struct {
		ID                      string               `json:"id"`
		Object                  string               `json:"object"`
		Model                   string               `json:"model"`
		Modalities              []string             `json:"modalities"`
		Instructions            string               `json:"instructions"`
		Voice                   string               `json:"voice"`
		InputAudioFormat        string               `json:"input_audio_format"`
		OutputAudioFormat       string               `json:"output_audio_format"`
		InputAudioTranscription *TranscriptionConfig `json:"input_audio_transcription"`
		TurnDetection           *TurnDetectionConfig `json:"turn_detection"`
		Tools                   []Tool               `json:"tools"`
		ToolChoice              string               `json:"tool_choice"`
		Temperature             float64              `json:"temperature"`
		MaxResponseOutputTokens any                  `json:"max_response_output_tokens"`
	} `json:"session"`
}

// SessionUpdatedEvent is sent when the session is updated.
type SessionUpdatedEvent struct {
	baseEvent
	Session SessionConfig `json:"session"`
}

// ConversationCreatedEvent is sent when a conversation is created.
type ConversationCreatedEvent struct {
	baseEvent
	Conversation struct {
		ID     string `json:"id"`
		Object string `json:"object"`
	} `json:"conversation"`
}

// InputAudioBufferSpeechStartedEvent is sent when speech is detected.
type InputAudioBufferSpeechStartedEvent struct {
	baseEvent
	AudioStartMs int    `json:"audio_start_ms"`
	ItemID       string `json:"item_id"`
}

// InputAudioBufferSpeechStoppedEvent is sent when speech ends.
type InputAudioBufferSpeechStoppedEvent struct {
	baseEvent
	AudioEndMs int    `json:"audio_end_ms"`
	ItemID     string `json:"item_id"`
}

// InputAudioBufferCommittedEvent is sent when audio buffer is committed.
type InputAudioBufferCommittedEvent struct {
	baseEvent
	PreviousItemID string `json:"previous_item_id"`
	ItemID         string `json:"item_id"`
}

// ConversationItem represents an item in the conversation.
type ConversationItem struct {
	ID        string        `json:"id"`
	Object    string        `json:"object"`
	Type      string        `json:"type"` // "message", "function_call", "function_call_output"
	Status    string        `json:"status"`
	Role      string        `json:"role"` // "user", "assistant", "system"
	Content   []ContentPart `json:"content,omitempty"`
	CallID    string        `json:"call_id,omitempty"`
	Name      string        `json:"name,omitempty"`
	Arguments string        `json:"arguments,omitempty"`
	Output    string        `json:"output,omitempty"`
}

// ContentPart represents a part of content.
type ContentPart struct {
	Type       string `json:"type"` // "input_text", "input_audio", "text", "audio"
	Text       string `json:"text,omitempty"`
	Audio      string `json:"audio,omitempty"` // base64
	Transcript string `json:"transcript,omitempty"`
}

// ConversationItemCreatedEvent is sent when an item is created.
type ConversationItemCreatedEvent struct {
	baseEvent
	PreviousItemID string           `json:"previous_item_id"`
	Item           ConversationItem `json:"item"`
}

// ResponseCreatedEvent is sent when a response starts.
type ResponseCreatedEvent struct {
	baseEvent
	Response Response `json:"response"`
}

// Response represents a model response.
type Response struct {
	ID            string `json:"id"`
	Object        string `json:"object"`
	Status        string `json:"status"`
	StatusDetails any    `json:"status_details,omitempty"`
	Output        []any  `json:"output"`
	Usage         *Usage `json:"usage,omitempty"`
}

// Usage contains token usage information.
type Usage struct {
	TotalTokens       int `json:"total_tokens"`
	InputTokens       int `json:"input_tokens"`
	OutputTokens      int `json:"output_tokens"`
	InputTokenDetails struct {
		CachedTokens int `json:"cached_tokens"`
		TextTokens   int `json:"text_tokens"`
		AudioTokens  int `json:"audio_tokens"`
	} `json:"input_token_details,omitempty"`
	OutputTokenDetails struct {
		TextTokens  int `json:"text_tokens"`
		AudioTokens int `json:"audio_tokens"`
	} `json:"output_token_details,omitempty"`
}

// ResponseDoneEvent is sent when a response completes.
type ResponseDoneEvent struct {
	baseEvent
	Response Response `json:"response"`
}

// ResponseAudioDeltaEvent is sent for audio output chunks.
type ResponseAudioDeltaEvent struct {
	baseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	ContentIndex int    `json:"content_index"`
	Delta        string `json:"delta"` // base64 audio
}

// ResponseAudioDoneEvent is sent when audio output completes.
type ResponseAudioDoneEvent struct {
	baseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	ContentIndex int    `json:"content_index"`
}

// ResponseTextDeltaEvent is sent for text output chunks.
type ResponseTextDeltaEvent struct {
	baseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	ContentIndex int    `json:"content_index"`
	Delta        string `json:"delta"`
}

// ResponseTextDoneEvent is sent when text output completes.
type ResponseTextDoneEvent struct {
	baseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	ContentIndex int    `json:"content_index"`
	Text         string `json:"text"`
}

// ResponseAudioTranscriptDeltaEvent is sent for transcript chunks.
type ResponseAudioTranscriptDeltaEvent struct {
	baseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	ContentIndex int    `json:"content_index"`
	Delta        string `json:"delta"`
}

// ResponseAudioTranscriptDoneEvent is sent when transcript completes.
type ResponseAudioTranscriptDoneEvent struct {
	baseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	ContentIndex int    `json:"content_index"`
	Transcript   string `json:"transcript"`
}

// ResponseFunctionCallArgumentsDeltaEvent is sent for function call argument chunks.
type ResponseFunctionCallArgumentsDeltaEvent struct {
	baseEvent
	ResponseID  string `json:"response_id"`
	ItemID      string `json:"item_id"`
	OutputIndex int    `json:"output_index"`
	CallID      string `json:"call_id"`
	Delta       string `json:"delta"`
}

// ResponseFunctionCallArgumentsDoneEvent is sent when function call completes.
type ResponseFunctionCallArgumentsDoneEvent struct {
	baseEvent
	ResponseID  string `json:"response_id"`
	ItemID      string `json:"item_id"`
	OutputIndex int    `json:"output_index"`
	CallID      string `json:"call_id"`
	Name        string `json:"name"`
	Arguments   string `json:"arguments"`
}

// ConversationItemInputAudioTranscriptionCompletedEvent is sent when input transcription completes.
type ConversationItemInputAudioTranscriptionCompletedEvent struct {
	baseEvent
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	Transcript   string `json:"transcript"`
}

// RateLimitsUpdatedEvent is sent when rate limits change.
type RateLimitsUpdatedEvent struct {
	baseEvent
	RateLimits []RateLimit `json:"rate_limits"`
}

// RateLimit represents a rate limit.
type RateLimit struct {
	Name         string `json:"name"`
	Limit        int    `json:"limit"`
	Remaining    int    `json:"remaining"`
	ResetSeconds int    `json:"reset_seconds"`
}

// Client events (sent from client to server)

// SessionUpdateEvent updates the session configuration.
type SessionUpdateEvent struct {
	Type    string        `json:"type"` // "session.update"
	Session SessionConfig `json:"session"`
}

// InputAudioBufferAppendEvent appends audio to the input buffer.
type InputAudioBufferAppendEvent struct {
	Type  string `json:"type"`  // "input_audio_buffer.append"
	Audio string `json:"audio"` // base64 PCM16 24kHz mono
}

// InputAudioBufferCommitEvent commits the audio buffer.
type InputAudioBufferCommitEvent struct {
	Type string `json:"type"` // "input_audio_buffer.commit"
}

// InputAudioBufferClearEvent clears the audio buffer.
type InputAudioBufferClearEvent struct {
	Type string `json:"type"` // "input_audio_buffer.clear"
}

// ResponseCreateEvent requests a response from the model.
type ResponseCreateEvent struct {
	Type     string          `json:"type"` // "response.create"
	Response *ResponseConfig `json:"response,omitempty"`
}

// ResponseConfig configures the response.
type ResponseConfig struct {
	Modalities              []string `json:"modalities,omitempty"`
	Instructions            string   `json:"instructions,omitempty"`
	Voice                   string   `json:"voice,omitempty"`
	OutputAudioFormat       string   `json:"output_audio_format,omitempty"`
	Tools                   []Tool   `json:"tools,omitempty"`
	ToolChoice              string   `json:"tool_choice,omitempty"`
	Temperature             float64  `json:"temperature,omitempty"`
	MaxResponseOutputTokens any      `json:"max_response_output_tokens,omitempty"`
}

// ResponseCancelEvent cancels an in-progress response.
type ResponseCancelEvent struct {
	Type string `json:"type"` // "response.cancel"
}

// ConversationItemCreateEvent creates a conversation item.
type ConversationItemCreateEvent struct {
	Type           string           `json:"type"` // "conversation.item.create"
	PreviousItemID string           `json:"previous_item_id,omitempty"`
	Item           ConversationItem `json:"item"`
}

// ConversationItemDeleteEvent deletes a conversation item.
type ConversationItemDeleteEvent struct {
	Type   string `json:"type"` // "conversation.item.delete"
	ItemID string `json:"item_id"`
}

// FunctionCallOutputEvent provides output for a function call.
type FunctionCallOutputEvent struct {
	Type string `json:"type"` // "conversation.item.create"
	Item struct {
		Type   string `json:"type"` // "function_call_output"
		CallID string `json:"call_id"`
		Output string `json:"output"`
	} `json:"item"`
}

// parseServerEvent parses a raw JSON message into a typed event.
func parseServerEvent(data []byte) (ServerEvent, error) {
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	var event ServerEvent
	switch base.Type {
	case TypeError:
		event = &ErrorEvent{}
	case TypeSessionCreated:
		event = &SessionCreatedEvent{}
	case TypeSessionUpdated:
		event = &SessionUpdatedEvent{}
	case TypeConversationCreated:
		event = &ConversationCreatedEvent{}
	case TypeInputAudioBufferSpeechStarted:
		event = &InputAudioBufferSpeechStartedEvent{}
	case TypeInputAudioBufferSpeechStopped:
		event = &InputAudioBufferSpeechStoppedEvent{}
	case TypeInputAudioBufferCommitted:
		event = &InputAudioBufferCommittedEvent{}
	case TypeConversationItemCreated:
		event = &ConversationItemCreatedEvent{}
	case TypeConversationItemInputAudioTranscriptionCompleted:
		event = &ConversationItemInputAudioTranscriptionCompletedEvent{}
	case TypeResponseCreated:
		event = &ResponseCreatedEvent{}
	case TypeResponseDone:
		event = &ResponseDoneEvent{}
	case TypeResponseAudioDelta:
		event = &ResponseAudioDeltaEvent{}
	case TypeResponseAudioDone:
		event = &ResponseAudioDoneEvent{}
	case TypeResponseTextDelta:
		event = &ResponseTextDeltaEvent{}
	case TypeResponseTextDone:
		event = &ResponseTextDoneEvent{}
	case TypeResponseAudioTranscriptDelta:
		event = &ResponseAudioTranscriptDeltaEvent{}
	case TypeResponseAudioTranscriptDone:
		event = &ResponseAudioTranscriptDoneEvent{}
	case TypeResponseFunctionCallArgumentsDelta:
		event = &ResponseFunctionCallArgumentsDeltaEvent{}
	case TypeResponseFunctionCallArgumentsDone:
		event = &ResponseFunctionCallArgumentsDoneEvent{}
	case TypeRateLimitsUpdated:
		event = &RateLimitsUpdatedEvent{}
	default:
		// Return a generic event for unknown types
		event = &baseEvent{}
	}

	if err := json.Unmarshal(data, event); err != nil {
		return nil, err
	}
	return event, nil
}
