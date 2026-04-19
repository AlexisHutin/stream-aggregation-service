package eventsanalyser

import (
	"strings"
	"testing"

	"github.com/AlexisHutin/stream-aggregation-service/types"
)

// ── decodeSocialPost ──────────────────────────────────────────────────────────

func TestDecodeSocialPost_Valid(t *testing.T) {
	payload := `{"tweet":{"timestamp":1700000000,"likes":10,"comments":2,"favorites":5,"retweets":1}}`

	post, err := decodeSocialPost(payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if post.Timestamp != 1700000000 {
		t.Fatalf("expected timestamp 1700000000, got %d", post.Timestamp)
	}
	if post.Likes != 10 {
		t.Fatalf("expected likes=10, got %d", post.Likes)
	}
}

func TestDecodeSocialPost_MalformedJSON(t *testing.T) {
	_, err := decodeSocialPost("{not valid json}")
	if err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}

func TestDecodeSocialPost_EmptyObject(t *testing.T) {
	_, err := decodeSocialPost("{}")
	if err == nil {
		t.Fatal("expected an error for empty payload object")
	}
	if !strings.Contains(err.Error(), "empty payload object") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeSocialPost_InvalidInnerJSON(t *testing.T) {
	payload := `{"tweet":"not an object"}`
	_, err := decodeSocialPost(payload)
	if err == nil {
		t.Fatal("expected an error when inner value is not an object")
	}
}

func TestDecodeSocialPost_DifferentKeys(t *testing.T) {
	payloads := []string{
		`{"instagram_media":{"timestamp":100,"likes":3}}`,
		`{"article":{"timestamp":200,"comments":7}}`,
	}

	for _, p := range payloads {
		_, err := decodeSocialPost(p)
		if err != nil {
			t.Fatalf("expected no error for payload %q, got %v", p, err)
		}
	}
}

// ── extractDimensionMetric ────────────────────────────────────────────────────

func TestExtractDimensionMetric_AllDimensions(t *testing.T) {
	post := socialPost{Likes: 1, Comments: 2, Favorites: 3, Retweets: 4}

	tests := []struct {
		dimension types.Dimension
		want      int
	}{
		{types.Likes, 1},
		{types.Comments, 2},
		{types.Favorites, 3},
		{types.Retweets, 4},
	}

	for _, tc := range tests {
		t.Run(string(tc.dimension), func(t *testing.T) {
			got, err := extractDimensionMetric(post, tc.dimension)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestExtractDimensionMetric_InvalidDimension(t *testing.T) {
	_, err := extractDimensionMetric(socialPost{}, "shares")
	if err == nil {
		t.Fatal("expected an error for unsupported dimension")
	}
	if !strings.Contains(err.Error(), "unsupported dimension") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── buildDimensionSamples ─────────────────────────────────────────────────────

func TestBuildDimensionSamples_Empty(t *testing.T) {
	samples, err := buildDimensionSamples([]string{}, types.Likes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(samples) != 0 {
		t.Fatalf("expected empty samples, got %d", len(samples))
	}
}

func TestBuildDimensionSamples_Valid(t *testing.T) {
	payloads := []string{
		`{"tweet":{"timestamp":1000,"likes":5}}`,
		`{"tweet":{"timestamp":2000,"likes":15}}`,
	}

	samples, err := buildDimensionSamples(payloads, types.Likes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}
	if samples[0].Metric != 5 || samples[1].Metric != 15 {
		t.Fatalf("unexpected metric values: %v", samples)
	}
}

func TestBuildDimensionSamples_BadPayload(t *testing.T) {
	payloads := []string{
		`{"tweet":{"timestamp":1000,"likes":5}}`,
		`{not valid json}`,
	}

	_, err := buildDimensionSamples(payloads, types.Likes)
	if err == nil {
		t.Fatal("expected an error when a payload is malformed")
	}
}

func TestBuildDimensionSamples_InvalidDimension(t *testing.T) {
	payloads := []string{
		`{"tweet":{"timestamp":1000,"likes":5}}`,
	}

	_, err := buildDimensionSamples(payloads, "shares")
	if err == nil {
		t.Fatal("expected an error for invalid dimension")
	}
}
