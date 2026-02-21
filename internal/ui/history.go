package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takeuchishougo/termtube/internal/storage"
	"github.com/takeuchishougo/termtube/internal/youtube"
)

// HistoryLoadedMsg is sent when history entries have been loaded from store.
type HistoryLoadedMsg struct {
	Entries []storage.HistoryEntry
	Err     error
}

// historyItem implements list.DefaultItem for displaying history entries.
type historyItem struct {
	entry storage.HistoryEntry
}

func (h historyItem) Title() string       { return h.entry.Title }
func (h historyItem) FilterValue() string { return h.entry.Title }
func (h historyItem) Description() string {
	watchedAt := h.entry.WatchedAt.Local().Format("2006/01/02 15:04")
	return fmt.Sprintf("%s  %s", h.entry.Channel, watchedAt)
}

// HistoryModel displays watch history using Bubbles list.
type HistoryModel struct {
	list   list.Model
	store  *storage.HistoryStore
	width  int
	height int
}

// NewHistoryModel creates a new HistoryModel with an empty list.
func NewHistoryModel(store *storage.HistoryStore) HistoryModel {
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "視聴履歴"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return HistoryModel{
		list:  l,
		store: store,
	}
}

// Refresh reloads history from store and returns a command that sends HistoryLoadedMsg.
func (m HistoryModel) Refresh() tea.Cmd {
	store := m.store
	return func() tea.Msg {
		entries, err := store.Load()
		return HistoryLoadedMsg{Entries: entries, Err: err}
	}
}

// Update handles messages for the history model.
func (m HistoryModel) Update(msg tea.Msg) (HistoryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h := msg.Height - 2
		if h < 1 {
			h = 1
		}
		m.list.SetSize(msg.Width, h)
		return m, nil

	case HistoryLoadedMsg:
		if msg.Err != nil {
			m.list.Title = fmt.Sprintf("エラー: %v", msg.Err)
			return m, nil
		}
		items := make([]list.Item, len(msg.Entries))
		for i, e := range msg.Entries {
			items[i] = historyItem{entry: e}
		}
		cmd := m.list.SetItems(items)
		m.list.Title = fmt.Sprintf("視聴履歴 (%d件)", len(msg.Entries))
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			item := m.list.SelectedItem()
			if item == nil {
				return m, nil
			}
			if hi, ok := item.(historyItem); ok {
				video := youtube.Video{
					ID:      hi.entry.VideoID,
					Title:   hi.entry.Title,
					Channel: hi.entry.Channel,
					URL:     youtube.WatchURL(hi.entry.VideoID),
				}
				return m, func() tea.Msg {
					return PlayVideoMsg{Video: video}
				}
			}
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the history screen.
func (m HistoryModel) View() string {
	if len(m.list.Items()) == 0 {
		empty := lipgloss.NewStyle().
			Padding(2, 4).
			Render("まだ視聴履歴はありません")
		help := HelpStyle.Render("1: 検索に戻る | q: 終了")
		return lipgloss.JoinVertical(lipgloss.Left, empty, help)
	}

	help := HelpStyle.Render("j/k: 移動 | enter: 再生 | 1: 検索に戻る | q: 終了")
	return lipgloss.JoinVertical(lipgloss.Left, m.list.View(), help)
}

// HistorySavedMsg は履歴保存結果を伝える。
type HistorySavedMsg struct {
	Err error
}

// AddToHistory creates a HistoryEntry from a video and adds it to the store.
func AddToHistory(store *storage.HistoryStore, video youtube.Video) tea.Cmd {
	return func() tea.Msg {
		entry := storage.HistoryEntry{
			VideoID:   video.ID,
			Title:     video.Title,
			Channel:   video.Channel,
			WatchedAt: time.Now(),
			Duration:  video.Duration,
		}
		err := store.Add(entry)
		return HistorySavedMsg{Err: err}
	}
}
