package theme

import "github.com/charmbracelet/lipgloss"

var (
	Header   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	Footer   = lipgloss.NewStyle().Faint(true)
	Cursor   = lipgloss.NewStyle().Reverse(true)
	Seen     = lipgloss.NewStyle().Faint(true)
	SevInfo  = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	SevWatch = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	SevAct   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

func SeverityStyle(sev string) lipgloss.Style {
	switch sev {
	case "act":
		return SevAct
	case "watch":
		return SevWatch
	default:
		return SevInfo
	}
}
