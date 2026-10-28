package omnivoice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/plexusone/omnivoice-core/audio/converter"
	"github.com/plexusone/omnivoice-core/audio/format"
)

// PCMSampleRate is the sample rate of OpenAI's raw "pcm" speech: 16-bit
// little-endian mono at 24 kHz.
const PCMSampleRate = 24000

// telephonySampleRate is the default rate for mu-law and A-law, the formats
// phone calls use.
const telephonySampleRate = 8000

// outputPlan is how a requested output format is produced: the format asked
// of OpenAI, and the conversion applied to its audio, if any.
type outputPlan struct {
	// apiFormat is the response_format sent to OpenAI; empty uses the
	// client's default.
	apiFormat string

	// convert is the format OpenAI's 24 kHz PCM is converted to, or nil
	// when the audio is returned as OpenAI sends it.
	convert *format.AudioFormat

	// resultFormat and sampleRate describe the returned audio.
	resultFormat string
	sampleRate   int
}

// planOutput maps a requested output format to OpenAI's formats. OpenAI
// produces raw audio only as 24 kHz PCM, so raw requests are served from
// that: PCM at another rate is resampled, and mu-law and A-law, which
// OpenAI does not offer, are encoded from it, at 8 kHz unless another rate
// is requested. Encoded formats such as mp3 or wav pass through.
//
// Both "mulaw" with a SampleRate and names with the rate appended, such as
// "ulaw_8000" or "pcm_16000", are accepted, so a config written for another
// provider works unchanged.
func planOutput(outputFormat string, sampleRate int) outputPlan {
	name := strings.ToLower(strings.TrimSpace(outputFormat))
	if base, rate, ok := splitRate(name); ok {
		name = base
		if sampleRate == 0 {
			sampleRate = rate
		}
	}
	// "wav" normalizes to linear16, but OpenAI's wav has a header.
	if name == "" || name == "wav" || !format.IsRawEncoding(name) {
		return outputPlan{apiFormat: outputFormat, resultFormat: outputFormat}
	}

	enc := format.Encoding(name).Normalize()
	if enc == format.Linear16 {
		if sampleRate == 0 || sampleRate == PCMSampleRate {
			return outputPlan{apiFormat: "pcm", resultFormat: "pcm", sampleRate: PCMSampleRate}
		}
	} else if sampleRate == 0 {
		sampleRate = telephonySampleRate
	}
	to := format.AudioFormat{Encoding: string(enc), SampleRate: sampleRate, Channels: 1}
	result := string(enc)
	if enc == format.Linear16 {
		result = "pcm"
	}
	return outputPlan{apiFormat: "pcm", convert: &to, resultFormat: result, sampleRate: sampleRate}
}

// splitRate splits a trailing sample rate from a format name such as
// "ulaw_8000".
func splitRate(name string) (string, int, bool) {
	i := strings.LastIndex(name, "_")
	if i <= 0 {
		return name, 0, false
	}
	rate, err := strconv.Atoi(name[i+1:])
	if err != nil || rate <= 0 {
		return name, 0, false
	}
	return name[:i], rate, true
}

// openAIPCM is the format of OpenAI's raw speech.
var openAIPCM = format.AudioFormat{Encoding: string(format.Linear16), SampleRate: PCMSampleRate, Channels: 1}

// convertAudio converts OpenAI's PCM to the planned format.
func (p outputPlan) convertAudio(audio []byte) ([]byte, error) {
	if p.convert == nil {
		return audio, nil
	}
	out, err := converter.New().Convert(audio, openAIPCM, *p.convert)
	if err != nil {
		return nil, fmt.Errorf("convert speech to %s at %d Hz: %w", p.convert.Encoding, p.convert.SampleRate, err)
	}
	return out, nil
}

// streamConverter converts streamed PCM chunk by chunk. A chunk can end in
// the middle of a sample, or of the group of samples one output sample is
// made from, so the remainder is held for the next chunk.
type streamConverter struct {
	plan    outputPlan
	pending []byte
}

// frameBytes is how many input bytes the converter must see together: one
// 16-bit sample, or a whole resampling group when the rate divides 24 kHz.
func (s *streamConverter) frameBytes() int {
	if s.plan.convert == nil || s.plan.convert.SampleRate == 0 {
		return 2
	}
	if ratio := PCMSampleRate / s.plan.convert.SampleRate; ratio > 1 && PCMSampleRate%s.plan.convert.SampleRate == 0 {
		return 2 * ratio
	}
	return 2
}

// convert converts as much of chunk, after anything held back, as fills
// whole frames.
func (s *streamConverter) convert(chunk []byte) ([]byte, error) {
	if s.plan.convert == nil {
		return chunk, nil
	}
	data := append(s.pending, chunk...)
	n := len(data) - len(data)%s.frameBytes()
	s.pending = append([]byte(nil), data[n:]...)
	return s.plan.convertAudio(data[:n])
}

// flush converts what is held back at the end of the stream, dropping a
// final odd byte, which cannot be a sample.
func (s *streamConverter) flush() ([]byte, error) {
	if s.plan.convert == nil || len(s.pending) < 2 {
		return nil, nil
	}
	data := s.pending[:len(s.pending)-len(s.pending)%2]
	s.pending = nil
	return s.plan.convertAudio(data)
}
