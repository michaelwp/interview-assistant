package ui

import (
	"time"

	"macInterviewCracking/assistant"
	"macInterviewCracking/audio"
)

// ── tea.Msg types sent between the pipeline and the Bubbletea loop ───────────

type transcriptMsg struct {
	speaker   audio.Speaker
	text      string
	timestamp time.Time
}

type suggestionMsg struct {
	suggestion *assistant.Suggestion
	lineIdx    int
}

// answerReadyMsg fires after the debounce delay; seq guards against stale timers.
type answerReadyMsg struct {
	lineIdx int
	seq     int
}

type errMsg struct{ err error }

type readyMsg struct{}

// ── Internal data ─────────────────────────────────────────────────────────────

type transcriptLine struct {
	speaker    audio.Speaker
	text       string
	timestamp  time.Time
	suggestion *assistant.Suggestion
}
