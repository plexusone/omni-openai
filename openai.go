// Package openai provides a Go client for OpenAI's audio APIs (Whisper STT and TTS).
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
)

// Client wraps the OpenAI API client for audio operations.
type Client struct {
	client openai.Client
}

// NewClient creates a new OpenAI client with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
	}
}

// NewClientFromEnv creates a new OpenAI client using the OPENAI_API_KEY environment variable.
func NewClientFromEnv() (*Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}
	return NewClient(apiKey), nil
}

// TranscriptionRequest configures a Whisper transcription request.
type TranscriptionRequest struct {
	// Audio is the audio data to transcribe.
	Audio []byte

	// Filename is the name of the audio file (used for format detection).
	Filename string

	// Model is the Whisper model to use (default: "whisper-1").
	Model string

	// Language is the language of the audio (ISO-639-1 code, e.g., "en").
	// Leave empty for automatic detection.
	Language string

	// Prompt is optional text to guide the model's style or continue a previous segment.
	Prompt string

	// ResponseFormat is the output format: "json", "text", "srt", "verbose_json", "vtt".
	ResponseFormat string

	// Temperature is the sampling temperature (0-1). Lower is more deterministic.
	Temperature float64

	// TimestampGranularities specifies timestamp detail: "word", "segment", or both.
	TimestampGranularities []string
}

// TranscriptionResponse contains the Whisper transcription result.
type TranscriptionResponse struct {
	// Text is the transcribed text.
	Text string

	// Language is the detected language.
	Language string

	// Duration is the audio duration in seconds.
	Duration float64

	// Words contains word-level timestamps (if requested).
	Words []WordTimestamp

	// Segments contains segment-level timestamps (if requested).
	Segments []Segment
}

