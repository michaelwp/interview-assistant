// Package audio handles real-time audio capture from one or two PortAudio devices.
// Audio is emitted in variable-length chunks based on voice activity detection (VAD):
// a chunk is sent as soon as the speaker stops talking, rather than on a fixed timer.
package audio

import "time"

// Audio constants.
const (
	SampleRate      = 16000 // Hz — Whisper performs best at 16 kHz
	Channels        = 1     // Mono
	FramesPerBuffer = 512   // ~32 ms per PortAudio Read() at 16 kHz
)

// VAD thresholds and timing (in number of Read() frames, each ~32 ms).
const (
	SilenceThreshold = 0.008 // RMS below this → silence
	silenceGapFrames = 12    // 12 × 32 ms = ~384 ms silence → end of utterance
	minSpeechFrames  = 5     // 5 × 32 ms = ~160 ms minimum speech to emit
	preSpeechFrames  = 3     // frames of pre-speech context kept in buffer
	maxChunkFrames   = 312   // ~10 s safety valve to avoid runaway chunks
)

// Speaker identifies whose audio is in a chunk.
type Speaker int

const (
	SpeakerMic    Speaker = iota // microphone — the interviewee
	SpeakerSystem                // system audio (BlackHole) — the interviewer
)

func (s Speaker) Label() string {
	if s == SpeakerMic {
		return "You"
	}
	return "Interviewer"
}

// Chunk is one speech segment of PCM audio from a single speaker.
type Chunk struct {
	Samples   []int16
	Speaker   Speaker
	Timestamp time.Time
	HasSpeech bool
}

// CaptureConfig configures the audio capture backend.
type CaptureConfig struct {
	MicDevIdx           int
	SysDevIdx           int  // -1 = no system audio via PortAudio; ignored when UseScreenCaptureKit is true
	UseScreenCaptureKit bool // capture system audio via ScreenCaptureKit (macOS 13+, no driver needed)
}
