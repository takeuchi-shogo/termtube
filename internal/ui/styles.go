package ui

import "github.com/charmbracelet/lipgloss"

var (
	ActiveTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Padding(0, 1)
	InactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1)
	TitleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	SubtitleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	HelpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
	BorderStyle      = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1)
)
