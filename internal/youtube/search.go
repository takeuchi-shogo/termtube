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
