// Package realtime provides a client for the OpenAI Realtime API.
//
// The Realtime API enables low-latency, multimodal conversational experiences
// with GPT-4o-realtime models. It supports native speech-to-speech processing
// with latency as low as ~100ms.
//
// # Features
//
//   - Native speech-to-speech (no intermediate TTS/STT)
//   - WebSocket-based bidirectional streaming
//   - Voice activity detection (VAD)
//   - Function calling support
//   - Multiple voice options
//
// # Audio Format
//
// The API uses PCM16 audio at 24kHz mono (native format for LiveKit integration).
// Audio is transmitted as base64-encoded chunks.
//
// # Usage
//
//	client := realtime.NewClient(apiKey,
//	    realtime.WithModel("gpt-4o-realtime-preview-2024-12-17"),
//	    realtime.WithVoice("alloy"),
//	)
//
//	session, err := client.Connect(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer session.Close()
//
//	// Send audio
//	session.SendAudio(audioData)
//
//	// Receive events
//	for event := range session.Events() {
//	    switch e := event.(type) {
//	    case *realtime.ResponseAudioDelta:
//	        // Handle audio output
//	    case *realtime.ResponseTextDelta:
//	        // Handle text transcript
//	    }
//	}
//
// # References
//
//   - OpenAI Realtime API: https://platform.openai.com/docs/guides/realtime
//   - API Reference: https://platform.openai.com/docs/api-reference/realtime
package realtime
