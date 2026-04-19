package controllers

import (
	"net/url"
	"testing"
	"time"

	"github.com/AlexisHutin/stream-aggregation-service/types"
)

func TestParseAndValidateParams_Valid(t *testing.T) {
	query := url.Values{}
	query.Set("duration", "30s")
	query.Set("dimension", string(types.Likes))

	params, err := parseAndValidateParams(query)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if params.Duration != 30*time.Second {
		t.Fatalf("expected duration 30s, got %v", params.Duration)
	}

	if params.Dimension != types.Likes {
		t.Fatalf("expected dimension %s, got %s", types.Likes, params.Dimension)
	}
}

func TestParseAndValidateParams_MissingDuration(t *testing.T) {
	query := url.Values{}
	query.Set("dimension", string(types.Comments))

	_, err := parseAndValidateParams(query)
	if err == nil {
		t.Fatal("expected an error for missing duration")
	}

	if err.Error() != "missing required query parameter: duration" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAndValidateParams_MissingDimension(t *testing.T) {
	query := url.Values{}
	query.Set("duration", "1m")

	_, err := parseAndValidateParams(query)
	if err == nil {
		t.Fatal("expected an error for missing dimension")
	}

	if err.Error() != "missing required query parameter: dimension" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAndValidateParams_InvalidDurationFormat(t *testing.T) {
	query := url.Values{}
	query.Set("duration", "invalid")
	query.Set("dimension", string(types.Favorites))

	_, err := parseAndValidateParams(query)
	if err == nil {
		t.Fatal("expected an error for invalid duration format")
	}

	if err.Error() != "invalid duration format" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAndValidateParams_InvalidDurationValue(t *testing.T) {
	tests := []struct {
		name     string
		duration string
	}{
		{name: "zero duration", duration: "0s"},
		{name: "negative duration", duration: "-5s"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query := url.Values{}
			query.Set("duration", tc.duration)
			query.Set("dimension", string(types.Retweets))

			_, err := parseAndValidateParams(query)
			if err == nil {
				t.Fatal("expected an error for invalid duration value")
			}

			if err.Error() != "invalid duration value" {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseAndValidateParams_InvalidDimension(t *testing.T) {
	query := url.Values{}
	query.Set("duration", "45s")
	query.Set("dimension", "shares")

	_, err := parseAndValidateParams(query)
	if err == nil {
		t.Fatal("expected an error for invalid dimension")
	}

	if err.Error() != "invalid dimension: shares" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAndValidateParams_TrimsWhitespace(t *testing.T) {
	query := url.Values{}
	query.Set("duration", "  2m  ")
	query.Set("dimension", "  likes  ")

	params, err := parseAndValidateParams(query)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if params.Duration != 2*time.Minute {
		t.Fatalf("expected duration 2m, got %v", params.Duration)
	}

	if params.Dimension != types.Likes {
		t.Fatalf("expected dimension %s, got %s", types.Likes, params.Dimension)
	}
}
