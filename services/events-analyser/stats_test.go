package eventsanalyser

import (
	"fmt"
	"math"
	"testing"
)

// ── percentile ────────────────────────────────────────────────────────────────

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		values []int
		p      float64
		want   int
	}{
		{name: "empty slice returns 0", values: []int{}, p: 0.50, want: 0},
		{name: "single value p50", values: []int{42}, p: 0.50, want: 42},
		{name: "single value p90", values: []int{42}, p: 0.90, want: 42},
		{name: "single value p99", values: []int{42}, p: 0.99, want: 42},
		{name: "p<=0 returns minimum", values: []int{1, 2, 3, 4, 5}, p: 0, want: 1},
		{name: "p>=1 returns maximum", values: []int{1, 2, 3, 4, 5}, p: 1, want: 5},
		{name: "negative p returns minimum", values: []int{10, 20, 30}, p: -1, want: 10},
		{name: "p>1 returns maximum", values: []int{10, 20, 30}, p: 2, want: 30},
		// [1..10]: nearest-rank ceil(0.5*10)=5 → index 4 → 5
		{name: "ten values p50", values: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, p: 0.50, want: 5},
		// ceil(0.9*10)=9 → index 8 → 9
		{name: "ten values p90", values: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, p: 0.90, want: 9},
		// ceil(0.99*10)=10 → index 9 → 10
		{name: "ten values p99", values: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, p: 0.99, want: 10},
		// two values
		{name: "two values p50", values: []int{3, 7}, p: 0.50, want: 3},
		{name: "two values p99", values: []int{3, 7}, p: 0.99, want: 7},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := percentile(tc.values, tc.p)
			if got != tc.want {
				t.Fatalf("percentile(%v, %v) = %d, want %d", tc.values, tc.p, got, tc.want)
			}
		})
	}
}

// ── computeAnalysisStats ─────────────────────────────────────────────────────

func TestComputeAnalysisStats_Empty(t *testing.T) {
	stats := computeAnalysisStats([]dimensionSample{})

	if stats.TotalPosts != 0 {
		t.Fatalf("expected 0 TotalPosts, got %d", stats.TotalPosts)
	}
	if stats.MinimumTimestamp != 0 || stats.MaximumTimestamp != 0 {
		t.Fatalf("expected zero timestamps, got min=%d max=%d", stats.MinimumTimestamp, stats.MaximumTimestamp)
	}
	if stats.P50 != 0 || stats.P90 != 0 || stats.P99 != 0 {
		t.Fatalf("expected zero percentiles, got P50=%d P90=%d P99=%d", stats.P50, stats.P90, stats.P99)
	}
}

func TestComputeAnalysisStats_Single(t *testing.T) {
	samples := []dimensionSample{
		{Timestamp: 1000, Metric: 50},
	}

	stats := computeAnalysisStats(samples)

	if stats.TotalPosts != 1 {
		t.Fatalf("expected 1 TotalPosts, got %d", stats.TotalPosts)
	}
	if stats.MinimumTimestamp != 1000 || stats.MaximumTimestamp != 1000 {
		t.Fatalf("expected timestamp 1000, got min=%d max=%d", stats.MinimumTimestamp, stats.MaximumTimestamp)
	}
	if stats.P50 != 50 || stats.P90 != 50 || stats.P99 != 50 {
		t.Fatalf("expected all percentiles=50, got P50=%d P90=%d P99=%d", stats.P50, stats.P90, stats.P99)
	}
}

func TestComputeAnalysisStats_Multiple(t *testing.T) {
	samples := make([]dimensionSample, 10)
	for i := range 10 {
		samples[i] = dimensionSample{
			Timestamp: int64(1000 + i),
			Metric:    i + 1, // values 1..10
		}
	}

	stats := computeAnalysisStats(samples)

	if stats.TotalPosts != 10 {
		t.Fatalf("expected 10 TotalPosts, got %d", stats.TotalPosts)
	}
	if stats.MinimumTimestamp != 1000 {
		t.Fatalf("expected min timestamp 1000, got %d", stats.MinimumTimestamp)
	}
	if stats.MaximumTimestamp != 1009 {
		t.Fatalf("expected max timestamp 1009, got %d", stats.MaximumTimestamp)
	}
	if stats.P50 != 5 {
		t.Fatalf("expected P50=5, got %d", stats.P50)
	}
	if stats.P90 != 9 {
		t.Fatalf("expected P90=9, got %d", stats.P90)
	}
	if stats.P99 != 10 {
		t.Fatalf("expected P99=10, got %d", stats.P99)
	}
}

func TestComputeAnalysisStats_UnsortedTimestamps(t *testing.T) {
	samples := []dimensionSample{
		{Timestamp: 3000, Metric: 10},
		{Timestamp: 1000, Metric: 20},
		{Timestamp: 2000, Metric: 30},
	}

	stats := computeAnalysisStats(samples)

	if stats.MinimumTimestamp != 1000 {
		t.Fatalf("expected min timestamp 1000, got %d", stats.MinimumTimestamp)
	}
	if stats.MaximumTimestamp != 3000 {
		t.Fatalf("expected max timestamp 3000, got %d", stats.MaximumTimestamp)
	}
	_ = math.MaxInt // keep import used
}

// BenchmarkPercentile_P99_10k measures pure percentile lookup cost on an
// already sorted 10k slice. It helps track regressions in the percentile
// function itself, independently from sorting/allocation overhead.
func BenchmarkPercentile_P99_10k(b *testing.B) {
	values := make([]int, 10_000)
	for i := range values {
		values[i] = i
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = percentile(values, 0.99)
	}
}

// BenchmarkComputeAnalysisStats_10k measures end-to-end stats computation on
// 10k events, including metric extraction bookkeeping and internal sorting for
// percentile calculation. Use ns/op and B/op to monitor scalability impact.
func BenchmarkComputeAnalysisStats_10k(b *testing.B) {
	samples := make([]dimensionSample, 10_000)
	for i := range samples {
		samples[i] = dimensionSample{
			Timestamp: int64(1_700_000_000 + i),
			Metric:    (i * 37) % 1000,
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = computeAnalysisStats(samples)
	}
}

// BenchmarkPercentile_P99_Scales measures percentile lookup across larger
// sorted datasets to observe scaling behavior with realistic high volumes.
func BenchmarkPercentile_P99_Scales(b *testing.B) {
	sizes := []int{50_000, 100_000, 500_000, 1_000_000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("N_%d", size), func(b *testing.B) {
			values := make([]int, size)
			for i := range values {
				values[i] = i
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = percentile(values, 0.99)
			}
		})
	}
}

// BenchmarkComputeAnalysisStats_Scales measures end-to-end stats computation
// at larger sizes (including sorting) to track CPU and allocation growth.
func BenchmarkComputeAnalysisStats_Scales(b *testing.B) {
	sizes := []int{50_000, 100_000, 500_000, 1_000_000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("N_%d", size), func(b *testing.B) {
			samples := make([]dimensionSample, size)
			for i := range samples {
				samples[i] = dimensionSample{
					Timestamp: int64(1_700_000_000 + i),
					Metric:    (i * 37) % 1000,
				}
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = computeAnalysisStats(samples)
			}
		})
	}
}