// WordTimestamp represents a word with timing information.
type WordTimestamp struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Segment represents a transcription segment.
type Segment struct {
	ID               int64   `json:"id"`
	Seek             int64   `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

// verboseTranscriptionResponse is used to parse the raw JSON response for verbose format.
type verboseTranscriptionResponse struct {
	Text     string          `json:"text"`
	Language string          `json:"language"`
	Duration float64         `json:"duration"`
	Words    []WordTimestamp `json:"words"`
	Segments []Segment       `json:"segments"`
}

// Transcribe converts audio to text using Whisper.
func (c *Client) Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResponse, error) {
	if req.Model == "" {
		req.Model = "whisper-1"
	}
	if req.Filename == "" {
		req.Filename = "audio.mp3"
	}

	// Build the request parameters
	params := openai.AudioTranscriptionNewParams{
		File:  bytes.NewReader(req.Audio),
		Model: req.Model,
	}

	if req.Language != "" {
		params.Language = param.NewOpt(req.Language)
	}
	if req.Prompt != "" {
		params.Prompt = param.NewOpt(req.Prompt)
	}
	if req.Temperature > 0 {
		params.Temperature = param.NewOpt(req.Temperature)
	}

	// Use verbose_json to get timestamps
	wantVerbose := len(req.TimestampGranularities) > 0 || req.ResponseFormat == "verbose_json"
	if wantVerbose {
		params.ResponseFormat = openai.AudioResponseFormatVerboseJSON
		if len(req.TimestampGranularities) > 0 {
			params.TimestampGranularities = req.TimestampGranularities
		}
	}

	// Call the API
	transcription, err := c.client.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	// If we requested verbose format, parse the raw JSON for extra fields
	if wantVerbose {
		var verbose verboseTranscriptionResponse
		if err := json.Unmarshal([]byte(transcription.RawJSON()), &verbose); err != nil {
			return nil, fmt.Errorf("failed to parse verbose response: %w", err)
		}

		return &TranscriptionResponse{
			Text:     verbose.Text,
			Language: verbose.Language,
			Duration: verbose.Duration,
			Words:    verbose.Words,
			Segments: verbose.Segments,
		}, nil
	}

	// Basic response
	return &TranscriptionResponse{
		Text: transcription.Text,
	}, nil
}

// TranscribeFile transcribes audio from a file path.
func (c *Client) TranscribeFile(ctx context.Context, filePath string, req TranscriptionRequest) (*TranscriptionResponse, error) {
	audio, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	req.Audio = audio
	if req.Filename == "" {
		req.Filename = filePath
	}
	return c.Transcribe(ctx, req)
}

// TTSRequest configures a text-to-speech request.
type TTSRequest struct {
	// Input is the text to convert to speech (max 4096 characters).
	Input string

	// Model is the TTS model: "tts-1" (faster) or "tts-1-hd" (higher quality).
	Model string

	// Voice is the voice to use: alloy, ash, ballad, coral, echo, fable, onyx, nova, sage, shimmer, verse, marin, cedar.
	Voice string

	// ResponseFormat is the audio format: "mp3", "opus", "aac", "flac", "wav", "pcm".
	ResponseFormat string

	// Speed is the speech speed (0.25 to 4.0, default 1.0).
	Speed float64
}

// TTSResponse contains the generated audio.
type TTSResponse struct {
	// Audio is the generated audio data.
	Audio []byte

	// Format is the audio format.
	Format string
}

// Synthesize converts text to speech.
func (c *Client) Synthesize(ctx context.Context, req TTSRequest) (*TTSResponse, error) {
	if req.Model == "" {
		req.Model = "tts-1"
	}
	if req.Voice == "" {
		req.Voice = "alloy"
	}
	if req.ResponseFormat == "" {
		req.ResponseFormat = "mp3"
	}

	params := openai.AudioSpeechNewParams{
		Input: req.Input,
		Model: req.Model,
		Voice: openai.AudioSpeechNewParamsVoice(req.Voice),
	}

	if req.ResponseFormat != "" {
		params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat(req.ResponseFormat)
	}

	if req.Speed > 0 {
		params.Speed = param.NewOpt(req.Speed)
	}

	// Call the API - returns an http.Response
	response, err := c.client.Audio.Speech.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("speech synthesis failed: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	// Read all audio data
	audio, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio: %w", err)
	}

	return &TTSResponse{
		Audio:  audio,
		Format: req.ResponseFormat,
	}, nil
}

// SynthesizeStream converts text to speech with streaming output.
func (c *Client) SynthesizeStream(ctx context.Context, req TTSRequest) (io.ReadCloser, error) {
	if req.Model == "" {
		req.Model = "tts-1"
	}
	if req.Voice == "" {
		req.Voice = "alloy"
	}
	if req.ResponseFormat == "" {
		req.ResponseFormat = "mp3"
	}

	params := openai.AudioSpeechNewParams{
		Input: req.Input,
		Model: req.Model,
		Voice: openai.AudioSpeechNewParamsVoice(req.Voice),
	}

	if req.ResponseFormat != "" {
		params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat(req.ResponseFormat)
	}

	if req.Speed > 0 {
		params.Speed = param.NewOpt(req.Speed)
	}

	response, err := c.client.Audio.Speech.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("speech synthesis failed: %w", err)
	}

	return response.Body, nil
}

// Voice constants for TTS.
const (
	VoiceAlloy   = "alloy"
	VoiceAsh     = "ash"
	VoiceBallad  = "ballad"
	VoiceCoral   = "coral"
	VoiceEcho    = "echo"
	VoiceFable   = "fable"
	VoiceOnyx    = "onyx"
	VoiceNova    = "nova"
	VoiceSage    = "sage"
	VoiceShimmer = "shimmer"
	VoiceVerse   = "verse"
	VoiceMarin   = "marin"
	VoiceCedar   = "cedar"
)

// Model constants.
const (
	ModelWhisper1 = "whisper-1"
	ModelTTS1     = "tts-1"
	ModelTTS1HD   = "tts-1-hd"
)
