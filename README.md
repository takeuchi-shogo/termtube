# termtube

ターミナルで YouTube を視聴する TUI アプリケーション。
mpv の Sixel 出力と yt-dlp を組み合わせ、ターミナル上で動画の検索・再生・プレイリスト管理を完結させます。

## 必要なもの

| ツール | 用途 | インストール |
|--------|------|-------------|
| **Go** 1.25+ | ビルド | [golang.org](https://golang.org/dl/) |
| **mpv** | 動画再生（Sixel出力） | `brew install mpv` |
| **yt-dlp** | YouTube検索・メタデータ取得 | `brew install yt-dlp` |

Sixel 対応ターミナルが必要です（WezTerm, iTerm2, kitty 等）。

## セットアップ

[Task](https://taskfile.dev/) を使えばワンコマンドで依存インストールからビルドまで完了します。

```bash
git clone https://github.com/takeuchi-shogo/termtube.git
cd termtube
task setup
```

### Task コマンド一覧

| コマンド | 内容 |
|---------|------|
| `task setup` | 依存ツール (mpv, yt-dlp) のインストール + ビルド |
| `task run` | ビルドして起動 |
| `task build` | バイナリをビルド |
| `task test` | テスト実行 |
| `task install` | `/usr/local/bin` にインストール |
| `task uninstall` | インストールしたバイナリを削除 |
| `task clean` | ビルド成果物を削除 |
| `task deps` | 依存ツールのみインストール |

### 手動インストール

Task を使わない場合:

```bash
brew install mpv yt-dlp
go build -o termtube .
```

または `go install`:

```bash
go install github.com/takeuchishougo/termtube@latest
```

## 使い方

```bash
# TUI を起動（検索画面から開始）
termtube

# URL を指定して直接再生
termtube play https://www.youtube.com/watch?v=xxxxx
```

## 画面構成

数字キーで4つのタブを切り替えます。

| キー | タブ | 内容 |
|------|------|------|
| `1` | 検索 | YouTube を検索して結果一覧から再生 |
| `2` | 再生 | 動画の再生操作・情報表示 |
| `3` | PL | プレイリストの管理・再生 |
| `4` | 履歴 | 視聴履歴から再生 |

## キーバインド

### グローバル

| キー | 操作 |
|------|------|
| `1` `2` `3` `4` | タブ切り替え |
| `q` / `Ctrl+C` | 終了 |

### 検索画面

| キー | 操作 |
|------|------|
| `/` | 検索入力モードに入る |
| `Enter` | 検索実行（入力モード時）/ 動画再生（一覧モード時） |
| `Esc` | 入力モードを抜ける |
| `j` / `k` | カーソル移動 |
| `a` | 選択中の動画を「お気に入り」に追加 |

### 再生画面

| キー | 操作 |
|------|------|
| `Space` | 再生 / 一時停止 |
| `←` / `→` | シーク ±5秒 |
| `↑` / `↓` | 音量 ±5 |
| `c` | ライブチャット表示切替 |
| `Tab` | 関連動画パネル表示切替 |
| `b` | Focus / BGV モード切替 |

関連動画パネル表示中:

| キー | 操作 |
|------|------|
| `j` / `k` | 関連動画を選択 |
| `Enter` | 選択した動画を再生 |

### プレイリスト画面

| キー | 操作 |
|------|------|
| `j` / `k` | 選択移動 |
| `Enter` | プレイリストを開く / 動画を再生 |
| `d` | 動画を削除（動画一覧時） |
| `Esc` / `Backspace` | プレイリスト一覧に戻る |

### 履歴画面

| キー | 操作 |
|------|------|
| `j` / `k` | 選択移動 |
| `Enter` | 動画を再生 |

## 表示モード

### Focus モード
動画タイトル・チャンネル名・再生位置・音量バーを大きく表示する集中視聴向けレイアウト。

### BGV モード
1行のコンパクト表示。コーディング中のBGMなど、ターミナルのスペースを確保したい場合に。

`b` キーで切り替え可能。デフォルトは設定ファイルで変更できます。

## 設定

`~/.local/share/termtube/config.toml` に保存されます（初回起動時に自動作成）。

```toml
[player]
default_mode = "focus"    # "focus" or "bgv"
default_volume = 80       # 0-100
sixel_quality = "medium"

[chat]
enabled = true
max_lines = 50

[search]
results_per_page = 20
region = "JP"
```

## データ保存先

| 種類 | パス |
|------|------|
| 設定 | `~/.local/share/termtube/config.toml` |
| 履歴 | `~/.local/share/termtube/history.json` |
| プレイリスト | `~/.local/share/termtube/playlists/*.json` |

履歴は最新100件を保持。プレイリストはプレイリストごとに個別の JSON ファイルとして保存されます。

## アーキテクチャ

```
termtube (Go + Bubble Tea)
├── TUI Layer ─── Bubble Tea + Lipgloss
│   ├── Search   ── textinput + list
│   ├── Player   ── Focus / BGV mode
│   ├── Playlist ── 2-level navigation
│   ├── History  ── list
│   └── Chat     ── scrolling messages
│
├── Player Layer ── mpv IPC (UNIX socket JSON RPC)
│   └── mpv --vo=sixel --input-ipc-server=<socket>
│
├── YouTube Layer ── yt-dlp subprocess
│   ├── Search:   ytsearchN:<query>
│   ├── Metadata: URL → video details
│   ├── Related:  title → keyword search
│   └── Chat:     live_chat subtitle stream
│
└── Storage Layer ── JSON / TOML files
    ├── Config    (TOML, atomic write)
    ├── History   (JSON, atomic write, mutex)
    └── Playlists (JSON, atomic write, mutex)
```

## ライセンス

MIT
