package common

import (
	"crypto/md5"
	"encoding/hex"
	"regexp"
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
	return tags
}
