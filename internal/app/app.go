package app

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takeuchishougo/termtube/internal/player"
	"github.com/takeuchishougo/termtube/internal/storage"
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
	currentView   View
	width         int
	height        int
	search        ui.SearchModel
	player        ui.PlayerModel
	history       ui.HistoryModel
	playlist      ui.PlaylistModel
	mpv           *player.MpvPlayer
	historyStore  *storage.HistoryStore
	playlistStore *storage.PlaylistStore
	statusMsg     string // temporary status message
}

func New() Model {
	mpv := player.NewMpvPlayer()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".local", "share", "termtube")

	historyStore := storage.NewHistoryStore(filepath.Join(dataDir, "history.json"))
	playlistStore := storage.NewPlaylistStore(filepath.Join(dataDir, "playlists"))

	return Model{
		currentView:   ViewSearch,
		search:        ui.NewSearchModel(),
		player:        ui.NewPlayerModel(mpv),
		history:       ui.NewHistoryModel(historyStore),
		playlist:      ui.NewPlaylistModel(playlistStore),
		mpv:           mpv,
		historyStore:  historyStore,
		playlistStore: playlistStore,
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
		// Pass window size to all sub-models (reserve space for tab bar)
		contentMsg := tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height - 1, // 1 line for tab bar
		}
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(contentMsg)
		cmds = append(cmds, cmd)
		m.player, cmd = m.player.Update(contentMsg)
		cmds = append(cmds, cmd)
		m.history, cmd = m.history.Update(contentMsg)
		cmds = append(cmds, cmd)
		m.playlist, cmd = m.playlist.Update(contentMsg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case ui.SearchResultMsg:
		if m.currentView == ViewSearch {
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd
		}

	case ui.PlayVideoMsg:
		// Received playback request: start playing, switch to player view, and add to history
		m.currentView = ViewPlayer
		m.statusMsg = ""
		var cmds []tea.Cmd
		cmds = append(cmds, m.player.PlayVideo(msg.Video))
		cmds = append(cmds, ui.AddToHistory(m.historyStore, msg.Video))
		// Fetch related videos async based on the video title
		cmds = append(cmds, ui.FetchRelatedVideos(msg.Video.Title))
		return m, tea.Batch(cmds...)

	case ui.PlayerStateMsg:
		// Forward player state updates to player model
		var cmd tea.Cmd
		m.player, cmd = m.player.Update(msg)
		return m, cmd

	case ui.HistoryLoadedMsg:
		var cmd tea.Cmd
		m.history, cmd = m.history.Update(msg)
		return m, cmd

	case ui.PlaylistsLoadedMsg:
		var cmd tea.Cmd
		m.playlist, cmd = m.playlist.Update(msg)
		return m, cmd

	case ui.PlaylistVideosLoadedMsg:
		var cmd tea.Cmd
		m.playlist, cmd = m.playlist.Update(msg)
		return m, cmd

	case ui.RelatedVideosMsg:
		// Forward related videos to player model
		var cmd tea.Cmd
		m.player, cmd = m.player.Update(msg)
		return m, cmd

	case ui.ChatMessageMsg:
		// Forward chat messages to player model
		var cmd tea.Cmd
		m.player, cmd = m.player.Update(msg)
		return m, cmd

	case ui.VideoAddedToPlaylistMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("エラー: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("「%s」を「%s」に追加しました", msg.VideoTitle, msg.PlaylistName)
		}
		return m, nil

	case tea.KeyMsg:
		// When search is in input mode, only handle ctrl+c for quitting
		if m.currentView == ViewSearch && m.search.IsInputMode() {
			if msg.String() == "ctrl+c" {
				m.mpv.Stop()
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
			m.mpv.Stop()
			return m, tea.Quit
		case "1":
			m.currentView = ViewSearch
			m.statusMsg = ""
			return m, nil
		case "2":
			m.currentView = ViewPlayer
			m.statusMsg = ""
			return m, nil
		case "3":
			m.currentView = ViewPlaylist
			m.statusMsg = ""
			return m, m.playlist.Refresh()
		case "4":
			m.currentView = ViewHistory
			m.statusMsg = ""
			return m, m.history.Refresh()
		case "enter":
			// In search list mode, Enter triggers video playback
			if m.currentView == ViewSearch && !m.search.IsInputMode() {
				video := m.search.SelectedVideo()
				if video != nil {
					return m, func() tea.Msg {
						return ui.PlayVideoMsg{Video: *video}
					}
				}
			}
		case "a":
			// In search list mode, add selected video to "お気に入り" playlist
			if m.currentView == ViewSearch && !m.search.IsInputMode() {
				video := m.search.SelectedVideo()
				if video != nil {
					return m, ui.AddVideoToPlaylist(m.playlistStore, "お気に入り", *video)
				}
			}
		}

		// Delegate to current view's model
		switch m.currentView {
		case ViewSearch:
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd
		case ViewPlayer:
			var cmd tea.Cmd
			m.player, cmd = m.player.Update(msg)
			return m, cmd
		case ViewHistory:
			var cmd tea.Cmd
			m.history, cmd = m.history.Update(msg)
			return m, cmd
		case ViewPlaylist:
			var cmd tea.Cmd
			m.playlist, cmd = m.playlist.Update(msg)
			return m, cmd
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
		content = m.player.View()
	case ViewPlaylist:
		content = m.playlist.View()
	case ViewHistory:
		content = m.history.View()
	}

	parts := []string{tabs}
	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Padding(0, 1)
		parts = append(parts, statusStyle.Render(m.statusMsg))
	}
	parts = append(parts, content)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
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
