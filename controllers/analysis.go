package controllers

import (
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

	var params types.AnalysisRequestParams
	query := c.Request.URL.Query()

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
	eventsAnalysis, err := eventsanalyser.EventsAnalysis(params.Dimension, params.Duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Status: "error",
			Error:  fmt.Sprintf("Analysis error: %v", err),
		})
		return
	}

	log.Printf("Analysis result: %+v\n", eventsAnalysis)

	c.JSON(http.StatusOK, types.APIResponse{
		Status: "ok",
		Data:   buildAnalysisResponseData(params.Dimension, eventsAnalysis),
	})
}

// buildAnalysisResponseData formats the analysis result with percentile keys
// derived from the requested dimension (for example likes_p50).
func buildAnalysisResponseData(dimension types.Dimension, result types.EventsAnalysisResult) gin.H {
	return gin.H{
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
// - dimension: one of the supported dimensions
func parseAndValidateParams(query url.Values) (types.AnalysisRequestParams, error) {
	var params types.AnalysisRequestParams

	durationStr := strings.TrimSpace(query.Get("duration"))
	if durationStr == "" {
		return params, fmt.Errorf("missing required query parameter: duration")
	}

	validDuration, err := time.ParseDuration(durationStr)
	if err != nil {
		return params, fmt.Errorf("invalid duration format")
	}

	if validDuration <= 0 {
		return params, fmt.Errorf("invalid duration value")
	}

	params.Duration = validDuration

	dimensionStr := strings.TrimSpace(query.Get("dimension"))
	if dimensionStr == "" {
		return params, fmt.Errorf("missing required query parameter: dimension")
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
