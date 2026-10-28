package omnivoice

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/plexusone/omnivoice-core/audio/format"
)

func TestPlanOutput(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		rate       int
		apiFormat  string
		convert    *format.AudioFormat
		result     string
		resultRate int
	}{
		{"default", "", 0, "", nil, "", 0},
		{"mp3 passes through", "mp3", 0, "mp3", nil, "mp3", 0},
		{"wav keeps its header", "wav", 0, "wav", nil, "wav", 0},
		{"pcm is 24 kHz", "pcm", 0, "pcm", nil, "pcm", 24000},
		{"pcm at 24 kHz", "linear16", 24000, "pcm", nil, "pcm", 24000},
		{"pcm resampled", "pcm", 16000, "pcm", &format.AudioFormat{Encoding: "linear16", SampleRate: 16000, Channels: 1}, "pcm", 16000},
		{"mu-law for calls", "mulaw", 0, "pcm", &format.AudioFormat{Encoding: "mulaw", SampleRate: 8000, Channels: 1}, "mulaw", 8000},
		{"ulaw alias", "ulaw", 8000, "pcm", &format.AudioFormat{Encoding: "mulaw", SampleRate: 8000, Channels: 1}, "mulaw", 8000},
		{"rate in the name", "ulaw_8000", 0, "pcm", &format.AudioFormat{Encoding: "mulaw", SampleRate: 8000, Channels: 1}, "mulaw", 8000},
		{"pcm rate in the name", "pcm_16000", 0, "pcm", &format.AudioFormat{Encoding: "linear16", SampleRate: 16000, Channels: 1}, "pcm", 16000},
		{"a-law", "alaw", 0, "pcm", &format.AudioFormat{Encoding: "alaw", SampleRate: 8000, Channels: 1}, "alaw", 8000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := planOutput(tt.format, tt.rate)
			if p.apiFormat != tt.apiFormat || p.resultFormat != tt.result || p.sampleRate != tt.resultRate {
				t.Errorf("plan = %+v", p)
			}
			if (p.convert == nil) != (tt.convert == nil) || (p.convert != nil && *p.convert != *tt.convert) {
				t.Errorf("convert = %+v, want %+v", p.convert, tt.convert)
			}
		})
	}
}

// tone is one second of 24 kHz PCM.
func tone() []byte {
	pcm := make([]byte, 2*PCMSampleRate)
	for i := range PCMSampleRate {
		binary.LittleEndian.PutUint16(pcm[2*i:], uint16((i%48)*600))
	}
	return pcm
}

func TestConvertAudioToMuLaw(t *testing.T) {
	out, err := planOutput("mulaw", 0).convertAudio(tone())
	if err != nil {
		t.Fatalf("convertAudio() error: %v", err)
	}
	if len(out) != 8000 {
		t.Errorf("one second of mu-law at 8 kHz = %d bytes, want 8000", len(out))
	}
	same, err := planOutput("pcm", 0).convertAudio(tone())
	if err != nil || !bytes.Equal(same, tone()) {
		t.Errorf("24 kHz pcm was changed (err %v)", err)
	}
}

func TestStreamConverterMatchesWholeConversion(t *testing.T) {
	plan := planOutput("mulaw", 0)
	whole, err := plan.convertAudio(tone())
	if err != nil {
		t.Fatal(err)
	}

	// Chunks of sizes that split samples and resampling groups.
	s := &streamConverter{plan: plan}
	var streamed []byte
	pcm := tone()
	sizes := []int{1, 5, 7, 4096, 333}
	for i := 0; len(pcm) > 0; i++ {
		n := min(sizes[i%len(sizes)], len(pcm))
		out, err := s.convert(pcm[:n])
		if err != nil {
			t.Fatal(err)
		}
		streamed = append(streamed, out...)
		pcm = pcm[n:]
	}
	rest, err := s.flush()
	if err != nil {
		t.Fatal(err)
	}
	streamed = append(streamed, rest...)
	if !bytes.Equal(streamed, whole) {
		t.Errorf("streamed %d bytes differ from whole conversion of %d bytes", len(streamed), len(whole))
	}
}

func TestStreamConverterPassesThroughAndDropsOddByte(t *testing.T) {
	s := &streamConverter{plan: planOutput("mp3", 0)}
	if out, err := s.convert([]byte{1, 2, 3}); err != nil || !bytes.Equal(out, []byte{1, 2, 3}) {
		t.Errorf("mp3 chunk changed: %v, %v", out, err)
	}

	s = &streamConverter{plan: planOutput("pcm", 16000)}
	if _, err := s.convert([]byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if rest, err := s.flush(); err != nil || len(rest)%2 != 0 {
		t.Errorf("flush() = %d bytes, %v; want whole samples", len(rest), err)
	}
}
