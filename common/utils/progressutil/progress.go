// Package progressutil provides shared progress-update throttling for task
// progress trackers.
package progressutil

// updatesLevels picks the percent step by file size.
var updatesLevels = []struct {
	size        int64 // file size threshold
	stepPercent int   // minimum percent step between updates
}{
	{10 << 20, 100},
	{50 << 20, 20},
	{200 << 20, 10},
	{500 << 20, 5},
}

// ShouldUpdate reports whether a byte-based progress update should be shown.
func ShouldUpdate(total, downloaded int64, lastUpdatePercent int) bool {
	if total <= 0 || downloaded <= 0 {
		return false
	}

	percent := int((downloaded * 100) / total)
	if percent <= lastUpdatePercent {
		return false
	}

	step := updatesLevels[len(updatesLevels)-1].stepPercent
	for _, lvl := range updatesLevels {
		if total < lvl.size {
			step = lvl.stepPercent
			break
		}
	}

	return percent >= lastUpdatePercent+step
}

// ShouldUpdateCount reports whether a count-based progress update (e.g. files
// downloaded so far) should be shown: every 10 units, or when finished.
func ShouldUpdateCount(downloaded, total int64) bool {
	if total <= 0 || downloaded <= 0 {
		return false
	}

	const step = int64(10)
	if downloaded < step {
		return downloaded == total
	}
	return downloaded%step == 0 || downloaded == total
}
