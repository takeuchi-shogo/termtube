package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takeuchishougo/termtube/internal/storage"
	"github.com/takeuchishougo/termtube/internal/youtube"
)

// PlaylistsLoadedMsg is sent when playlist names have been loaded.
type PlaylistsLoadedMsg struct {
	Names []string
	Err   error
}

// PlaylistVideosLoadedMsg is sent when a playlist's videos have been loaded.
type PlaylistVideosLoadedMsg struct {
	Playlist storage.Playlist
	Err      error
}

// VideoAddedToPlaylistMsg is sent after adding a video to a playlist.
type VideoAddedToPlaylistMsg struct {
	PlaylistName string
	VideoTitle   string
	Err          error
}

// playlistItem implements list.DefaultItem for displaying playlist names.
type playlistItem struct {
	name       string
	videoCount int
}

func (p playlistItem) Title() string       { return p.name }
func (p playlistItem) FilterValue() string { return p.name }
func (p playlistItem) Description() string {
	return fmt.Sprintf("%d 本の動画", p.videoCount)
}

// playlistVideoItem implements list.DefaultItem for displaying videos in a playlist.
type playlistVideoItem struct {
	video storage.PlaylistVideo
}

func (v playlistVideoItem) Title() string       { return v.video.Title }
func (v playlistVideoItem) FilterValue() string { return v.video.Title }
func (v playlistVideoItem) Description() string { return v.video.Channel }

// PlaylistModel manages playlist UI with 2-level navigation: playlist list -> video list.
type PlaylistModel struct {
	store            *storage.PlaylistStore
	plList           list.Model
	videoList        list.Model
	viewingPlaylist  string // empty = PL list, non-empty = video list
	width            int
	height           int
}

// NewPlaylistModel creates a new PlaylistModel with empty lists.
func NewPlaylistModel(store *storage.PlaylistStore) PlaylistModel {
	delegate := list.NewDefaultDelegate()

	plList := list.New([]list.Item{}, delegate, 0, 0)
	plList.Title = "プレイリスト"
	plList.SetShowStatusBar(true)
	plList.SetFilteringEnabled(false)
	plList.SetShowHelp(false)

	videoList := list.New([]list.Item{}, delegate, 0, 0)
	videoList.Title = ""
	videoList.SetShowStatusBar(true)
	videoList.SetFilteringEnabled(false)
	videoList.SetShowHelp(false)

	return PlaylistModel{
		store:     store,
		plList:    plList,
		videoList: videoList,
	}
}

// Refresh reloads playlists from store.
func (m PlaylistModel) Refresh() tea.Cmd {
	store := m.store
	return func() tea.Msg {
		names, err := store.List()
		return PlaylistsLoadedMsg{Names: names, Err: err}
	}
}

// loadPlaylistVideos loads videos for a specific playlist.
func (m PlaylistModel) loadPlaylistVideos(name string) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		pl, err := store.Load(name)
		return PlaylistVideosLoadedMsg{Playlist: pl, Err: err}
	}
}

