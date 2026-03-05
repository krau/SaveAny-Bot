package dlutil

import (
	"fmt"
	"time"
)

var threadsLevels = []struct {
	threads int
	size    int64
}{
	{1, 10 << 20},
	{2, 50 << 20},
	{4, 200 << 20},
	{8, 500 << 20},
}

func BestThreads(size int64, max int) int {
	for _, thread := range threadsLevels {
		if size < thread.size {
			return min(thread.threads, max)
		}
	}
	return max
}

func GetSpeed(downloaded int64, startTime time.Time) float64 {
	if startTime.IsZero() {
		return 0
	}
	elapsed := time.Since(startTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(downloaded) / elapsed
}

// FormatSize formats a byte size as a human-readable string
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
