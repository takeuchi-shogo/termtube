package youtube

import (
	"fmt"
	"net/url"
	"strings"
)

// Video は YouTube 動画のメタデータを表す。
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

// WatchURL は videoID から YouTube の視聴 URL を生成する。
func WatchURL(videoID string) string {
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", url.QueryEscape(videoID))
}

// IsValidYouTubeURL は URL が YouTube の URL かどうかを検証する。
func IsValidYouTubeURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "www.youtube.com" || host == "youtube.com" ||
		host == "youtu.be" || host == "m.youtube.com" ||
		host == "music.youtube.com"
}
