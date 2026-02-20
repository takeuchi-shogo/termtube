package youtube

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ParseSearchResults parses newline-delimited JSON output from yt-dlp search.
// Each line is expected to be a separate JSON object representing a video.
func ParseSearchResults(data []byte) ([]Video, error) {
	var videos []Video

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var v Video
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			return nil, fmt.Errorf("failed to parse video JSON: %w", err)
		}
		videos = append(videos, v)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan search results: %w", err)
	}

	return videos, nil
}

// Search runs yt-dlp to search YouTube for videos matching the query.
// maxResults controls how many results to return (yt-dlp ytsearchN: syntax).
func Search(ctx context.Context, query string, maxResults int) ([]Video, error) {
	searchQuery := fmt.Sprintf("ytsearch%d:%s", maxResults, query)

	cmd := exec.CommandContext(ctx, "yt-dlp",
		searchQuery,
		"--dump-json",
		"--flat-playlist",
		"--no-download",
		"--no-warnings",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp search failed: %w", err)
	}

	return ParseSearchResults(output)
}

// GetRelated は指定された動画に関連する動画を取得する。
// 動画タイトルの先頭キーワードを使って検索することで、
// 関連動画に近い結果を返す。
func GetRelated(ctx context.Context, videoTitle string, maxResults int) ([]Video, error) {
	// タイトルから検索キーワードを抽出（先頭の数単語を使用）
	keywords := extractKeywords(videoTitle)
	if keywords == "" {
		return nil, fmt.Errorf("no keywords extracted from video title")
	}

	return Search(ctx, keywords, maxResults)
}

// extractKeywords はタイトルからスペースで区切った先頭5単語を返す。
// 短すぎる単語や記号を除外する。
func extractKeywords(title string) string {
	words := strings.Fields(title)
	var keywords []string
	for _, w := range words {
		// 1文字以下の単語や記号のみの単語をスキップ
		cleaned := strings.TrimFunc(w, func(r rune) bool {
			return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r >= 0x3000) // 日本語文字も含む
		})
		if len(cleaned) > 1 || len([]rune(cleaned)) > 0 {
			keywords = append(keywords, cleaned)
		}
		if len(keywords) >= 5 {
			break
		}
	}
	return strings.Join(keywords, " ")
}
