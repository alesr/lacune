package gcbench

import (
	"bufio"
	"strconv"
	"strings"
)

// parseGCTrace extracts GC metrics from GODEBUG=gctrace=1 output. The expected
// line shape (Go 1.25+) is:
//
//	gc 1 @0.004s 2%: 0.018+0.46+0.003 ms clock, 0.14+0.20/0.40/0+0.025 ms cpu, 4->4->1 MB, 5 MB goal, 0 MB stacks, 0 MB globals, 8 P
//
// Stop-the-world pause is the first and last components of the "ms clock"
// group; the middle component is concurrent and not counted as a pause.
func parseGCTrace(out string) GCStats {
	var stats GCStats

	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "gc ") || !strings.Contains(line, "ms clock") {
			continue
		}

		stats.NumGC++
		fields := strings.Fields(trimmed)

		for i, f := range fields {
			// cumulative GC CPU fraction, e.g. "2%:" — last value wins
			if before, ok := strings.CutSuffix(f, "%:"); ok {
				if v, err := strconv.ParseFloat(before, 64); err == nil {
					stats.GCCPUPercent = v
				}
			}

			// stop-the-world pause: the triple immediately before "ms clock,"
			if f == "clock," && i >= 2 && fields[i-1] == "ms" {
				parts := strings.Split(fields[i-2], "+")
				if len(parts) > 0 {
					if v, err := strconv.ParseFloat(parts[0], 64); err == nil {
						stats.TotalPauseMs += v
					}
				}

				if len(parts) > 1 {
					if v, err := strconv.ParseFloat(parts[len(parts)-1], 64); err == nil {
						stats.TotalPauseMs += v
					}
				}
			}

			// heap sizes "start->markEnd->live MB"; track the peak (mark end).
			if strings.Contains(f, "->") {
				hp := strings.Split(f, "->")
				if len(hp) == 3 {
					if v, err := strconv.Atoi(strings.TrimSuffix(hp[1], "MB")); err == nil && v > stats.PeakHeapMB {
						stats.PeakHeapMB = v
					}
				}
			}
		}
	}
	return stats
}
