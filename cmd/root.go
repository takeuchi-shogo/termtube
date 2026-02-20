package cmd

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/takeuchishougo/termtube/internal/app"
)

var rootCmd = &cobra.Command{
	Use:   "termtube",
	Short: "WezTermで動くYouTube TUIプレイヤー",
	Long: `termtube は WezTerm 上で動作する YouTube TUI プレイヤーです。
yt-dlp で動画を検索し、mpv (sixel 出力) で再生します。`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return checkDependencies()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

// Execute はルートコマンドを実行する。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// checkDependencies は外部依存 (mpv, yt-dlp) がインストールされているか確認する。
func checkDependencies() error {
	deps := []string{"mpv", "yt-dlp"}
	for _, dep := range deps {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("%s が見つかりません。brew install %s でインストールしてください", dep, dep)
		}
	}
	return nil
}

// runTUI は通常の TUI モードを起動する。
func runTUI() error {
	p := tea.NewProgram(app.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI の起動に失敗しました: %w", err)
	}
	return nil
}

// runWithURL は指定された URL を即時再生する TUI モードを起動する。
func runWithURL(url string) error {
	p := tea.NewProgram(app.NewWithURL(url), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI の起動に失敗しました: %w", err)
	}
	return nil
}
