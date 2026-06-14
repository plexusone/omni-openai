package realtime

import "errors"

// Common errors.
var (
	// ErrNoAPIKey is returned when API key is not provided.
	ErrNoAPIKey = errors.New("API key is required")

	// ErrSessionClosed is returned when operating on a closed session.
	ErrSessionClosed = errors.New("session closed")

	// ErrConnectionFailed is returned when WebSocket connection fails.
	ErrConnectionFailed = errors.New("websocket connection failed")

	// ErrSendFailed is returned when sending a message fails.
	ErrSendFailed = errors.New("failed to send message")

	// ErrInvalidAudioFormat is returned for unsupported audio formats.
	ErrInvalidAudioFormat = errors.New("invalid audio format")
)
