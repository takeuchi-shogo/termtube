package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ParseMetadata parses a single JSON object representing video metadata from yt-dlp.
func ParseMetadata(data []byte) (Video, error) {
	data = bytes.TrimSpace(data)

	var v Video
	if err := json.Unmarshal(data, &v); err != nil {
		return Video{}, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	return v, nil
}

// GetMetadata runs yt-dlp to retrieve full metadata for a given video URL.
func GetMetadata(ctx context.Context, videoURL string) (Video, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--dump-json",
		"--no-download",
		"--no-warnings",
		"--", videoURL,
	)

	output, err := cmd.Output()
	if err != nil {
		return Video{}, fmt.Errorf("yt-dlp metadata failed: %w", err)
	}

	return ParseMetadata(output)
}

// GetStreamURL runs yt-dlp to retrieve the direct stream URL for a given video URL.
func GetStreamURL(ctx context.Context, videoURL string) (string, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--get-url",
		"--format", "best",
		"--no-warnings",
		"--", videoURL,
	)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp get-url failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
