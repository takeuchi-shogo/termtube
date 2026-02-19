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
