package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takeuchishougo/termtube/internal/ui"
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
	search      ui.SearchModel
}

func New() Model {
	return Model{
		currentView: ViewSearch,
		search:      ui.NewSearchModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return m.search.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Pass window size to search model (reserve space for tab bar)
		searchMsg := tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height - 1, // 1 line for tab bar
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(searchMsg)
		return m, cmd

	case ui.SearchResultMsg:
		if m.currentView == ViewSearch {
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		// When search is in input mode, only handle ctrl+c for quitting
		if m.currentView == ViewSearch && m.search.IsInputMode() {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			// Forward all other keys to search model
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd
		}

		// Global key handling (when search is NOT in input mode)
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
		default:
			// Delegate to current view's model
			if m.currentView == ViewSearch {
				var cmd tea.Cmd
				m.search, cmd = m.search.Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	tabs := m.renderTabs()

	var content string
	switch m.currentView {
	case ViewSearch:
		content = m.search.View()
	case ViewPlayer:
		content = "再生画面（未実装）"
	case ViewPlaylist:
		content = "プレイリスト画面（未実装）"
	case ViewHistory:
		content = "履歴画面（未実装）"
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabs, content)
}

func (m Model) renderTabs() string {
	tabs := []string{"[1]検索", "[2]再生", "[3]PL", "[4]履歴"}

	var rendered []string
	for i, t := range tabs {
		if View(i) == m.currentView {
			rendered = append(rendered, ui.ActiveTabStyle.Render(t))
		} else {
			rendered = append(rendered, ui.InactiveTabStyle.Render(t))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}