// Update handles messages for the playlist model.
func (m PlaylistModel) Update(msg tea.Msg) (PlaylistModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.plList.SetSize(msg.Width, msg.Height-2)
		m.videoList.SetSize(msg.Width, msg.Height-2)
		return m, nil

	case PlaylistsLoadedMsg:
		if msg.Err != nil {
			m.plList.Title = fmt.Sprintf("エラー: %v", msg.Err)
			return m, nil
		}
		// Load each playlist to get video counts
		store := m.store
		items := make([]list.Item, 0, len(msg.Names))
		for _, name := range msg.Names {
			pl, err := store.Load(name)
			count := 0
			if err == nil {
				count = len(pl.Videos)
			}
			items = append(items, playlistItem{name: name, videoCount: count})
		}
		cmd := m.plList.SetItems(items)
		m.plList.Title = fmt.Sprintf("プレイリスト (%d件)", len(msg.Names))
		return m, cmd

	case PlaylistVideosLoadedMsg:
		if msg.Err != nil {
			m.videoList.Title = fmt.Sprintf("エラー: %v", msg.Err)
			return m, nil
		}
		m.viewingPlaylist = msg.Playlist.Name
		items := make([]list.Item, len(msg.Playlist.Videos))
		for i, v := range msg.Playlist.Videos {
			items[i] = playlistVideoItem{video: v}
		}
		cmd := m.videoList.SetItems(items)
		m.videoList.Title = fmt.Sprintf("📋 %s (%d本)", msg.Playlist.Name, len(msg.Playlist.Videos))
		return m, cmd

	case tea.KeyMsg:
		if m.viewingPlaylist == "" {
			// PL list view
			switch msg.String() {
			case "enter":
				item := m.plList.SelectedItem()
				if item == nil {
					return m, nil
				}
				if pi, ok := item.(playlistItem); ok {
					return m, m.loadPlaylistVideos(pi.name)
				}
			}

			var cmd tea.Cmd
			m.plList, cmd = m.plList.Update(msg)
			return m, cmd
		}

		// Video list view
		switch msg.String() {
		case "enter":
			item := m.videoList.SelectedItem()
			if item == nil {
				return m, nil
			}
			if vi, ok := item.(playlistVideoItem); ok {
				video := youtube.Video{
					ID:      vi.video.VideoID,
					Title:   vi.video.Title,
					Channel: vi.video.Channel,
					URL:     fmt.Sprintf("https://www.youtube.com/watch?v=%s", vi.video.VideoID),
				}
				return m, func() tea.Msg {
					return PlayVideoMsg{Video: video}
				}
			}
		case "esc", "backspace":
			m.viewingPlaylist = ""
			return m, m.Refresh()
		case "d":
			// Delete selected video from playlist
			item := m.videoList.SelectedItem()
			if item == nil {
				return m, nil
			}
			if vi, ok := item.(playlistVideoItem); ok {
				plName := m.viewingPlaylist
				videoID := vi.video.VideoID
				store := m.store
				return m, func() tea.Msg {
					pl, err := store.Load(plName)
					if err != nil {
						return PlaylistVideosLoadedMsg{Err: err}
					}
					// Remove the video
					filtered := make([]storage.PlaylistVideo, 0, len(pl.Videos))
					for _, v := range pl.Videos {
						if v.VideoID != videoID {
							filtered = append(filtered, v)
						}
					}
					pl.Videos = filtered
					if err := store.Save(pl); err != nil {
						return PlaylistVideosLoadedMsg{Err: err}
					}
					return PlaylistVideosLoadedMsg{Playlist: pl}
				}
			}
		}

		var cmd tea.Cmd
		m.videoList, cmd = m.videoList.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the playlist screen.
func (m PlaylistModel) View() string {
	if m.viewingPlaylist != "" {
		// Video list view
		if len(m.videoList.Items()) == 0 {
			empty := lipgloss.NewStyle().
				Padding(2, 4).
				Render(fmt.Sprintf("「%s」にはまだ動画がありません", m.viewingPlaylist))
			help := HelpStyle.Render("esc: プレイリスト一覧に戻る | q: 終了")
			return lipgloss.JoinVertical(lipgloss.Left, empty, help)
		}
		help := HelpStyle.Render("j/k: 移動 | enter: 再生 | d: 削除 | esc: 戻る | q: 終了")
		return lipgloss.JoinVertical(lipgloss.Left, m.videoList.View(), help)
	}

	// PL list view
	if len(m.plList.Items()) == 0 {
		empty := lipgloss.NewStyle().
			Padding(2, 4).
			Render("まだプレイリストはありません\n検索画面で a キーを押すと「お気に入り」に追加できます")
		help := HelpStyle.Render("1: 検索に戻る | q: 終了")
		return lipgloss.JoinVertical(lipgloss.Left, empty, help)
	}

	help := HelpStyle.Render("j/k: 移動 | enter: 開く | 1: 検索に戻る | q: 終了")
	return lipgloss.JoinVertical(lipgloss.Left, m.plList.View(), help)
}

// AddVideoToPlaylist adds a video to the named playlist.
// If the playlist does not exist, it creates it first.
func AddVideoToPlaylist(store *storage.PlaylistStore, playlistName string, video youtube.Video) tea.Cmd {
	return func() tea.Msg {
		// Try loading; if not found, create
		_, err := store.Load(playlistName)
		if err != nil {
			pl := storage.Playlist{
				Name:   playlistName,
				Videos: []storage.PlaylistVideo{},
			}
			if saveErr := store.Save(pl); saveErr != nil {
				return VideoAddedToPlaylistMsg{Err: saveErr}
			}
		}

		pv := storage.PlaylistVideo{
			VideoID: video.ID,
			Title:   video.Title,
			Channel: video.Channel,
		}
		addErr := store.AddVideo(playlistName, pv)
		return VideoAddedToPlaylistMsg{
			PlaylistName: playlistName,
			VideoTitle:   video.Title,
			Err:          addErr,
		}
	}
}
