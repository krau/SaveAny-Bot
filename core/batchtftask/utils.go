package batchtftask

var progressUpdatesLevels = []struct {
	size        int64 // 文件大小阈值
	stepPercent int   // 每多少 % 更新一次
}{
	{10 << 20, 100},
	{50 << 20, 20},
	{200 << 20, 10},
	{500 << 20, 5},
}

func shouldUpdateProgress(total, downloaded int64) bool {
	if total <= 0 || downloaded <= 0 {
		return false
	}

	percent := int((downloaded * 100) / total)

	var step int
	for _, level := range progressUpdatesLevels {
		if total < level.size {
			step = level.stepPercent
			break
		}
	}

	if step == 0 {
		step = progressUpdatesLevels[len(progressUpdatesLevels)-1].stepPercent
	}

	return percent > 0 && percent%step == 0
}
