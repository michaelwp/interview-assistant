package transcription

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"macInterviewCracking/audio"
)

// LocalTranscriber runs whisper-cpp as a subprocess for fully offline transcription.
// Requires: brew install whisper-cpp  and a downloaded model file.
type LocalTranscriber struct {
	modelPath string
	language  string
	binary    string // path or name of the whisper-cpp binary
}

// NewLocal creates a LocalTranscriber.
// binaryPath may be empty to use "whisper-cli" from PATH.
func NewLocal(modelPath, language, binaryPath string) *LocalTranscriber {
	if binaryPath == "" {
		binaryPath = "whisper-cli"
	}
	return &LocalTranscriber{
		modelPath: modelPath,
		language:  language,
		binary:    binaryPath,
	}
}

// Transcribe writes the chunk to a temp WAV file, runs whisper-cpp, and returns the text.
// Returns nil, nil when the chunk is silence.
func (t *LocalTranscriber) Transcribe(ctx context.Context, chunk audio.Chunk) (*Segment, error) {
	if !chunk.HasSpeech {
		return nil, nil
	}

	// Write WAV to a temp file.
	tmp, err := os.CreateTemp("", "whisper-*.wav")
	if err != nil {
		return nil, fmt.Errorf("local transcribe: create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	wavData := audio.EncodeWAV(chunk.Samples, audio.SampleRate, audio.Channels)
	if _, err := tmp.Write(wavData); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("local transcribe: write wav: %w", err)
	}
	tmp.Close()

	// Run whisper-cpp.
	//   -m  model file
	//   -l  language
	//   -nt no timestamps in output
	//   -np no progress bar
	cmd := exec.CommandContext(ctx, t.binary,
		"-m", t.modelPath,
		"-l", t.language,
		"-nt",
		"-np",
		tmp.Name(),
	)

	out, err := cmd.Output()
	if err != nil {
		// Include stderr for diagnostics when available.
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("whisper-cpp: %w\n%s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("whisper-cpp: %w", err)
	}

	text := parseWhisperOutput(string(out))
	if text == "" {
		return nil, nil
	}

	return &Segment{
		Speaker:   chunk.Speaker,
		Text:      text,
		Timestamp: chunk.Timestamp,
	}, nil
}

// parseWhisperOutput extracts clean transcribed text from whisper-cpp stdout.
// It skips initialization log lines and strips leading/trailing whitespace.
func parseWhisperOutput(raw string) string {
	var parts []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip whisper-cpp log/info lines.
		if strings.HasPrefix(line, "whisper_") ||
			strings.HasPrefix(line, "main:") ||
			strings.HasPrefix(line, "system_info:") ||
			strings.HasPrefix(line, "ggml_") {
			continue
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, " ")
}
