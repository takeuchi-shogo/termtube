package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takeuchishougo/termtube/internal/player"
	"github.com/takeuchishougo/termtube/internal/youtube"
)

// PlayerTickMsg は定期的な状態ポーリングのための内部メッセージ。
type PlayerTickMsg struct{}

// MpvExecFinishedMsg はターミナル VO でのフォアグラウンド mpv 再生が終了したときのメッセージ。
type MpvExecFinishedMsg struct {
	Err error
}

// StreamURLResolvedMsg はストリーム URL 解決の結果を伝えるメッセージ。
type StreamURLResolvedMsg struct {
	StreamURL   string
	OriginalURL string // 鮮度チェック用（動画切替対策）
	Err         error
}

// ViewMode は再生画面の表示モードを表す。
type ViewMode int

const (
	// ModeFocus は集中視聴モード: 大きな動画情報表示。
	ModeFocus ViewMode = iota
	// ModeBGV は BGV モード: コンパクトなプレイヤー情報（1行表示）。
	ModeBGV
)

// ViewModeFromString は文字列から ViewMode を返す。
func ViewModeFromString(s string) ViewMode {
	switch s {
	case "bgv":
		return ModeBGV
	default:
		return ModeFocus
	}
}

// String は ViewMode の文字列表現を返す。
func (v ViewMode) String() string {
	switch v {
	case ModeBGV:
		return "bgv"
	default:
		return "focus"
	}
}

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

// VideoMetadataMsg carries fetched video metadata for direct URL playback.
type VideoMetadataMsg struct {
	Video        youtube.Video
	RequestedURL string // リクエスト元の URL（順序逆転対策）
	Err          error
}

// PlayerModel is a Bubble Tea sub-model that displays current playback info and controls mpv.
type PlayerModel struct {
	mpv             *player.MpvPlayer
	current         *youtube.Video
	state           player.PlayerState
	chat            ChatModel
	relatedVideos   []youtube.Video
	showRelated     bool
	relatedCursor   int
	viewMode        ViewMode
	width           int
	height          int
	resolving       bool               // URL解決中フラグ
	resolvingCancel context.CancelFunc // 解決のキャンセル関数
	resolvingDots   int                // ローディングアニメーション用カウンタ
}

// NewPlayerModel creates a new PlayerModel with the given MpvPlayer.
func NewPlayerModel(mpv *player.MpvPlayer) PlayerModel {
	return PlayerModel{
		mpv:      mpv,
		state:    mpv.GetState(),
		chat:     NewChatModel(defaultMaxChatMessages),
		viewMode: ModeFocus,
	}
}

// SetViewMode は表示モードを設定する。
func (m *PlayerModel) SetViewMode(mode ViewMode) {
	m.viewMode = mode
}

// GetViewMode は現在の表示モードを返す。
func (m PlayerModel) GetViewMode() ViewMode {
	return m.viewMode
}

// PlayerTick は 500ms ごとに PlayerTickMsg を送る Bubble Tea コマンドを返す。
func PlayerTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return PlayerTickMsg{}
	})
}

