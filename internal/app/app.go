package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takeuchishougo/termtube/internal/player"
	"github.com/takeuchishougo/termtube/internal/storage"
	"github.com/takeuchishougo/termtube/internal/ui"
	"github.com/takeuchishougo/termtube/internal/youtube"
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
	config        storage.Config
	configPath    string
	statusMsg     string // temporary status message
	initialURL    string // CLI から直接再生する URL
}

// New は通常の TUI モードで Model を生成する。
func New() Model {
	return newModel("")
}

// NewWithURL は指定 URL を即時再生する TUI モードで Model を生成する。
func NewWithURL(url string) Model {
	return newModel(url)
}

func newModel(initialURL string) Model {
	mpv := player.NewMpvPlayer()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".local", "share", "termtube")
	configPath := filepath.Join(dataDir, "config.toml")

	cfg, cfgErr := storage.LoadConfig(configPath)
	if cfgErr != nil {
		// config load failure is non-fatal; use defaults but log to stderr
		fmt.Fprintf(os.Stderr, "warning: failed to load config: %v\n", cfgErr)
	}

	// 設定から映像出力方式を適用
	if cfg.Player.VideoOutput != "" {
		mpv.SetVideoOutput(cfg.Player.VideoOutput)
	}

	historyStore := storage.NewHistoryStore(filepath.Join(dataDir, "history.json"))
	playlistStore := storage.NewPlaylistStore(filepath.Join(dataDir, "playlists"))

	playerModel := ui.NewPlayerModel(mpv)
	// 設定からデフォルトの表示モードを適用
	playerModel.SetViewMode(ui.ViewModeFromString(cfg.Player.DefaultMode))

	m := Model{
		currentView:   ViewSearch,
		search:        ui.NewSearchModel(),
		player:        playerModel,
		history:       ui.NewHistoryModel(historyStore),
		playlist:      ui.NewPlaylistModel(playlistStore),
		mpv:           mpv,
		historyStore:  historyStore,
		playlistStore: playlistStore,
		config:        cfg,
		configPath:    configPath,
	}

	if initialURL != "" {
		m.initialURL = initialURL
		m.currentView = ViewPlayer
	}

	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.search.Init()}

	// URL が指定されている場合、即時再生を開始する
	if m.initialURL != "" {
		videoURL := m.initialURL
		cmds = append(cmds, func() tea.Msg {
			return ui.PlayVideoMsg{
				Video: youtube.Video{
					Title: "読み込み中...",
					URL:   videoURL,
				},
			}
		})
		// メタデータを非同期で取得して表示を更新
		cmds = append(cmds, ui.FetchVideoMetadata(m.initialURL))
	}

	return tea.Batch(cmds...)
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
		var cmd tea.Cmd
		m.player, cmd = m.player.PlayVideo(msg.Video)
		cmds := []tea.Cmd{cmd}
		// Video.ID がある場合のみ即座に履歴保存（直接 URL 再生時は ID が空なのでスキップ）
		if msg.Video.ID != "" {
			cmds = append(cmds, ui.AddToHistory(m.historyStore, msg.Video))
		}
		// Fetch related videos async based on the video title
		cmds = append(cmds, ui.FetchRelatedVideos(msg.Video.Title))
		return m, tea.Batch(cmds...)

	case ui.StreamURLResolvedMsg:
		// ストリーム URL 解決結果を player に転送
		var cmd tea.Cmd
		m.player, cmd = m.player.Update(msg)
		if msg.Err != nil && !errors.Is(msg.Err, context.Canceled) {
			if errors.Is(msg.Err, context.DeadlineExceeded) {
				m.statusMsg = "ストリームURLの取得がタイムアウトしました"
			} else {
				m.statusMsg = fmt.Sprintf("URL解決エラー: %v", msg.Err)
			}
		}
		return m, cmd

	case ui.MpvExecFinishedMsg:
		// ターミナル VO でのフォアグラウンド再生が終了 → player に転送
		var cmd tea.Cmd
		m.player, cmd = m.player.Update(msg)
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("再生エラー: %v", msg.Err)
		}
		return m, cmd

	case ui.PlayerTickMsg:
		// Forward player tick to player model (state polling)
		var cmd tea.Cmd
		m.player, cmd = m.player.Update(msg)
		return m, cmd

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

	case ui.VideoMetadataMsg:
		// Forward video metadata to player model
		var cmd tea.Cmd
		m.player, cmd = m.player.Update(msg)
		cmds := []tea.Cmd{cmd}
		// メタデータ取得成功時、ID を取得できたら履歴保存と関連動画再取得
		if msg.Err == nil && msg.Video.ID != "" {
			cmds = append(cmds, ui.AddToHistory(m.historyStore, msg.Video))
			cmds = append(cmds, ui.FetchRelatedVideos(msg.Video.Title))
		}
		return m, tea.Batch(cmds...)

	case ui.ChatMessageMsg:
		// Forward chat messages to player model
		var cmd tea.Cmd
		m.player, cmd = m.player.Update(msg)
		return m, cmd

	case ui.HistorySavedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("履歴保存エラー: %v", msg.Err)
		}
		return m, nil

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
