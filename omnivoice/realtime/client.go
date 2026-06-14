package realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

const (
	// RealtimeAPIEndpoint is the WebSocket endpoint for the Realtime API.
	RealtimeAPIEndpoint = "wss://api.openai.com/v1/realtime"
)

// Client is the OpenAI Realtime API client.
type Client struct {
	config Config
}

// NewClient creates a new Realtime API client.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		config: Config{
			APIKey: apiKey,
		},
	}

	for _, opt := range opts {
		opt(&c.config)
	}

	applyDefaults(&c.config)

	return c
}

// NewClientFromEnv creates a client using the OPENAI_API_KEY environment variable.
func NewClientFromEnv(opts ...Option) (*Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY environment variable not set")
	}
	return NewClient(apiKey, opts...), nil
}

// Connect establishes a WebSocket connection and returns a Session.
func (c *Client) Connect(ctx context.Context) (*Session, error) {
	url := fmt.Sprintf("%s?model=%s", RealtimeAPIEndpoint, c.config.Model)

	header := http.Header{}
	header.Set("Authorization", "Bearer "+c.config.APIKey)
	header.Set("OpenAI-Beta", "realtime=v1")

	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.ConnectTimeout,
	}

	conn, resp, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("websocket dial failed with status %d: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	session := &Session{
		conn:     conn,
		config:   c.config,
		eventsCh: make(chan ServerEvent, 100),
		sendCh:   make(chan any, 100),
		closeCh:  make(chan struct{}),
	}

	// Start read/write goroutines
	session.wg.Add(2)
	go session.readLoop()
	go session.writeLoop()

	// Wait for session.created event
	select {
	case event := <-session.eventsCh:
		if created, ok := event.(*SessionCreatedEvent); ok {
			session.sessionID = created.Session.ID
		} else if errEvent, ok := event.(*ErrorEvent); ok {
			session.Close()
			return nil, fmt.Errorf("session creation failed: %s", errEvent.Error.Message)
		}
	case <-ctx.Done():
		session.Close()
		return nil, ctx.Err()
	}

	// Update session with config if needed
	if c.config.Instructions != "" || c.config.TurnDetection != nil || len(c.config.Tools) > 0 {
		sessionConfig := SessionConfig{
			Modalities:        c.config.Modalities,
			Instructions:      c.config.Instructions,
			Voice:             c.config.Voice,
			InputAudioFormat:  c.config.InputAudioFormat,
			OutputAudioFormat: c.config.OutputAudioFormat,
			TurnDetection:     c.config.TurnDetection,
			Tools:             c.config.Tools,
			Temperature:       c.config.Temperature,
		}
		if c.config.MaxResponseOutputTokens != nil {
			sessionConfig.MaxResponseOutputTokens = c.config.MaxResponseOutputTokens
		}
		if err := session.UpdateSession(sessionConfig); err != nil {
			session.Close()
			return nil, fmt.Errorf("failed to update session config: %w", err)
		}
	}

	return session, nil
}

// Session represents an active Realtime API session.
type Session struct {
	conn      *websocket.Conn
	config    Config
	sessionID string

	eventsCh chan ServerEvent
	sendCh   chan any
	closeCh  chan struct{}
	closed   bool
	closeMu  sync.Mutex
	wg       sync.WaitGroup
}

// ID returns the session ID.
func (s *Session) ID() string {
	return s.sessionID
}

// Events returns a channel of server events.
func (s *Session) Events() <-chan ServerEvent {
	return s.eventsCh
}

// SendAudio sends audio data to the input buffer.
// Audio should be PCM16 24kHz mono (or the configured input format).
func (s *Session) SendAudio(audio []byte) error {
	encoded := base64.StdEncoding.EncodeToString(audio)
	return s.send(InputAudioBufferAppendEvent{
		Type:  TypeInputAudioBufferAppend,
		Audio: encoded,
	})
}

// CommitAudio commits the audio buffer.
// This is typically called when turn detection is disabled.
func (s *Session) CommitAudio() error {
	return s.send(InputAudioBufferCommitEvent{
		Type: TypeInputAudioBufferCommit,
	})
}

// ClearAudio clears the audio buffer.
func (s *Session) ClearAudio() error {
	return s.send(InputAudioBufferClearEvent{
		Type: TypeInputAudioBufferClear,
	})
}

// CreateResponse requests a response from the model.
func (s *Session) CreateResponse(config *ResponseConfig) error {
	return s.send(ResponseCreateEvent{
		Type:     TypeResponseCreate,
		Response: config,
	})
}

// CancelResponse cancels an in-progress response.
func (s *Session) CancelResponse() error {
	return s.send(ResponseCancelEvent{
		Type: TypeResponseCancel,
	})
}

// UpdateSession updates the session configuration.
func (s *Session) UpdateSession(config SessionConfig) error {
	return s.send(SessionUpdateEvent{
		Type:    TypeSessionUpdate,
		Session: config,
	})
}

// SendText sends a text message (bypasses audio input).
func (s *Session) SendText(text string) error {
	return s.send(ConversationItemCreateEvent{
		Type: TypeConversationItemCreate,
		Item: ConversationItem{
			Type: "message",
			Role: "user",
			Content: []ContentPart{
				{Type: "input_text", Text: text},
			},
		},
	})
}

// SendFunctionOutput sends the result of a function call.
func (s *Session) SendFunctionOutput(callID, output string) error {
	return s.send(FunctionCallOutputEvent{
		Type: TypeConversationItemCreate,
		Item: struct {
			Type   string `json:"type"`
			CallID string `json:"call_id"`
			Output string `json:"output"`
		}{
			Type:   "function_call_output",
			CallID: callID,
			Output: output,
		},
	})
}

// Close closes the session and underlying connection.
func (s *Session) Close() error {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return nil
	}
	s.closed = true
	close(s.closeCh)
	s.closeMu.Unlock()

	err := s.conn.Close()
	s.wg.Wait()
	close(s.eventsCh)

	return err
}

// send sends an event to the server.
func (s *Session) send(event any) error {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return errors.New("session closed")
	}
	s.closeMu.Unlock()

	select {
	case s.sendCh <- event:
		return nil
	default:
		return errors.New("send channel full")
	}
}

// readLoop reads messages from the WebSocket.
func (s *Session) readLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.closeCh:
			return
		default:
		}

		_, data, err := s.conn.ReadMessage()
		if err != nil {
			s.closeMu.Lock()
			closed := s.closed
			s.closeMu.Unlock()
			if !closed {
				// Send error event
				errEvent := &ErrorEvent{}
				errEvent.Type = TypeError
				errEvent.Error.Message = err.Error()
				select {
				case s.eventsCh <- errEvent:
				default:
				}
			}
			return
		}

		event, err := parseServerEvent(data)
		if err != nil {
			continue // Skip unparseable events
		}

		select {
		case s.eventsCh <- event:
		case <-s.closeCh:
			return
		default:
			// Drop event if channel is full
		}
	}
}

// writeLoop writes messages to the WebSocket.
func (s *Session) writeLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.closeCh:
			return
		case event := <-s.sendCh:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}
