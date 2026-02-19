# termtube Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.
>
> **前提:** すべてのコマンド・パスはリポジトリルート (`termtube/`) 基準。

**Goal:** WezTermターミナル内で完結するYouTube TUIプレイヤーをGoで実装する

**Architecture:** Bubble Tea (Elm Architecture) でTUIを構築。mpvをSixelモードの子プロセスとして起動しJSON IPCで制御。yt-dlpを子プロセスで呼び出しYouTube検索・URL解決を行う。WezTermのペイン分割で映像とTUIを共存させる。

**Tech Stack:** Go, Bubble Tea, Lip Gloss, Bubbles, mpv (IPC), yt-dlp

---

## Task 1: プロジェクト初期化とスケルトン

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/app/app.go`

**Step 1: Go module初期化**

```bash
go mod init github.com/takeuchishougo/termtube
```

**Step 2: 最小限のBubble Teaアプリを作成**

Create `main.go`:
```go
package main

import (
	"fmt"
	"os"

	"github.com/takeuchishougo/termtube/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(app.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

Create `internal/app/app.go`:
```go
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
```

**Step 3: 依存をインストール**

```bash
go mod tidy
```

**Step 4: ビルド&動作確認**

```bash
go run .
```
Expected: タブ付きのTUI画面が表示。数字キーで切替、qで終了。

**Step 5: コミット**

```bash
git add main.go internal/ go.mod go.sum
git commit -m "✨ feat: Bubble Teaスケルトンアプリを作成"
```

---

## Task 2: ストレージ層 — 設定・履歴・プレイリスト

**Files:**
- Create: `internal/storage/config.go`
- Create: `internal/storage/config_test.go`
- Create: `internal/storage/history.go`
- Create: `internal/storage/history_test.go`
- Create: `internal/storage/playlist.go`
- Create: `internal/storage/playlist_test.go`

**Step 1: 設定の型定義とテストを書く**

Create `internal/storage/config_test.go`:
```go
package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/takeuchishougo/termtube/internal/storage"
)

func TestDefaultConfig(t *testing.T) {
	cfg := storage.DefaultConfig()
	if cfg.Player.DefaultVolume != 80 {
		t.Errorf("expected default volume 80, got %d", cfg.Player.DefaultVolume)
	}
	if cfg.Player.DefaultMode != "focus" {
		t.Errorf("expected default mode 'focus', got %s", cfg.Player.DefaultMode)
	}
	if cfg.Search.Region != "JP" {
		t.Errorf("expected region 'JP', got %s", cfg.Search.Region)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := storage.DefaultConfig()
	cfg.Player.DefaultVolume = 50

	if err := storage.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := storage.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Player.DefaultVolume != 50 {
		t.Errorf("expected volume 50, got %d", loaded.Player.DefaultVolume)
	}
}

func TestLoadConfigCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "config.toml")

	cfg, err := storage.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Player.DefaultVolume != 80 {
		t.Errorf("expected default volume 80, got %d", cfg.Player.DefaultVolume)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file should have been created")
	}
}
```

**Step 2: テストが失敗することを確認**

```bash
go test ./internal/storage/ -v
```
Expected: FAIL (型が存在しない)

**Step 3: 設定の実装**

Create `internal/storage/config.go`:
```go
package storage

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Player PlayerConfig `toml:"player"`
	Chat   ChatConfig   `toml:"chat"`
	Search SearchConfig `toml:"search"`
}

type PlayerConfig struct {
	DefaultMode   string `toml:"default_mode"`
	DefaultVolume int    `toml:"default_volume"`
	SixelQuality  string `toml:"sixel_quality"`
}

type ChatConfig struct {
	Enabled  bool `toml:"enabled"`
	MaxLines int  `toml:"max_lines"`
}

type SearchConfig struct {
	ResultsPerPage int    `toml:"results_per_page"`
	Region         string `toml:"region"`
}

func DefaultConfig() Config {
	return Config{
		Player: PlayerConfig{
			DefaultMode:   "focus",
			DefaultVolume: 80,
			SixelQuality:  "medium",
		},
		Chat: ChatConfig{
			Enabled:  true,
			MaxLines: 50,
		},
		Search: SearchConfig{
			ResultsPerPage: 20,
			Region:         "JP",
		},
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := SaveConfig(path, cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func SaveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
```

**Step 4: テスト通過を確認**

```bash
go mod tidy && go test ./internal/storage/ -v -run TestConfig
```
Expected: PASS

**Step 5: 履歴のテストを書く**

Create `internal/storage/history_test.go`:
```go
package storage_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/takeuchishougo/termtube/internal/storage"
)

func TestAddAndLoadHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	store := storage.NewHistoryStore(path)

	entry := storage.HistoryEntry{
		VideoID:   "abc123",
		Title:     "テスト動画",
		Channel:   "テストチャンネル",
		WatchedAt: time.Now(),
		Duration:  300,
		Position:  150,
	}

	if err := store.Add(entry); err != nil {
		t.Fatal(err)
	}

	entries, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].VideoID != "abc123" {
		t.Errorf("expected video_id abc123, got %s", entries[0].VideoID)
	}
}

func TestHistoryMaxEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	store := storage.NewHistoryStore(path)

	for i := 0; i < 150; i++ {
		store.Add(storage.HistoryEntry{VideoID: fmt.Sprintf("vid%d", i)})
	}

	entries, _ := store.Load()
	if len(entries) > 100 {
		t.Errorf("history should cap at 100, got %d", len(entries))
	}
}
```

**Step 6: 履歴の実装**

Create `internal/storage/history.go`:
```go
package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const maxHistoryEntries = 100

type HistoryEntry struct {
	VideoID   string    `json:"video_id"`
	Title     string    `json:"title"`
	Channel   string    `json:"channel"`
	WatchedAt time.Time `json:"watched_at"`
	Duration  int       `json:"duration"`
	Position  int       `json:"position"`
}

type HistoryStore struct {
	path string
}

func NewHistoryStore(path string) *HistoryStore {
	return &HistoryStore{path: path}
}

func (h *HistoryStore) Load() ([]HistoryEntry, error) {
	data, err := os.ReadFile(h.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []HistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (h *HistoryStore) Add(entry HistoryEntry) error {
	entries, err := h.Load()
	if err != nil {
		entries = nil
	}
	filtered := make([]HistoryEntry, 0, len(entries))
	for _, e := range entries {
		if e.VideoID != entry.VideoID {
			filtered = append(filtered, e)
		}
	}
	entries = append([]HistoryEntry{entry}, filtered...)
	if len(entries) > maxHistoryEntries {
		entries = entries[:maxHistoryEntries]
	}
	return h.save(entries)
}

func (h *HistoryStore) save(entries []HistoryEntry) error {
	if err := os.MkdirAll(filepath.Dir(h.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.path, data, 0o644)
}
```

**Step 7: テスト通過確認**

```bash
go test ./internal/storage/ -v -run TestHistory
```
Expected: PASS

**Step 8: プレイリストのテストを書く**

Create `internal/storage/playlist_test.go`:
```go
package storage_test

import (
	"testing"

	"github.com/takeuchishougo/termtube/internal/storage"
)

func TestCreateAndLoadPlaylist(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewPlaylistStore(dir)

	pl := storage.Playlist{
		Name: "作業用BGM",
		Videos: []storage.PlaylistVideo{
			{VideoID: "abc", Title: "テスト", Channel: "ch1"},
		},
	}

	if err := store.Save(pl); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load("作業用BGM")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(loaded.Videos))
	}
}

func TestListPlaylists(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewPlaylistStore(dir)

	store.Save(storage.Playlist{Name: "list1"})
	store.Save(storage.Playlist{Name: "list2"})

	names, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Errorf("expected 2 playlists, got %d", len(names))
	}
}

func TestAddVideoToPlaylist(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewPlaylistStore(dir)

	store.Save(storage.Playlist{Name: "mypl"})
	store.AddVideo("mypl", storage.PlaylistVideo{VideoID: "v1", Title: "title1"})

	pl, _ := store.Load("mypl")
	if len(pl.Videos) != 1 {
		t.Errorf("expected 1 video, got %d", len(pl.Videos))
	}
}
```

**Step 9: プレイリストの実装**

Create `internal/storage/playlist.go`:
```go
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PlaylistVideo struct {
	VideoID string `json:"video_id"`
	Title   string `json:"title"`
	Channel string `json:"channel"`
}

type Playlist struct {
	Name   string          `json:"name"`
	Videos []PlaylistVideo `json:"videos"`
}

type PlaylistStore struct {
	dir string
}

func NewPlaylistStore(dir string) *PlaylistStore {
	return &PlaylistStore{dir: dir}
}

func (s *PlaylistStore) filename(name string) string {
	safe := strings.ReplaceAll(name, "/", "_")
	return filepath.Join(s.dir, safe+".json")
}

func (s *PlaylistStore) Save(pl Playlist) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filename(pl.Name), data, 0o644)
}

func (s *PlaylistStore) Load(name string) (Playlist, error) {
	var pl Playlist
	data, err := os.ReadFile(s.filename(name))
	if err != nil {
		return pl, err
	}
	err = json.Unmarshal(data, &pl)
	return pl, err
}

func (s *PlaylistStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			name := strings.TrimSuffix(e.Name(), ".json")
			names = append(names, name)
		}
	}
	return names, nil
}

func (s *PlaylistStore) AddVideo(playlistName string, video PlaylistVideo) error {
	pl, err := s.Load(playlistName)
	if err != nil {
		return fmt.Errorf("playlist %q not found: %w", playlistName, err)
	}
	pl.Videos = append(pl.Videos, video)
	return s.Save(pl)
}
```

**Step 10: 全テスト通過確認**

```bash
go test ./internal/storage/ -v
```
Expected: ALL PASS

**Step 11: コミット**

```bash
git add internal/storage/
git commit -m "✨ feat: ストレージ層（設定・履歴・プレイリスト）を実装"
```

---

## Task 3: YouTube層 — yt-dlp経由の検索・メタデータ取得

**Files:**
- Create: `internal/youtube/types.go`
- Create: `internal/youtube/search.go`
- Create: `internal/youtube/search_test.go`
- Create: `internal/youtube/metadata.go`
- Create: `internal/youtube/metadata_test.go`

**Step 1: 共通型定義**

Create `internal/youtube/types.go`:
```go
package youtube

type Video struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Channel     string `json:"channel"`
	ChannelID   string `json:"channel_id"`
	Duration    int    `json:"duration"`
	DurationStr string `json:"duration_string"`
	ViewCount   int64  `json:"view_count"`
	URL         string `json:"webpage_url"`
	Thumbnail   string `json:"thumbnail"`
	IsLive      bool   `json:"is_live"`
}
```

**Step 2: 検索のテストを書く（yt-dlp JSONパース）**

Create `internal/youtube/search_test.go`:
```go
package youtube_test

import (
	"testing"

	"github.com/takeuchishougo/termtube/internal/youtube"
)

func TestParseSearchResults(t *testing.T) {
	jsonLines := `{"id":"abc123","title":"Test Video","channel":"TestCh","duration":120,"duration_string":"2:00","view_count":1000,"webpage_url":"https://www.youtube.com/watch?v=abc123","is_live":false}
{"id":"def456","title":"Another Video","channel":"OtherCh","duration":300,"duration_string":"5:00","view_count":5000,"webpage_url":"https://www.youtube.com/watch?v=def456","is_live":false}`

	videos, err := youtube.ParseSearchResults([]byte(jsonLines))
	if err != nil {
		t.Fatal(err)
	}
	if len(videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(videos))
	}
	if videos[0].ID != "abc123" {
		t.Errorf("expected id abc123, got %s", videos[0].ID)
	}
	if videos[1].Duration != 300 {
		t.Errorf("expected duration 300, got %d", videos[1].Duration)
	}
}
```

**Step 3: テスト失敗を確認**

```bash
go test ./internal/youtube/ -v
```
Expected: FAIL

**Step 4: 検索の実装**

Create `internal/youtube/search.go`:
```go
package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

func ParseSearchResults(data []byte) ([]Video, error) {
	var videos []Video
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var v Video
		if err := json.Unmarshal(line, &v); err != nil {
			continue
		}
		videos = append(videos, v)
	}
	return videos, nil
}

func Search(ctx context.Context, query string, maxResults int) ([]Video, error) {
	args := []string{
		fmt.Sprintf("ytsearch%d:%s", maxResults, query),
		"--dump-json",
		"--flat-playlist",
		"--no-download",
		"--no-warnings",
	}
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp search failed: %w", err)
	}
	return ParseSearchResults(output)
}
```

**Step 5: メタデータ取得のテストと実装**

Create `internal/youtube/metadata_test.go`:
```go
package youtube_test

import (
	"testing"

	"github.com/takeuchishougo/termtube/internal/youtube"
)

func TestParseMetadata(t *testing.T) {
	jsonData := `{"id":"abc123","title":"Test","channel":"Ch","duration":60,"duration_string":"1:00","webpage_url":"https://www.youtube.com/watch?v=abc123","is_live":false}`

	v, err := youtube.ParseMetadata([]byte(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != "abc123" {
		t.Errorf("expected id abc123, got %s", v.ID)
	}
}
```

Create `internal/youtube/metadata.go`:
```go
package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

func ParseMetadata(data []byte) (Video, error) {
	var v Video
	err := json.Unmarshal(data, &v)
	return v, err
}

func GetMetadata(ctx context.Context, videoURL string) (Video, error) {
	args := []string{videoURL, "--dump-json", "--no-download", "--no-warnings"}
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	output, err := cmd.Output()
	if err != nil {
		return Video{}, fmt.Errorf("yt-dlp metadata failed: %w", err)
	}
	return ParseMetadata(output)
}

func GetStreamURL(ctx context.Context, videoURL string) (string, error) {
	args := []string{videoURL, "--get-url", "--format", "best", "--no-warnings"}
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp get-url failed: %w", err)
	}
	return string(bytes.TrimSpace(output)), nil
}
```

**Step 6: テスト通過確認**

```bash
go test ./internal/youtube/ -v
```
Expected: PASS

**Step 7: コミット**

```bash
git add internal/youtube/
git commit -m "✨ feat: YouTube層（yt-dlp経由の検索・メタデータ取得）を実装"
```

---

## Task 4: mpv IPC制御

**Files:**
- Create: `internal/player/state.go`
- Create: `internal/player/mpv.go`
- Create: `internal/player/mpv_test.go`

**Step 1: 再生状態の型定義**

Create `internal/player/state.go`:
```go
package player

type State int

const (
	StateStopped State = iota
	StatePlaying
	StatePaused
)

type PlayerState struct {
	State    State
	Title    string
	Position float64
	Duration float64
	Volume   int
	VideoID  string
}
```

**Step 2: mpv IPCコマンド構造のテスト**

Create `internal/player/mpv_test.go`:
```go
package player_test

import (
	"encoding/json"
	"testing"

	"github.com/takeuchishougo/termtube/internal/player"
)

func TestMpvCommandJSON(t *testing.T) {
	cmd := player.MpvCommand{
		Command:   []interface{}{"set_property", "pause", true},
		RequestID: 1,
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"command":["set_property","pause",true],"request_id":1}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestParseMpvEvent(t *testing.T) {
	eventJSON := `{"event":"playback-restart"}`
	var event player.MpvEvent
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		t.Fatal(err)
	}
	if event.Event != "playback-restart" {
		t.Errorf("expected playback-restart, got %s", event.Event)
	}
}
```

**Step 3: mpv IPC実装**

Create `internal/player/mpv.go`:
```go
package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

type MpvCommand struct {
	Command   []interface{} `json:"command"`
	RequestID int           `json:"request_id"`
}

type MpvResponse struct {
	Error     string      `json:"error"`
	Data      interface{} `json:"data"`
	RequestID int         `json:"request_id"`
}

type MpvEvent struct {
	Event string `json:"event"`
}

type MpvPlayer struct {
	socketPath    string
	cmd           *exec.Cmd
	conn          net.Conn
	mu            sync.Mutex
	state         PlayerState
	requestID     int
	onStateChange func(PlayerState)
}

func NewMpvPlayer() *MpvPlayer {
	return &MpvPlayer{
		socketPath: fmt.Sprintf("/tmp/termtube-mpv-%d.sock", os.Getpid()),
		state:      PlayerState{State: StateStopped, Volume: 80},
	}
}

func (m *MpvPlayer) Play(url string) error {
	m.Stop()

	args := []string{
		"--vo=sixel",
		"--no-terminal",
		"--input-ipc-server=" + m.socketPath,
		fmt.Sprintf("--volume=%d", m.state.Volume),
		url,
	}
	m.cmd = exec.Command("mpv", args...)
	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("mpv start failed: %w", err)
	}

	for i := 0; i < 50; i++ {
		conn, err := net.Dial("unix", m.socketPath)
		if err == nil {
			m.conn = conn
			m.state.State = StatePlaying
			go m.listenEvents()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("failed to connect to mpv socket")
}

func (m *MpvPlayer) sendCommand(cmd MpvCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn == nil {
		return fmt.Errorf("not connected to mpv")
	}

	m.requestID++
	cmd.RequestID = m.requestID

	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = m.conn.Write(data)
	return err
}

func (m *MpvPlayer) TogglePause() error {
	return m.sendCommand(MpvCommand{Command: []interface{}{"cycle", "pause"}})
}

func (m *MpvPlayer) Seek(seconds float64) error {
	return m.sendCommand(MpvCommand{Command: []interface{}{"seek", seconds, "relative"}})
}

func (m *MpvPlayer) SetVolume(vol int) error {
	m.state.Volume = vol
	return m.sendCommand(MpvCommand{Command: []interface{}{"set_property", "volume", vol}})
}

func (m *MpvPlayer) GetState() PlayerState {
	return m.state
}

func (m *MpvPlayer) Stop() {
	if m.conn != nil {
		m.sendCommand(MpvCommand{Command: []interface{}{"quit"}})
		m.conn.Close()
		m.conn = nil
	}
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
		m.cmd = nil
	}
	os.Remove(m.socketPath)
	m.state.State = StateStopped
}

func (m *MpvPlayer) OnStateChange(fn func(PlayerState)) {
	m.onStateChange = fn
}

func (m *MpvPlayer) listenEvents() {
	scanner := bufio.NewScanner(m.conn)
	for scanner.Scan() {
		var event MpvEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err == nil && event.Event != "" {
			m.handleEvent(event)
		}
	}
}

func (m *MpvPlayer) handleEvent(event MpvEvent) {
	switch event.Event {
	case "pause":
		m.state.State = StatePaused
	case "unpause":
		m.state.State = StatePlaying
	case "end-file":
		m.state.State = StateStopped
	}
	if m.onStateChange != nil {
		m.onStateChange(m.state)
	}
}
```

**Step 4: テスト通過確認**

```bash
go test ./internal/player/ -v
```
Expected: PASS

**Step 5: コミット**

```bash
git add internal/player/
git commit -m "✨ feat: mpv IPC制御層を実装"
```

---

## Task 5: 検索UI画面

**Files:**
- Create: `internal/ui/styles.go`
- Create: `internal/ui/search.go`
- Modify: `internal/app/app.go`

**Step 1: 共通スタイル定義**

Create `internal/ui/styles.go`:
```go
package ui

import "github.com/charmbracelet/lipgloss"

var (
	ActiveTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Padding(0, 1)
	InactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1)
	TitleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	SubtitleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	HelpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
	BorderStyle      = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1)
)
```

**Step 2: 検索画面のBubble Teaモデル**

Create `internal/ui/search.go`:
```go
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

type SearchResultMsg struct {
	Videos []youtube.Video
	Err    error
}

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

type SearchModel struct {
	textInput  textinput.Model
	resultList list.Model
	searching  bool
	inputMode  bool
	width      int
	height     int
}

func NewSearchModel() SearchModel {
	ti := textinput.New()
	ti.Placeholder = "YouTube検索..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 60

	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "検索結果"
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return SearchModel{textInput: ti, resultList: l, inputMode: true}
}

func (m SearchModel) Init() tea.Cmd { return textinput.Blink }

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resultList.SetSize(msg.Width-4, msg.Height-8)
	case SearchResultMsg:
		m.searching = false
		if msg.Err == nil {
			items := make([]list.Item, len(msg.Videos))
			for i, v := range msg.Videos {
				items[i] = videoItem{video: v}
			}
			m.resultList.SetItems(items)
			m.inputMode = false
		}
	case tea.KeyMsg:
		if m.inputMode {
			switch msg.String() {
			case "enter":
				if q := m.textInput.Value(); q != "" {
					m.searching = true
					return m, doSearch(q)
				}
			case "esc":
				if len(m.resultList.Items()) > 0 {
					m.inputMode = false
				}
			}
		} else {
			if msg.String() == "/" {
				m.inputMode = true
				m.textInput.Focus()
				return m, textinput.Blink
			}
		}
	}

	if m.inputMode {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.resultList, cmd = m.resultList.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m SearchModel) View() string {
	searchBar := m.textInput.View()
	if m.searching {
		searchBar += "  検索中..."
	}
	header := BorderStyle.Width(m.width - 4).Render(searchBar)

	var content string
	if len(m.resultList.Items()) > 0 {
		content = m.resultList.View()
	} else if !m.searching {
		content = SubtitleStyle.Render("\n  /で検索を開始")
	}

	help := HelpStyle.Render("[/] 検索  [Enter] 再生  [a] PLに追加  [q] 終了")
	return lipgloss.JoinVertical(lipgloss.Left, header, content, help)
}

func (m SearchModel) SelectedVideo() *youtube.Video {
	item, ok := m.resultList.SelectedItem().(videoItem)
	if !ok {
		return nil
	}
	return &item.video
}

func doSearch(query string) tea.Cmd {
	return func() tea.Msg {
		videos, err := youtube.Search(context.Background(), query, 20)
		return SearchResultMsg{Videos: videos, Err: err}
	}
}
```

**Step 3: app.goを検索画面と統合**

`internal/app/app.go` のModelにSearchModelを組み込み、ViewSearchのときに検索画面をレンダリング。検索画面のUpdate/Viewを委譲。

**Step 4: ビルド&動作確認**

```bash
go mod tidy && go run .
```
Expected: 検索バーが表示、キーワード入力→Enter→検索結果がリスト表示。

**Step 5: コミット**

```bash
git add internal/ui/ internal/app/
git commit -m "✨ feat: YouTube検索TUI画面を実装"
```

---

## Task 6: 再生UI画面 + mpv統合

**Files:**
- Create: `internal/ui/player.go`
- Modify: `internal/app/app.go`

**Step 1: 再生画面モデル**

Create `internal/ui/player.go` — mpvの状態表示、タイトル/チャンネル名、再生位置バー、音量表示、キーバインドヘルプ。

**Step 2: app.goに再生画面を統合**

検索結果でEnter→SelectedVideo取得→PlayerModel.PlayVideo呼出→ViewPlayerに切替。

**Step 3: ビルド&動作確認**

```bash
go run .
```
Expected: 検索→動画選択→mpvがSixelで映像再生、操作パネル表示。

**Step 4: コミット**

```bash
git add internal/ui/player.go internal/app/app.go
git commit -m "✨ feat: 再生UI画面とmpv統合を実装"
```

---

## Task 7: 履歴UI画面

**Files:**
- Create: `internal/ui/history.go`
- Modify: `internal/app/app.go`

**Step 1: 履歴画面モデル**

Bubbles listでHistoryStoreから読み込んだ履歴を表示。Enterで再生再開。

**Step 2: app.goに統合**

動画再生時に自動で履歴追加。ViewHistoryのとき履歴画面表示。

**Step 3: ビルド&動作確認**

```bash
go run .
```
Expected: 動画視聴後に履歴タブで視聴履歴が表示。

**Step 4: コミット**

```bash
git add internal/ui/history.go internal/app/app.go
git commit -m "✨ feat: 履歴UI画面を実装"
```

---

## Task 8: プレイリストUI画面

**Files:**
- Create: `internal/ui/playlist.go`
- Modify: `internal/app/app.go`

**Step 1: プレイリスト画面モデル**

PL一覧→動画一覧の2階層ナビゲーション。`a`キーで検索結果からPLに追加。新規PL作成対応。

**Step 2: app.goに統合**

検索画面で`a`→PL選択→追加。ViewPlaylistのときPL画面表示。

**Step 3: ビルド&動作確認**

```bash
go run .
```
Expected: プレイリストの作成・追加・再生が機能。

**Step 4: コミット**

```bash
git add internal/ui/playlist.go internal/app/app.go
git commit -m "✨ feat: プレイリストUI画面を実装"
```

---

## Task 9: ライブチャット表示

**Files:**
- Create: `internal/youtube/chat.go`
- Create: `internal/youtube/chat_test.go`
- Create: `internal/ui/chat.go`
- Modify: `internal/ui/player.go`

**Step 1: チャットメッセージのパーサーテスト**

Create `internal/youtube/chat_test.go`:
```go
package youtube_test

import (
	"testing"

	"github.com/takeuchishougo/termtube/internal/youtube"
)

func TestParseChatMessage(t *testing.T) {
	jsonData := `{"author":"user1","message":"こんにちは","timestamp":1708300000}`
	msg, err := youtube.ParseChatMessage([]byte(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Author != "user1" {
		t.Errorf("expected author user1, got %s", msg.Author)
	}
}
```

**Step 2: チャット取得の実装**

Create `internal/youtube/chat.go` — yt-dlpのライブチャット取得 or chat_downloaderを子プロセス起動。goroutineでストリーミング取得し、channelでUIに送信。

**Step 3: チャットUI表示**

Create `internal/ui/chat.go` — 再生画面の下部にチャットをスクロール表示。`c`キーで表示/非表示切替。

**Step 4: テスト&ビルド確認**

```bash
go test ./internal/youtube/ -v && go run .
```

**Step 5: コミット**

```bash
git add internal/youtube/chat.go internal/youtube/chat_test.go internal/ui/chat.go internal/ui/player.go
git commit -m "✨ feat: ライブチャット表示を実装"
```

---

## Task 10: 関連動画表示

**Files:**
- Modify: `internal/youtube/search.go`
- Modify: `internal/ui/player.go`

**Step 1: 関連動画取得関数を追加**

`internal/youtube/search.go` に `GetRelated(ctx, videoID)` を追加。

**Step 2: 再生画面でTab→関連動画パネル**

Enterで選択して次の動画を再生。

**Step 3: ビルド&動作確認**

```bash
go run .
```

**Step 4: コミット**

```bash
git add internal/youtube/search.go internal/ui/player.go
git commit -m "✨ feat: 関連動画表示を実装"
```

---

## Task 11: BGV/集中視聴モード切替

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/ui/player.go`

**Step 1: モード切替ロジック**

`b`キーでBGV↔集中モード切替。WezTermの`wezterm cli split-pane`でペイン分割を制御。

**Step 2: ビルド&動作確認**

```bash
go run .
```
Expected: `b`キーでレイアウトが切り替わる。

**Step 3: コミット**

```bash
git add internal/app/app.go internal/ui/player.go
git commit -m "✨ feat: BGV/集中視聴モード切替を実装"
```

---

## Task 12: CLIインターフェース + 外部依存チェック

**Files:**
- Create: `cmd/root.go`
- Create: `cmd/play.go`
- Modify: `main.go`

**Step 1: cobra CLIセットアップ**

Create `cmd/root.go`:
```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "termtube",
	Short: "WezTermで動くYouTube TUIプレイヤー",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return checkDependencies()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func checkDependencies() error {
	for _, dep := range []string{"mpv", "yt-dlp"} {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("%s が見つかりません。brew install %s でインストールしてください", dep, dep)
		}
	}
	return nil
}
```

Create `cmd/play.go` — `termtube play <URL>` で直接再生。

**Step 2: main.goを更新**

```go
package main

import "github.com/takeuchishougo/termtube/cmd"

func main() {
	cmd.Execute()
}
```

**Step 3: ビルド&動作確認**

```bash
go build -o termtube . && ./termtube --help
```

**Step 4: コミット**

```bash
git add cmd/ main.go
git commit -m "✨ feat: cobra CLIインターフェースと依存チェックを実装"
```

---

## Task 13: 最終統合テスト

**Step 1: 全ユニットテスト**

```bash
go test ./... -v
```
Expected: ALL PASS

**Step 2: 手動統合チェック**

- [ ] `./termtube` → TUI起動、検索、再生
- [ ] `./termtube play <URL>` → 直接再生
- [ ] プレイリスト作成・追加・再生
- [ ] 履歴の保存・表示・再生
- [ ] ライブ配信のチャット表示
- [ ] 関連動画の表示・再生
- [ ] BGV/集中モード切替

**Step 3: コミット**

```bash
git add .
git commit -m "🔧 chore: 最終統合テスト完了"
```
