# Release Notes: v0.1.0

**Release Date:** 2026-02-28

## Summary

Initial release of omnivoice-openai, providing OpenAI audio integration for the OmniVoice voice pipeline framework.

## Features

### OpenAI Client

- `Transcribe` - Convert audio to text using Whisper
- `TranscribeFile` - Transcribe audio from file path
- `Synthesize` - Generate speech from text
- `SynthesizeStream` - Stream speech synthesis output

### OmniVoice Integration

- STT provider implementing `stt.Provider` interface
- TTS provider implementing `tts.Provider` interface
- Seamless integration with OmniVoice pipelines

### Supported Voices

alloy, ash, ballad, coral, echo, fable, nova, onyx, sage, shimmer, verse, marin, cedar

## Installation

```bash
go get github.com/plexusone/omnivoice-openai@v0.1.0
```
