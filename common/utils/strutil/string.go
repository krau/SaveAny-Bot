package strutil

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
)

func HashString(s string) string {
	hash := md5.New()
	hash.Write([]byte(s))
	return hex.EncodeToString(hash.Sum(nil))
}

var TagRe = regexp.MustCompile(`(?:^|[\p{Zs}\s.,!?(){}[\]<>\"\'，。！？（）：；、])#([\p{L}\d_]+)`)

func ExtractTagsFromText(text string) []string {
	matches := TagRe.FindAllStringSubmatch(text, -1)
	tags := make([]string, 0)
	for _, match := range matches {
		if len(match) > 1 {
			tags = append(tags, match[1])
		}
	}
	return slice.Compact(tags)
}

func ParseIntStrRange(input string, sep string) (int64, int64, error) {
	parts := strings.Split(input, sep)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format: %s", input)
	}
	min, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minimum value: %s", parts[0])
	}
	max, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid maximum value: %s", parts[1])
	}
	if min > max {
		min, max = max, min
	}
	return min, max, nil
}
