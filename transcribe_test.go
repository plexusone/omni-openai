package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func TestTranscribeSendsFileName(t *testing.T) {
	var gotName, gotType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mr, err := r.MultipartReader()
		if err != nil {
			t.Errorf("multipart: %v", err)
			return
		}
		for {
			part, err := mr.NextPart()
			if err != nil {
				break // io.EOF ends the form
			}
			if part.FormName() == "file" {
				gotName, gotType = part.FileName(), part.Header.Get("Content-Type")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"text": "hello"}`)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer srv.Close()

	c := &Client{client: openai.NewClient(option.WithAPIKey("test"), option.WithBaseURL(srv.URL))}
	resp, err := c.Transcribe(context.Background(), TranscriptionRequest{Audio: []byte("RIFF....WAVE"), Filename: "audio.wav"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "hello" {
		t.Errorf("text %q", resp.Text)
	}
	if gotName != "audio.wav" || gotType != "audio/wav" {
		t.Errorf("uploaded file %q (%s), want audio.wav (audio/wav)", gotName, gotType)
	}
}

func TestAudioContentType(t *testing.T) {
	for name, want := range map[string]string{"a.wav": "audio/wav", "a.MP3": "audio/mpeg", "a.flac": "audio/flac", "a.webm": "audio/webm", "a": "audio/mpeg"} {
		if got := audioContentType(name); got != want {
			t.Errorf("audioContentType(%q) = %q, want %q", name, got, want)
		}
	}
}
