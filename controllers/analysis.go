package controllers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AlexisHutin/stream-aggregation-service/config"
	ea "github.com/AlexisHutin/stream-aggregation-service/services/events-analyser"
	"github.com/AlexisHutin/stream-aggregation-service/types"
	"github.com/gin-gonic/gin"
)

// AnalysisHandler handles GET /analysis requests.
//
// It validates query parameters and returns a JSON payload with normalized
// analysis inputs when validation succeeds.
func AnalysisHandler(c *gin.Context) {
	if c.Request.Method != http.MethodGet {
		c.JSON(http.StatusMethodNotAllowed, types.APIResponse{
			Status: "error",
			Error:  "Method Not Allowed",
		})
		return
	}
	ctx := c.Request.Context()
	query := c.Request.URL.Query()
	var params types.AnalysisRequestParams

	params, err := parseAndValidateParams(query)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Status: "error",
			Error:  err.Error(),
		})
		return
	}

	configs, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	eventsanalyser := ea.NewEventsAnalyser(configs.StreamURL)

	log.Printf("Performing analysis for dimension: %s, duration: %s\n", params.Dimension, params.Duration)
	eventsAnalysis, err := eventsanalyser.EventsAnalysis(ctx, params.Dimension, params.Duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Status: "error",
			Error:  fmt.Sprintf("Analysis error: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, buildAnalysisResponseData(params.Dimension, eventsAnalysis))
}

// buildAnalysisResponseData formats the analysis result with percentile keys
// derived from the requested dimension (for example likes_p50).
func buildAnalysisResponseData(dimension types.Dimension, result types.EventsAnalysisResult) map[string]interface{} {
	return map[string]interface{}{
		"total_posts":                    result.TotalPosts,
		"minimum_timestamp":              result.MinimumTimestamp,
		"maximum_timestamp":              result.MaximumTimestamp,
		fmt.Sprintf("%s_p50", dimension): result.DimensionP50,
		fmt.Sprintf("%s_p90", dimension): result.DimensionP90,
		fmt.Sprintf("%s_p99", dimension): result.DimensionP99,
	}
}

// parseAndValidateParams parses and validates the analysis query parameters.
//
// Required parameters are:
// - duration: a positive Go duration string (for example, 5s, 30m or 1h)
// - dimension: one of the supported dimensions.
func parseAndValidateParams(query url.Values) (types.AnalysisRequestParams, error) {
	var params types.AnalysisRequestParams

	durationStr := strings.TrimSpace(query.Get("duration"))
	if durationStr == "" {
		return params, errors.New("missing required query parameter: duration")
	}

	validDuration, err := time.ParseDuration(durationStr)
	if err != nil {
		return params, errors.New("invalid duration format")
	}

	if validDuration <= 0 {
		return params, errors.New("invalid duration value")
	}

	params.Duration = validDuration

	dimensionStr := strings.TrimSpace(query.Get("dimension"))
	if dimensionStr == "" {
		return params, errors.New("missing required query parameter: dimension")
	}

	if !isValidDimension(types.Dimension(dimensionStr)) {
		return params, fmt.Errorf("invalid dimension: %s", dimensionStr)
	}

	params.Dimension = types.Dimension(dimensionStr)

	return params, nil
}

// isValidDimension reports whether d is a supported analysis dimension.
func isValidDimension(d types.Dimension) bool {
	switch d {
	case types.Likes, types.Comments, types.Favorites, types.Retweets:
		return true
	default:
		return false
	}
}
