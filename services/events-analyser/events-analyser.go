package eventsanalyser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/AlexisHutin/stream-aggregation-service/services/sse"
	"github.com/AlexisHutin/stream-aggregation-service/types"
)

// EventsAnalyser orchestrates event collection from SSE and computes analysis
// results for requested dimensions and windows.
type EventsAnalyser struct {
	sseClient *sse.Client
}

// NewEventsAnalyser builds an analyser configured to consume events from the
// given SSE stream URL.
func NewEventsAnalyser(sseStreamURL string) *EventsAnalyser {
	return &EventsAnalyser{
		sseClient: sse.NewClient(sseStreamURL),
	}
}

// EventsAnalysis collects events for the provided duration and returns an
// aggregated analysis result for the selected dimension.
func (ea *EventsAnalyser) EventsAnalysis(
	ctx context.Context,
	dimension types.Dimension,
	duration time.Duration,
) (types.EventsAnalysisResult, error) {
	streamReader, streamCloser, err := ea.sseClient.OpenStreamReader(ctx)
	if err != nil {
		return types.EventsAnalysisResult{}, fmt.Errorf("failed to open SSE stream: %w", err)
	}
	defer streamCloser.Close()

	scanner := ea.sseClient.NewScanner(streamReader)
	rawEventPayloads, err := collectRawEventPayloads(scanner, streamCloser, duration)
	if err != nil {
		return types.EventsAnalysisResult{}, err
	}

	dimensionSamples, err := buildDimensionSamples(rawEventPayloads, dimension)
	if err != nil {
		return types.EventsAnalysisResult{}, err
	}

	analysisStats := computeAnalysisStats(dimensionSamples)

	return types.EventsAnalysisResult{
		TotalPosts:       analysisStats.TotalPosts,
		MinimumTimestamp: analysisStats.MinimumTimestamp,
		MaximumTimestamp: analysisStats.MaximumTimestamp,
		DimensionP50:     analysisStats.P50,
		DimensionP90:     analysisStats.P90,
		DimensionP99:     analysisStats.P99,
	}, nil
}

// socialPost holds the subset of fields needed for dimension aggregation.
type socialPost struct {
	Timestamp int64 `json:"timestamp"`
	Likes     int   `json:"likes"`
	Comments  int   `json:"comments"`
	Favorites int   `json:"favorites"`
	Retweets  int   `json:"retweets"`
}

// dimensionSample stores the timestamp and selected metric value used for
// downstream aggregation.
type dimensionSample struct {
	Timestamp int64
	Metric    int
}

// computedStats contains aggregate values derived from projected samples.
type computedStats struct {
	TotalPosts       int
	MinimumTimestamp int64
	MaximumTimestamp int64
	P50              int
	P90              int
	P99              int
}

// buildDimensionSamples parses raw SSE payloads and projects each event to the
// requested analysis dimension.
func buildDimensionSamples(rawPayloads []string, dimension types.Dimension) ([]dimensionSample, error) {
	samples := make([]dimensionSample, 0, len(rawPayloads))

	for index, payload := range rawPayloads {
		post, err := decodeSocialPost(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to decode payload at index %d: %w", index, err)
		}

		dimensionMetric, err := extractDimensionMetric(post, dimension)
		if err != nil {
			return nil, fmt.Errorf("failed to project payload at index %d: %w", index, err)
		}

		samples = append(samples, dimensionSample{
			Timestamp: post.Timestamp,
			Metric:    dimensionMetric,
		})
	}

	return samples, nil
}

// decodeSocialPost decodes a payload with one dynamic root key
// (for example instagram_media, tweet, article) into a socialPost.
func decodeSocialPost(payload string) (socialPost, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		return socialPost{}, err
	}

	if len(envelope) == 0 {
		return socialPost{}, errors.New("empty payload object")
	}

	for _, raw := range envelope {
		var post socialPost
		if err := json.Unmarshal(raw, &post); err != nil {
			return socialPost{}, err
		}
		return post, nil
	}

	return socialPost{}, errors.New("missing social post object")
}

// extractDimensionMetric returns the metric value of an event for the selected
// analysis dimension.
func extractDimensionMetric(event socialPost, dimension types.Dimension) (int, error) {
	switch dimension {
	case types.Likes:
		return event.Likes, nil
	case types.Comments:
		return event.Comments, nil
	case types.Favorites:
		return event.Favorites, nil
	case types.Retweets:
		return event.Retweets, nil
	default:
		return 0, fmt.Errorf("unsupported dimension: %s", dimension)
	}
}

// computeAnalysisStats calculates global counters and percentile values for the
// projected dimension samples.
func computeAnalysisStats(samples []dimensionSample) computedStats {
	stats := computedStats{TotalPosts: len(samples)}
	if len(samples) == 0 {
		return stats
	}

	stats.MinimumTimestamp = samples[0].Timestamp
	stats.MaximumTimestamp = samples[0].Timestamp

	metricValues := make([]int, 0, len(samples))
	for _, sample := range samples {
		metricValues = append(metricValues, sample.Metric)
		if sample.Timestamp < stats.MinimumTimestamp {
			stats.MinimumTimestamp = sample.Timestamp
		}
		if sample.Timestamp > stats.MaximumTimestamp {
			stats.MaximumTimestamp = sample.Timestamp
		}
	}

	slices.Sort(metricValues)
	stats.P50 = percentile(metricValues, 0.50)
	stats.P90 = percentile(metricValues, 0.90)
	stats.P99 = percentile(metricValues, 0.99)

	return stats
}

// percentile returns the nearest-rank percentile value from a sorted slice.
func percentile(sortedValues []int, p float64) int {
	if len(sortedValues) == 0 {
		return 0
	}
	if p <= 0 {
		return sortedValues[0]
	}
	if p >= 1 {
		return sortedValues[len(sortedValues)-1]
	}

	// Nearest-rank method: rank = ceil(p*n), index = rank-1.
	index := int(math.Ceil(p*float64(len(sortedValues)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sortedValues) {
		index = len(sortedValues) - 1
	}

	return sortedValues[index]
}

// collectRawEventPayloads reads SSE lines until the time window elapses and
// returns non-empty payloads from lines prefixed with "data:".
func collectRawEventPayloads(scanner interface {
	Scan() bool
	Text() string
	Err() error
}, streamCloser io.Closer, window time.Duration) ([]string, error) {
	payloads := make([]string, 0, 128)

	lineChannel := make(chan string)
	scannerErrorChannel := make(chan error, 1)

	go func() {
		for scanner.Scan() {
			lineChannel <- scanner.Text()
		}
		scannerErrorChannel <- scanner.Err()
		close(lineChannel)
	}()

	timer := time.NewTimer(window)
	defer timer.Stop()
	timedOut := false

	for {
		select {
		case <-timer.C:
			timedOut = true
			// Closing the stream unblocks scanner and lets the goroutine finish.
			_ = streamCloser.Close()
		case line, ok := <-lineChannel:
			if !ok {
				scannerError := <-scannerErrorChannel
				if scannerError != nil && !timedOut {
					return nil, fmt.Errorf("error while reading SSE stream: %w", scannerError)
				}
				return payloads, nil
			}

			if strings.HasPrefix(line, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if payload != "" {
					payloads = append(payloads, payload)
				}
			}
		}
	}
}
