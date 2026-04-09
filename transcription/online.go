package transcription

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"macInterviewCracking/audio"
)

// OnlineTranscriber sends WAV audio to the OpenAI Whisper API.
type OnlineTranscriber struct {
	client   *openai.Client
	language string
}

// NewOnline creates an OnlineTranscriber.
func NewOnline(apiKey, language string) *OnlineTranscriber {
	return &OnlineTranscriber{
		client:   openai.NewClient(apiKey),
		language: language,
	}
}

// Transcribe encodes the chunk as WAV and sends it to the Whisper API.
// Returns nil, nil when the chunk is silence.
func (t *OnlineTranscriber) Transcribe(ctx context.Context, chunk audio.Chunk) (*Segment, error) {
	if !chunk.HasSpeech {
		return nil, nil
	}

	wavData := audio.EncodeWAV(chunk.Samples, audio.SampleRate, audio.Channels)

	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		Reader:   bytes.NewReader(wavData),
		FilePath: "audio.wav",
		Language: t.language,
	}

	resp, err := t.client.CreateTranscription(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("whisper API: %w", err)
	}

	text := strings.TrimSpace(resp.Text)
	if text == "" {
		return nil, nil
	}

	return &Segment{
		Speaker:   chunk.Speaker,
		Text:      text,
		Timestamp: chunk.Timestamp,
	}, nil
}
