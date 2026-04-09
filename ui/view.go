package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"macInterviewCracking/audio"
)

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	w := m.width

	// Header
	var modeLabel string
	if m.cfg.Mode == "dual" {
		modeLabel = "Dual"
	} else {
		modeLabel = "Single"
	}
	profileLabel := ""
	if m.cfg.ProfileName != "" {
		profileLabel = "  |  profile: " + m.cfg.ProfileName
	}
	if m.cfg.CompanyName != "" {
		profileLabel += "  |  company: " + m.cfg.CompanyName
	}
	header := headerStyle.Width(w).Render(
		fmt.Sprintf(" Interview Assistant   [%s mode%s]", modeLabel, profileLabel),
	)

	divider := dividerStyle.Render(strings.Repeat("─", w))

	transcriptPane := ""
	if m.vpReady {
		transcriptPane = m.viewport.View()
	}

	errLine := ""
	if m.lastErr != "" {
		errLine = errorStyle.Render("  Error: "+m.lastErr) + "\n"
	}

	help := helpStyle.Render("[q] Quit  [c] Clear  [↑↓] Scroll")
	status := statusStyle.Render(m.status)
	gap := max(0, w-lipgloss.Width(help)-lipgloss.Width(status))
	statusBar := help + strings.Repeat(" ", gap) + status

	return strings.Join([]string{
		header,
		divider,
		transcriptPane,
		divider,
		errLine,
		statusBar,
	}, "\n")
}

func (m *Model) rebuildViewport() {
	// header(1) + 2×divider(2) + error(1) + statusBar(1) = 5 lines overhead
	overhead := 6
	vpHeight := m.height - overhead
	if vpHeight < 3 {
		vpHeight = 3
	}
	m.viewport = viewport.New(m.width, vpHeight)
	m.viewport.SetContent(m.renderTranscript())
	m.viewport.GotoBottom()
	m.vpReady = true
}

func (m *Model) refreshViewport() {
	if !m.vpReady {
		return
	}
	atBottom := m.viewport.AtBottom()
	m.viewport.SetContent(m.renderTranscript())
	if atBottom {
		m.viewport.GotoBottom()
	}
}

func (m *Model) renderTranscript() string {
	if len(m.transcript) == 0 {
		return statusStyle.Render("  Listening... (speak or let the interviewer speak)")
	}

	var sb strings.Builder
	for _, line := range m.transcript {
		ts := timestampStyle.Render(line.timestamp.Format("15:04:05"))
		var speaker string
		if line.speaker == audio.SpeakerMic {
			speaker = youStyle.Render(fmt.Sprintf("%-12s", "You"))
		} else {
			speaker = interviewerStyle.Render(fmt.Sprintf("%-12s", "Interviewer"))
		}

		wrapped := wordWrap(line.text, m.width-28)
		for i, l := range strings.Split(wrapped, "\n") {
			if i == 0 {
				sb.WriteString(fmt.Sprintf("  %s │ %s │ %s\n", ts, speaker, l))
			} else {
				sb.WriteString(fmt.Sprintf("  %s │ %s │ %s\n",
					timestampStyle.Render("        "),
					strings.Repeat(" ", 12),
					l))
			}
		}

		if line.suggestion != nil {
			ansWidth := m.width - 14
			sb.WriteString(fmt.Sprintf("  %s\n", suggestionLabelStyle.Render("           ╰─ 💡 Suggested Answer:")))
			for _, al := range strings.Split(wordWrap(line.suggestion.Answer, ansWidth), "\n") {
				sb.WriteString(fmt.Sprintf("  %s %s\n",
					strings.Repeat(" ", 12),
					answerStyle.Render(al)))
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m *Model) buildStatusLine() string {
	return fmt.Sprintf("Transcriptions: %d | Questions answered: %d | model: %s",
		m.apiCalls, m.questions, m.cfg.AnswerModelName)
}
