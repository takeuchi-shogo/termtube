package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takeuchishougo/termtube/internal/youtube"
)

// SearchResultMsg carries search results back to the model.
type SearchResultMsg struct {
	Videos []youtube.Video
	Err    error
}

// videoItem implements list.DefaultItem for displaying search results.
type videoItem struct{ video youtube.Video }

func (v videoItem) Title() string       { return v.video.Title }
func (v videoItem) FilterValue() string { return v.video.Title }
func (v videoItem) Description() string {
	dur := v.video.DurationStr
	if dur == "" {
		dur = "LIVE"
	}
	return fmt.Sprintf("%s  %s", v.video.Channel, dur)
}

// SearchModel is a Bubble Tea sub-model for the search screen.
type SearchModel struct {
	textInput  textinput.Model
	resultList list.Model
	searching  bool
	inputMode  bool
	width      int
	height     int
}

// NewSearchModel creates a new SearchModel with a focused text input and empty result list.
func NewSearchModel() SearchModel {
	ti := textinput.New()
	ti.Placeholder = "YouTube検索..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 60

	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "検索結果"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return SearchModel{
		textInput:  ti,
		resultList: l,
		inputMode:  true,
	}
}

// Init returns the initial command for the search model.
func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

// IsInputMode returns whether the text input is currently focused.
func (m SearchModel) IsInputMode() bool {
	return m.inputMode
}

// SelectedVideo returns the currently selected video, or nil if none.
func (m SearchModel) SelectedVideo() *youtube.Video {
	item := m.resultList.SelectedItem()
	if item == nil {
		return nil
	}
	if vi, ok := item.(videoItem); ok {
		v := vi.video
		return &v
	}
	return nil
}

// Update handles messages for the search model.
func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve space for search bar (3 lines) + help bar (2 lines)
		listHeight := m.height - 6
		if listHeight < 1 {
			listHeight = 1
		}
		m.resultList.SetSize(m.width, listHeight)
		return m, nil

	case SearchResultMsg:
		m.searching = false
		if msg.Err != nil {
			m.resultList.Title = fmt.Sprintf("エラー: %v", msg.Err)
			return m, nil
		}
		items := make([]list.Item, len(msg.Videos))
		for i, v := range msg.Videos {
			items[i] = videoItem{video: v}
		}
		cmd := m.resultList.SetItems(items)
		m.resultList.Title = fmt.Sprintf("検索結果 (%d件)", len(msg.Videos))
		m.inputMode = false
		m.textInput.Blur()
		return m, cmd

	case tea.KeyMsg:
		if m.inputMode {
			switch msg.String() {
			case "enter":
				query := m.textInput.Value()
				if query != "" {
					m.searching = true
					return m, doSearch(query)
				}
				return m, nil
			case "esc":
				// Switch to list mode if there are results
				if len(m.resultList.Items()) > 0 {
					m.inputMode = false
					m.textInput.Blur()
				}
				return m, nil
			}
			// Forward key to textinput
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// List mode key handling
		switch msg.String() {
		case "/":
			m.inputMode = true
			m.textInput.Focus()
			return m, textinput.Blink
		}

		// Forward key to list
		var cmd tea.Cmd
		m.resultList, cmd = m.resultList.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

// View renders the search screen.
func (m SearchModel) View() string {
	var searchBar string
	if m.searching {
		searchBar = BorderStyle.Render(m.textInput.View() + "  検索中...")
	} else {
		searchBar = BorderStyle.Render(m.textInput.View())
	}

	resultView := m.resultList.View()

	var helpText string
	if m.inputMode {
		helpText = "Enter: 検索 | Esc: リストに戻る"
	} else {
		helpText = "/: 検索 | j/k: 移動 | enter: 選択 | q: 終了"
	}
	help := HelpStyle.Render(helpText)

	return lipgloss.JoinVertical(lipgloss.Left, searchBar, resultView, help)
}

// doSearch returns an async command that calls youtube.Search.
func doSearch(query string) tea.Cmd {
	return func() tea.Msg {
		videos, err := youtube.Search(context.Background(), query, 10)
		return SearchResultMsg{Videos: videos, Err: err}
	}
}
