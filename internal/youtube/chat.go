package youtube

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ChatMessage はライブチャットの1メッセージを表す。
type ChatMessage struct {
	Author    string `json:"author"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// ParseChatMessage は単一の JSON チャットメッセージをパースする。
func ParseChatMessage(data []byte) (ChatMessage, error) {
	var msg ChatMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return ChatMessage{}, fmt.Errorf("failed to parse chat message: %w", err)
	}
	return msg, nil
}

// StreamChat はライブチャットのストリームを開始し、メッセージをチャネルに送信する。
// yt-dlp の live_chat サブタイトル抽出を利用する。
// プロセスは ctx がキャンセルされると停止する。
func StreamChat(ctx context.Context, videoURL string, ch chan<- ChatMessage) {
	defer close(ch)

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--write-sub",
		"--sub-lang", "live_chat",
		"--skip-download",
		"--dump-json",
		"--no-warnings",
		"--", videoURL,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	if err := cmd.Start(); err != nil {
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		msg, err := ParseChatMessage(line)
		if err != nil {
			continue
		}

		select {
		case ch <- msg:
		case <-ctx.Done():
			return
		}
	}

	_ = cmd.Wait()
}
