package ui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"macInterviewCracking/audio"
)

// startPipeline initialises PortAudio capture and starts the background
// transcription worker. Returns readyMsg on success, errMsg on failure.
func (m *Model) startPipeline() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelCtx = cancel
		m.ctx = ctx

		cap, err := audio.New(audio.CaptureConfig{
				MicDevIdx:           m.cfg.MicDeviceIdx,
				SysDevIdx:           m.cfg.SysDeviceIdx,
				UseScreenCaptureKit: m.cfg.UseScreenCaptureKit,
			})
		if err != nil {
			return errMsg{fmt.Errorf("audio: %w", err)}
		}
		m.capturer = cap

		if err := cap.Start(); err != nil {
			return errMsg{fmt.Errorf("start capture: %w", err)}
		}

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case chunk, ok := <-cap.Chunks():
					if !ok {
						return
					}
					seg, err := m.transcriber.Transcribe(ctx, chunk)
					if err != nil {
						m.msgCh <- errMsg{err}
						continue
					}
					if seg == nil {
						continue
					}
					m.msgCh <- transcriptMsg{
						speaker:   seg.Speaker,
						text:      seg.Text,
						timestamp: seg.Timestamp,
					}
				}
			}
		}()

		return readyMsg{}
	}
}

// waitForMsg blocks until the background pipeline sends a message.
func (m *Model) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		return <-m.msgCh
	}
}

// scheduleAnswer waits for a pause in speech before triggering GPT.
// Any new speech increments debounceSeq, invalidating stale timers.
func (m *Model) scheduleAnswer(lineIdx, seq int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		return answerReadyMsg{lineIdx: lineIdx, seq: seq}
	}
}

// fetchSuggestion calls GPT and returns the answer tagged with the transcript line index.
func (m *Model) fetchSuggestion(text string, lineIdx int) tea.Cmd {
	return func() tea.Msg {
		suggestion, err := m.gpt.Answer(m.ctx, text)
		if err != nil {
			return errMsg{err}
		}
		return suggestionMsg{suggestion: suggestion, lineIdx: lineIdx}
	}
}
