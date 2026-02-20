package cmd

import "github.com/spf13/cobra"

var playCmd = &cobra.Command{
	Use:   "play [URL]",
	Short: "URLを指定して直接再生",
	Long:  `指定された YouTube URL の動画を直接再生します。検索画面をスキップしてプレイヤー画面から開始します。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithURL(args[0])
	},
}

func init() {
	rootCmd.AddCommand(playCmd)
}
