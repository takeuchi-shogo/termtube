package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takeuchishougo/termtube/internal/player"
	"github.com/takeuchishougo/termtube/internal/youtube"
)

// PlayerStateMsg wraps player.PlayerState, sent when mpv state changes.
type PlayerStateMsg struct {
	State player.PlayerState
}

// PlayVideoMsg is a message requesting playback of a youtube.Video.
type PlayVideoMsg struct {
	Video youtube.Video
}

// RelatedVideosMsg carries fetched related videos.
type RelatedVideosMsg struct {
	Videos []youtube.Video
	Err    error
}

// PlayerModel is a Bubble Tea sub-model that displays current playback info and controls mpv.
type PlayerModel struct {
	mpv           *player.MpvPlayer
	current       *youtube.Video
	state         player.PlayerState
	chat          ChatModel
	relatedVideos []youtube.Video
	showRelated   bool
	relatedCursor int
	width         int
	height        int
}

// NewPlayerModel creates a new PlayerModel with the given MpvPlayer.
func NewPlayerModel(mpv *player.MpvPlayer) PlayerModel {
	return PlayerModel{
		mpv:   mpv,
		state: mpv.GetState(),
		chat:  NewChatModel(defaultMaxChatMessages),
	}
}

// Update handles messages for the player model.
func (m PlayerModel) Update(msg tea.Msg) (PlayerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.chat.SetSize(msg.Width, msg.Height/3)
		return m, nil

	case PlayerStateMsg:
		m.state = msg.State
		return m, nil

	case ChatMessageMsg:
		m.chat.AddMessage(msg.Message)
		return m, nil

	case RelatedVideosMsg:
		if msg.Err == nil {
			m.relatedVideos = msg.Videos
		}
		return m, nil

	case tea.KeyMsg:
		// 関連動画パネルが表示中の場合、j/k/enter を関連動画リストに委譲
		if m.showRelated && len(m.relatedVideos) > 0 {
			switch msg.String() {
			case "j":
				if m.relatedCursor < len(m.relatedVideos)-1 {
					m.relatedCursor++
				}
				return m, nil
			case "k":
				if m.relatedCursor > 0 {
					m.relatedCursor--
				}
				return m, nil
			case "enter":
				video := m.relatedVideos[m.relatedCursor]
				return m, func() tea.Msg {
					return PlayVideoMsg{Video: video}
				}
			}
		}

		switch msg.String() {
		case " ":
			// Toggle pause
			if m.mpv != nil && m.current != nil {
				_ = m.mpv.TogglePause()
			}
			return m, nil
		case "left":
			// Seek -5s
			if m.mpv != nil && m.current != nil {
				_ = m.mpv.Seek(-5)
			}
			return m, nil
		case "right":
			// Seek +5s
			if m.mpv != nil && m.current != nil {
				_ = m.mpv.Seek(5)
			}
			return m, nil
		case "up":
			// Volume +5 (clamped 0-100)
			if m.mpv != nil {
				newVol := m.state.Volume + 5
				if newVol > 100 {
					newVol = 100
				}
				_ = m.mpv.SetVolume(newVol)
				m.state.Volume = newVol
			}
			return m, nil
		case "down":
			// Volume -5 (clamped 0-100)
			if m.mpv != nil {
				newVol := m.state.Volume - 5
				if newVol < 0 {
					newVol = 0
				}
				_ = m.mpv.SetVolume(newVol)
				m.state.Volume = newVol
			}
			return m, nil
		case "c":
			// Toggle chat visibility
			m.chat.Toggle()
			return m, nil
		case "tab":
			// Toggle related videos panel
			m.showRelated = !m.showRelated
			return m, nil
		}
	}

	return m, nil
}

