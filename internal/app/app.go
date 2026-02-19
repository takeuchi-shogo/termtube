package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type View int

const (
	ViewSearch View = iota
	ViewPlayer
	ViewPlaylist
	ViewHistory
)

type Model struct {
	currentView View
	width       int
	height      int
}

func New() Model {
	return Model{
		currentView: ViewSearch,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.currentView = ViewSearch
		case "2":
			m.currentView = ViewPlayer
		case "3":
			m.currentView = ViewPlaylist
		case "4":
			m.currentView = ViewHistory
		}
	}
	return m, nil
}

func (m Model) View() string {
	tabs := m.renderTabs()
	content := "termtube へようこそ！\n\n数字キーでタブ切替 / q で終了"
	return lipgloss.JoinVertical(lipgloss.Left, tabs, content)
}

func (m Model) renderTabs() string {
	tabs := []string{"[1]検索", "[2]再生", "[3]PL", "[4]履歴"}
	active := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	inactive := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var rendered []string
	for i, t := range tabs {
		if View(i) == m.currentView {
			rendered = append(rendered, active.Render(t))
		} else {
			rendered = append(rendered, inactive.Render(t))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}
