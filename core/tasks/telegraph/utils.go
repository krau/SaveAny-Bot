package telegraph

func shouldUpdateProgress(downloaded int64, total int64) bool {
	if total <= 0 || downloaded <= 0 {
		return false
	}

	step := int64(10)
	if downloaded < step {
		return downloaded == total
	}
	return downloaded%step == 0 || downloaded == total
}