// View renders the player screen.
func (m PlayerModel) View() string {
	if m.current == nil {
		return lipgloss.NewStyle().
			Padding(2, 4).
			Render("再生中の動画はありません")
	}

	// Video title
	title := TitleStyle.Render(m.current.Title)

	// Channel name
	channel := SubtitleStyle.Render(m.current.Channel)

	// State icon + position / duration
	var stateIcon string
	switch m.state.State {
	case player.StatePlaying:
		stateIcon = "▶"
	case player.StatePaused:
		stateIcon = "⏸"
	case player.StateStopped:
		stateIcon = "⏹"
	}

	posStr := formatTime(int(m.state.Position))
	durStr := formatTime(int(m.state.Duration))
	playbackInfo := fmt.Sprintf("%s  %s / %s", stateIcon, posStr, durStr)

	// Volume bar
	volumeBar := renderVolumeBar(m.state.Volume, m.width)

	// Help text
	help := HelpStyle.Render("Space: 再生/一時停止 | ←/→: シーク ±5秒 | ↑/↓: 音量 ±5 | c: チャット | Tab: 関連 | 1: 検索に戻る | q: 終了")

	parts := []string{
		"",
		title,
		channel,
		"",
		playbackInfo,
		volumeBar,
		"",
	}

	// Chat panel
	chatView := m.chat.View()
	if chatView != "" {
		parts = append(parts, chatView, "")
	}

	// Related videos panel
	if m.showRelated {
		relatedView := m.renderRelated()
		parts = append(parts, relatedView, "")
	}

	parts = append(parts, help)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderRelated は関連動画パネルを描画する。
func (m PlayerModel) renderRelated() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	header := titleStyle.Render("-- 関連動画 --")

	if len(m.relatedVideos) == 0 {
		noVideos := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("関連動画を取得中...")
		return lipgloss.JoinVertical(lipgloss.Left, header, noVideos)
	}

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170"))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	channelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	var lines []string
	maxDisplay := 10
	if len(m.relatedVideos) < maxDisplay {
		maxDisplay = len(m.relatedVideos)
	}

	for i := 0; i < maxDisplay; i++ {
		v := m.relatedVideos[i]
		dur := v.DurationStr
		if dur == "" {
			dur = "LIVE"
		}
		line := fmt.Sprintf("%s  [%s]", v.Title, dur)
		channelLine := channelStyle.Render(fmt.Sprintf("  %s", v.Channel))

		if i == m.relatedCursor {
			lines = append(lines, selectedStyle.Render("> "+line))
		} else {
			lines = append(lines, normalStyle.Render("  "+line))
		}
		lines = append(lines, channelLine)
	}

	content := strings.Join(lines, "\n")
	relatedBox := BorderStyle.
		Width(m.width - 4).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, header, relatedBox)
}

// PlayVideo sets the current video and returns an async command that calls mpv.Play.
func (m *PlayerModel) PlayVideo(video youtube.Video) tea.Cmd {
	m.current = &video
	m.chat.Clear()
	m.relatedVideos = nil
	m.relatedCursor = 0
	mpv := m.mpv
	url := video.URL
	return func() tea.Msg {
		err := mpv.Play(url)
		if err != nil {
			// Return stopped state on error
			return PlayerStateMsg{
				State: player.PlayerState{
					State: player.StateStopped,
				},
			}
		}
		return PlayerStateMsg{
			State: mpv.GetState(),
		}
	}
}

// FetchRelatedVideos は関連動画を非同期で取得するコマンドを返す。
func FetchRelatedVideos(videoTitle string) tea.Cmd {
	return func() tea.Msg {
		videos, err := youtube.GetRelated(context.Background(), videoTitle, 10)
		return RelatedVideosMsg{Videos: videos, Err: err}
	}
}

// formatTime converts seconds to "MM:SS" format.
func formatTime(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

// renderVolumeBar renders a visual volume bar.
func renderVolumeBar(volume int, width int) string {
	barWidth := 20
	if width > 0 && width < 40 {
		barWidth = width / 2
	}

	filled := volume * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("音量: %3d%% [%s]", volume, bar)
}
