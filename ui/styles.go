package ui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5C6BC0")).
			Padding(0, 2)

	dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#37474F"))

	youStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#66BB6A"))

	interviewerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#42A5F5"))

	timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#546E7A"))

	answerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#80CBC4"))

	suggestionLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#80CBC4")).
				Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#78909C")).
			PaddingLeft(1)

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF5350"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#546E7A")).
			PaddingRight(2)
)
