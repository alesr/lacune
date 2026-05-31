package gcbench

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGCTrace(t *testing.T) {
	t.Parallel()

	const sampleLine = "gc 1 @0.004s 2%: 0.018+0.46+0.003 ms clock, 0.14+0.20/0.40/0+0.025 ms cpu, 4->4->1 MB, 5 MB goal, 0 MB stacks, 0 MB globals, 8 P"

	tests := []struct {
		name string
		in   string
		want GCStats
	}{
		{
			name: "empty",
			in:   "",
			want: GCStats{},
		},
		{
			name: "single line",
			in:   sampleLine,
			want: GCStats{
				NumGC:        1,
				TotalPauseMs: 0.021, // 0.018 + 0.003
				GCCPUPercent: 2,
				PeakHeapMB:   4,
			},
		},
		{
			name: "multiple lines",
			in: sampleLine + "\n" +
				"gc 2 @0.010s 5%: 0.010+0.30+0.010 ms clock, 0.10+0.15/0.20/0+0.020 ms cpu, 8->10->6 MB, 12 MB goal, 0 MB stacks, 0 MB globals, 8 P",
			want: GCStats{
				NumGC:        2,
				TotalPauseMs: 0.041, // (0.018+0.003) + (0.010+0.010)
				GCCPUPercent: 5,
				PeakHeapMB:   10,
			},
		},
		{
			name: "noise ignored",
			in: "=== RUN   TestFoo\n" +
				"--- PASS: TestFoo (0.00s)\n" +
				sampleLine + "\n" +
				"PASS\n",
			want: GCStats{
				NumGC:        1,
				TotalPauseMs: 0.021,
				GCCPUPercent: 2,
				PeakHeapMB:   4,
			},
		},
		{
			name: "no ms clock",
			in:   "gc 1 @0.004s 2%: 0.018+0.46+0.003 ms cpu, 4->4->1 MB",
			want: GCStats{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseGCTrace(tt.in)

			assert.Equal(t, tt.want.NumGC, got.NumGC)
			assert.InDelta(t, tt.want.TotalPauseMs, got.TotalPauseMs, 0.001)
			assert.InDelta(t, tt.want.GCCPUPercent, got.GCCPUPercent, 0.001)
			assert.Equal(t, tt.want.PeakHeapMB, got.PeakHeapMB)
			assert.Zero(t, got.WallTime)
		})
	}
}