// Update handles messages for the player model.
func (m PlayerModel) Update(msg tea.Msg) (PlayerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.chat.SetSize(msg.Width, msg.Height/3)
		return m, nil

	case PlayerTickMsg:
		if m.resolving {
			// URL 解決中: ローディングアニメーションを更新
			m.resolvingDots = (m.resolvingDots + 1) % 4
			return m, PlayerTick()
		}
		// mpv から最新の再生状態を取得して表示を更新
		if m.mpv != nil && m.current != nil {
			m.state = m.mpv.GetState()
		}
		return m, PlayerTick()

	case PlayerStateMsg:
		m.state = msg.State
		return m, nil

	case MpvExecFinishedMsg:
		// ターミナル VO でのフォアグラウンド再生が終了 → Stopped にリセット
		m.state.State = player.StateStopped
		return m, nil

	case StreamURLResolvedMsg:
		// resolving フラグが false の場合はキャンセル済みとして破棄
		if !m.resolving {
			return m, nil
		}
		// 鮮度チェック: 動画が切り替わっていたら破棄
		if m.current == nil || m.current.URL != msg.OriginalURL {
			return m, nil
		}
		m.resolving = false
		if m.resolvingCancel != nil {
			m.resolvingCancel()
			m.resolvingCancel = nil
		}
		if msg.Err != nil || msg.StreamURL == "" {
			m.state.State = player.StateStopped
			return m, nil
		}
		// Phase 2: 解決済み URL で mpv をフォアグラウンド起動
		cmd := m.mpv.BuildForegroundCmd(msg.StreamURL)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return MpvExecFinishedMsg{Err: err}
		})

	case ChatMessageMsg:
		m.chat.AddMessage(msg.Message)
		return m, nil

	case RelatedVideosMsg:
		if msg.Err == nil {
			m.relatedVideos = msg.Videos
		}
		return m, nil

	case VideoMetadataMsg:
		// 順序逆転対策: リクエスト元 URL が現在の動画と一致する場合のみ適用
		if msg.Err == nil && m.current != nil && m.current.URL == msg.RequestedURL {
			// URL はメタデータの webpage_url で上書きされるため保持
			currentURL := m.current.URL
			*m.current = msg.Video
			if m.current.URL == "" {
				m.current.URL = currentURL
			}
		}
		return m, nil

	case tea.KeyMsg:
		// URL 解決中: Esc でキャンセル、他のキーはブロック
		if m.resolving {
			if msg.String() == "esc" {
				if m.resolvingCancel != nil {
					m.resolvingCancel()
				}
				m.resolving = false
				m.resolvingCancel = nil
				m.state.State = player.StateStopped
			}
			return m, nil
		}

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
		case "b":
			// Toggle between Focus and BGV mode
			if m.viewMode == ModeFocus {
				m.viewMode = ModeBGV
			} else {
				m.viewMode = ModeFocus
			}
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

	if m.resolving {
		return m.viewResolving()
	}

	if m.viewMode == ModeBGV {
		return m.viewBGV()
	}

	return m.viewFocus()
}

// viewResolving はストリーム URL 解決中のローディング画面を描画する。
func (m PlayerModel) viewResolving() string {
	title := TitleStyle.Render(m.current.Title)
	channel := SubtitleStyle.Render(m.current.Channel)

	dots := strings.Repeat(".", m.resolvingDots)
	loadingStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))
	loading := loadingStyle.Render(fmt.Sprintf("ストリームURL解決中%s", dots))

	help := HelpStyle.Render("Esc: キャンセル")

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		title,
		channel,
		"",
		loading,
		"",
		help,
	)
}

