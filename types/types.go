package types

import "time"

type Dimension string

const (
	Likes     Dimension = "likes"
	Comments  Dimension = "comments"
	Favorites Dimension = "favorites"
	Retweets  Dimension = "retweets"
)

type AnalysisRequestParams struct {
	Duration  time.Duration  `json:"duration"`
	Dimension Dimension `json:"dimension"`
}

type APIResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type ResponseData struct {
	Duration  string    `json:"duration"`
	Dimension Dimension `json:"dimension"`
}

type EventsAnalysisResult struct {
	TotalPosts       int   `json:"total_posts"`
	MinimumTimestamp int64 `json:"minimum_timestamp"`
	MaximumTimestamp int64 `json:"maximum_timestamp"`
	DimensionP50     int   `json:"dimension_p50"`
	DimensionP90     int   `json:"dimension_p90"`
	DimensionP99     int   `json:"dimension_p99"`
}
