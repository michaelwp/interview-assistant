// Package transcription wraps speech-to-text backends (online and local).
package transcription

import (
	"context"
	"time"

	"macInterviewCracking/audio"
)

// Segment is one transcribed chunk with speaker metadata.
type Segment struct {
	Speaker   audio.Speaker
	Text      string
	Timestamp time.Time
}

// Transcriber is the common interface for all transcription backends.
type Transcriber interface {
	Transcribe(ctx context.Context, chunk audio.Chunk) (*Segment, error)
}
