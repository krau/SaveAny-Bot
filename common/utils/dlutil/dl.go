package dlutil

import "time"

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
