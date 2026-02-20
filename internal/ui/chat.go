package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/takeuchishougo/termtube/internal/youtube"
)

const defaultMaxChatMessages = 50

// ChatMessageMsg はライブチャットメッセージ受信時に送信される。
type ChatMessageMsg struct {
	Message youtube.ChatMessage
}

// ChatModel はライブチャットをスクロール表示するモデル。
type ChatModel struct {
	messages    []youtube.ChatMessage
	visible     bool
	maxMessages int
	width       int
	height      int
}

// NewChatModel は新しい ChatModel を作成する。
func NewChatModel(maxMessages int) ChatModel {
	if maxMessages <= 0 {
		maxMessages = defaultMaxChatMessages
	}
	return ChatModel{
		messages:    make([]youtube.ChatMessage, 0),
		visible:     false,
		maxMessages: maxMessages,
	}
}

// AddMessage はチャットメッセージを追加し、最大件数を超えた分を削除する。
func (m *ChatModel) AddMessage(msg youtube.ChatMessage) {
	m.messages = append(m.messages, msg)
	if len(m.messages) > m.maxMessages {
		m.messages = m.messages[len(m.messages)-m.maxMessages:]
	}
}

// Toggle はチャットの表示/非表示を切り替える。
func (m *ChatModel) Toggle() {
	m.visible = !m.visible
}

// IsVisible はチャットが表示中かどうかを返す。
func (m ChatModel) IsVisible() bool {
	return m.visible
}

// SetSize はチャットパネルのサイズを設定する。
func (m *ChatModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Clear はチャットメッセージをすべてクリアする。
func (m *ChatModel) Clear() {
	m.messages = make([]youtube.ChatMessage, 0)
}

// View はチャットパネルを描画する。
func (m ChatModel) View() string {
	if !m.visible {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	authorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("81"))

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	title := titleStyle.Render("-- Live Chat --")

	if len(m.messages) == 0 {
		noMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("チャットメッセージはありません")
		return lipgloss.JoinVertical(lipgloss.Left, title, noMsg)
	}

	// 表示可能な行数を計算（タイトル行を除く）
	visibleLines := m.height - 2
	if visibleLines <= 0 {
		visibleLines = 10
	}

	// 末尾からvisibleLines分を表示
	start := 0
	if len(m.messages) > visibleLines {
		start = len(m.messages) - visibleLines
	}

	var lines []string
	for _, msg := range m.messages[start:] {
		author := authorStyle.Render(msg.Author)
		text := messageStyle.Render(msg.Message)
		lines = append(lines, fmt.Sprintf("%s: %s", author, text))
	}

	chatContent := strings.Join(lines, "\n")

	chatBox := BorderStyle.
		Width(m.width - 4).
		Render(chatContent)

	return lipgloss.JoinVertical(lipgloss.Left, title, chatBox)
}
