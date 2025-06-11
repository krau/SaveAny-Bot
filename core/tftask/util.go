package tftask

var threadsLevels = []struct {
	threads int
	size    int64
}{
	{1, 1 << 20},
	{2, 5 << 20},
	{4, 20 << 20},
	{8, 50 << 20},
}

func BestThreads(size int64, max int) int {
	for _, thread := range threadsLevels {
		if size < thread.size {
			return min(thread.threads, max)
		}
	}
	return max
}
