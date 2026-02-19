# termtube - WezTerm内YouTube TUIプレイヤー 設計書

## 概要

WezTermターミナル内で完結するYouTube動画プレイヤー。ウィンドウ切り替えなしで検索・再生・チャット表示ができる。

## 外部依存

- `mpv` — Sixelモードで映像再生（子プロセス、IPC制御）
- `yt-dlp` — YouTube検索・動画URL解決・メタデータ取得

## アーキテクチャ

```
termtube (Go / Bubble Tea)
  ├── TUI Layer (Ratatui → Bubble Tea + Lip Gloss + Bubbles)
  │   ├── 検索画面
  │   ├── 再生画面（操作パネル）
  │   ├── プレイリスト画面
  │   ├── 履歴画面
  │   └── チャット表示
  ├── Player Control (mpv IPC over UNIX socket)
  │   ├── 再生/一時停止/シーク/音量
  │   └── 状態監視（再生位置、再生状態）
  ├── YouTube Layer (yt-dlp subprocess)
  │   ├── キーワード検索
  │   ├── 動画URL解決
  │   ├── メタデータ取得
  │   ├── 関連動画取得
  │   └── ライブチャット取得
  └── Storage Layer (JSON/TOML files)
      ├── 設定 (~/.config/termtube/config.toml)
      ├── 履歴 (~/.local/share/termtube/history.json)
      └── プレイリスト (~/.local/share/termtube/playlists/)
```

## 画面モード

### BGVモード
- ターミナル下部に小さく映像表示
- 上部にTUI操作パネル（検索/プレイリスト/履歴）
- コーディング中のBGV視聴を想定

### 集中視聴モード
- 映像を大きく表示
- 操作は最小限のオーバーレイ
- チャット表示対応（ライブ配信時）

## 画面構成

### メイン再生画面
```
┌─ termtube ──────────────────────────────────────────┐
│  [検索] [プレイリスト] [履歴] [設定]                   │
├─────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────┐    │
│  │           映像エリア (mpv sixel)              │    │
│  └─────────────────────────────────────────────┘    │
│  ┌──── チャット / 関連動画 ────────────────────┐     │
│  │  user1: コメント                            │     │
│  │  user2: コメント                            │     │
│  └────────────────────────────────────────────┘     │
│  ▶ タイトル名         03:24 / 10:00                 │
│  [再生] [次] [音量] [BGV/集中]                       │
└─────────────────────────────────────────────────────┘
```

### 検索画面
```
┌─ termtube > 検索 ──────────────────────────────────┐
│  [検索キーワードを入力...                      ]     │
├─────────────────────────────────────────────────────┤
│  1. 動画タイトル        チャンネル名     10:24       │
│  2. 動画タイトル        チャンネル名     45:12       │
├─────────────────────────────────────────────────────┤
│  [Enter] 再生  [a] PLに追加  [Tab] 関連動画         │
└─────────────────────────────────────────────────────┘
```

## キーバインド

| キー | アクション |
|------|-----------|
| `/` | 検索 |
| `Space` | 再生/一時停止 |
| `←` `→` | 5秒シーク |
| `↑` `↓` | 音量 |
| `n` | 次の動画 |
| `p` | 前の動画 |
| `b` | BGV/集中モード切替 |
| `c` | チャット表示切替 |
| `q` | 終了 |
| `Tab` | パネル切替 |

## データ構造

### 設定 (~/.config/termtube/config.toml)
```toml
[player]
default_mode = "focus"
default_volume = 80
sixel_quality = "medium"

[chat]
enabled = true
max_lines = 50

[search]
results_per_page = 20
region = "JP"
```

### 履歴 (~/.local/share/termtube/history.json)
```json
[
  {
    "video_id": "xxxxx",
    "title": "動画タイトル",
    "channel": "チャンネル名",
    "watched_at": "2026-02-18T10:30:00Z",
    "duration": 624,
    "position": 300
  }
]
```

### プレイリスト (~/.local/share/termtube/playlists/*.json)
```json
{
  "name": "作業用BGM",
  "videos": [
    { "video_id": "xxxxx", "title": "...", "channel": "..." }
  ]
}
```

## 技術スタック (Go)

| 用途 | パッケージ |
|------|-----------|
| TUI | bubbletea |
| スタイル | lipgloss |
| UIコンポーネント | bubbles |
| CLI引数 | cobra |
| 設定 | viper |
| JSON | encoding/json (標準) |
| ログ | charmbracelet/log |

## モジュール構成

```
termtube/
├── main.go
├── cmd/
│   └── root.go
├── internal/
│   ├── app/
│   │   └── app.go
│   ├── ui/
│   │   ├── search.go
│   │   ├── player.go
│   │   ├── playlist.go
│   │   ├── history.go
│   │   ├── chat.go
│   │   └── styles.go
│   ├── player/
│   │   ├── mpv.go
│   │   └── state.go
│   ├── youtube/
│   │   ├── search.go
│   │   ├── metadata.go
│   │   └── chat.go
│   └── storage/
│       ├── config.go
│       ├── history.go
│       └── playlist.go
├── go.mod
└── go.sum
```

## 映像再生方式

mpvを `--vo=sixel` モードで子プロセスとして起動し、UNIX socketベースのJSON IPCで制御する。

### mpv起動コマンド例
```bash
mpv --vo=sixel --no-terminal --input-ipc-server=/tmp/termtube-mpv.sock "VIDEO_URL"
```

### TUIとmpvの画面共存
WezTermのペイン分割（`wezterm cli split-pane`）を活用し、上ペインでmpv映像、下ペインでRatatui TUIを表示する方式を基本とする。

## ライブチャット

yt-dlpの `--write-chat` オプション、もしくは `chat_downloader` (Python) を子プロセスで呼び出してライブチャットを取得。チャットメッセージはgoroutineでストリーミング受信し、UIに反映する。
