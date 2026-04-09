// Package ui implements the Bubbletea TUI.
package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"macInterviewCracking/assistant"
	"macInterviewCracking/audio"
	"macInterviewCracking/transcription"
)

// Model is the Bubbletea model.
type Model struct {
	cfg Config

	// background pipeline
	capturer    *audio.Capturer
	transcriber transcription.Transcriber
	gpt         assistant.Answerer
	msgCh       chan tea.Msg
	cancelCtx   context.CancelFunc
	ctx         context.Context

	// UI state
	transcript         []transcriptLine
	lastErr            string
	status             string
	apiCalls           int
	questions          int
	debounceSeq        int
	pendingQuestionIdx int // -1 = none pending

	viewport viewport.Model
	vpReady  bool
	width    int
	height   int
}

// New creates a new Model.
func New(cfg Config) *Model {
	return &Model{
		cfg:                cfg,
		transcriber:        cfg.Transcriber,
		gpt:                cfg.Answerer,
		msgCh:              make(chan tea.Msg, 32),
		status:             "Initializing...",
		pendingQuestionIdx: -1,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.startPipeline(),
		m.waitForMsg(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rebuildViewport()

	case readyMsg:
		m.status = m.buildStatusLine()
		cmds = append(cmds, m.waitForMsg())

	case transcriptMsg:
		m.transcript = append(m.transcript, transcriptLine{
			speaker:   msg.speaker,
			text:      msg.text,
			timestamp: msg.timestamp,
		})
		lineIdx := len(m.transcript) - 1
		m.apiCalls++
		m.status = m.buildStatusLine()
		m.refreshViewport()
		cmds = append(cmds, m.waitForMsg())

		shouldAnswer := m.cfg.Mode == "single" || msg.speaker == audio.SpeakerSystem
		if shouldAnswer {
			if assistant.IsQuestion(msg.text) {
				m.pendingQuestionIdx = lineIdx
			}
			if m.pendingQuestionIdx >= 0 {
				m.debounceSeq++
				cmds = append(cmds, m.scheduleAnswer(m.pendingQuestionIdx, m.debounceSeq))
			}
		}

	case answerReadyMsg:
		if msg.seq == m.debounceSeq && msg.lineIdx >= 0 && msg.lineIdx < len(m.transcript) {
			m.pendingQuestionIdx = -1
			cmds = append(cmds, m.fetchSuggestion(m.transcript[msg.lineIdx].text, msg.lineIdx))
		}

	case suggestionMsg:
		if msg.suggestion != nil && msg.lineIdx < len(m.transcript) {
			m.transcript[msg.lineIdx].suggestion = msg.suggestion
			m.questions++
			m.status = m.buildStatusLine()
			m.refreshViewport()
		}
		cmds = append(cmds, m.waitForMsg())

	case errMsg:
		m.lastErr = msg.err.Error()
		cmds = append(cmds, m.waitForMsg())

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.cancelCtx != nil {
				m.cancelCtx()
			}
			if m.capturer != nil {
				m.capturer.Stop()
			}
			return m, tea.Quit
		case "c":
			m.transcript = nil
			m.lastErr = ""
			m.pendingQuestionIdx = -1
			m.debounceSeq++
			m.refreshViewport()
		}
	}

	if m.vpReady {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