// viewFocus は集中視聴モードの表示を描画する。
func (m PlayerModel) viewFocus() string {
	// Video title
	title := TitleStyle.Render(m.current.Title)

	// Channel name
	channel := SubtitleStyle.Render(m.current.Channel)

	// State icon + position / duration
	stateIcon := m.stateIcon()

	posStr := formatTime(int(m.state.Position))
	durStr := formatTime(int(m.state.Duration))
	playbackInfo := fmt.Sprintf("%s  %s / %s", stateIcon, posStr, durStr)

	// Volume bar
	volumeBar := renderVolumeBar(m.state.Volume, m.width)

	// Mode indicator
	modeIndicator := SubtitleStyle.Render("[Focus Mode]")

	// Help text
	help := HelpStyle.Render("Space: 再生/一時停止 | ←/→: シーク ±5秒 | ↑/↓: 音量 ±5 | c: チャット | Tab: 関連 | b: BGVモード | 1: 検索に戻る | q: 終了")

	parts := []string{
		"",
		title,
		channel,
		"",
		playbackInfo,
		volumeBar,
		modeIndicator,
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

// viewBGV は BGV モードの表示を描画する。
// コンパクトな1行表示で、TUI のスペースを多く残す。
func (m PlayerModel) viewBGV() string {
	stateIcon := m.stateIcon()
	posStr := formatTime(int(m.state.Position))
	durStr := formatTime(int(m.state.Duration))

	// タイトルを短縮表示（幅に合わせて切り詰め）
	titleText := m.current.Title
	maxTitleLen := m.width - 40
	if maxTitleLen < 10 {
		maxTitleLen = 10
	}
	titleRunes := []rune(titleText)
	if len(titleRunes) > maxTitleLen {
		titleText = string(titleRunes[:maxTitleLen]) + "..."
	}

	// コンパクトな1行プレイヤー情報
	compactLine := fmt.Sprintf("%s %s | %s/%s | Vol:%d%%",
		stateIcon, titleText, posStr, durStr, m.state.Volume)

	compactStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		Padding(0, 1)

	modeIndicator := SubtitleStyle.Render("[BGV Mode]")

	// BGV モードのヘルプ（コンパクト）
	help := HelpStyle.Render("Space: ⏯ | ←/→: シーク | ↑/↓: 音量 | c: チャット | Tab: 関連 | b: Focusモード | 1: 検索 | q: 終了")

	parts := []string{
		compactStyle.Render(compactLine),
		modeIndicator,
	}

	// Chat panel (BGV mode でも表示可能)
	chatView := m.chat.View()
	if chatView != "" {
		parts = append(parts, chatView)
	}

	// Related videos panel
	if m.showRelated {
		relatedView := m.renderRelated()
		parts = append(parts, relatedView)
	}

	parts = append(parts, help)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// stateIcon は現在の再生状態に対応するアイコンを返す。
func (m PlayerModel) stateIcon() string {
	switch m.state.State {
	case player.StatePlaying:
		return "▶"
	case player.StatePaused:
		return "⏸"
	case player.StateStopped:
		return "⏹"
	default:
		return "?"
	}
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

// PlayVideo は動画の再生を開始し、更新された PlayerModel と非同期コマンドを返す。
// Bubble Tea の値セマンティクスに従い、ポインタレシーバではなく値レシーバで
// 更新後のモデルを返す。
func (m PlayerModel) PlayVideo(video youtube.Video) (PlayerModel, tea.Cmd) {
	m.current = &video
	m.chat.Clear()
	m.relatedVideos = nil
	m.relatedCursor = 0

	if m.mpv.IsTerminalVO() {
		// ターミナル VO: 2フェーズ方式
		// Phase 1: yt-dlp でストリーム URL を非同期解決（TUI はレスポンシブなまま）
		// Phase 2: 解決後に tea.ExecProcess で mpv を起動
		if m.resolvingCancel != nil {
			m.resolvingCancel()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		m.resolving = true
		m.resolvingCancel = cancel
		m.resolvingDots = 0
		videoURL := video.URL
		resolveCmd := func() tea.Msg {
			defer cancel() // context リソースを必ず解放
			streamURL, err := youtube.GetStreamURL(ctx, videoURL)
			return StreamURLResolvedMsg{
				StreamURL:   streamURL,
				OriginalURL: videoURL,
				Err:         err,
			}
		}
		return m, tea.Batch(resolveCmd, PlayerTick())
	}

	// IPC VO: 既存のバックグラウンド再生 + IPC 制御
	mpv := m.mpv
	url := video.URL
	playCmd := func() tea.Msg {
		err := mpv.Play(url)
		if err != nil {
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
	return m, tea.Batch(playCmd, PlayerTick())
}

// FetchRelatedVideos は関連動画を非同期で取得するコマンドを返す。
func FetchRelatedVideos(videoTitle string) tea.Cmd {
	return func() tea.Msg {
		videos, err := youtube.GetRelated(context.Background(), videoTitle, 10)
		return RelatedVideosMsg{Videos: videos, Err: err}
	}
}

// FetchVideoMetadata は URL から動画メタデータを非同期で取得するコマンドを返す。
func FetchVideoMetadata(videoURL string) tea.Cmd {
	return func() tea.Msg {
		video, err := youtube.GetMetadata(context.Background(), videoURL)
		return VideoMetadataMsg{Video: video, RequestedURL: videoURL, Err: err}
	}
}

// formatTime converts seconds to "MM:SS" or "HH:MM:SS" format.
func formatTime(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
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
