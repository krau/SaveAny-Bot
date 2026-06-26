package rule

import "path"

// PresetCategory describes a built-in filetype classification: files whose name
// matches Regex are routed into the Dir subdirectory (joined with a user base path).
type PresetCategory struct {
	// Name is a stable identifier for the category (used in logs/messages).
	Name string
	// Regex is a FILENAME-REGEX rule data string matching this category's extensions.
	Regex string
	// Dir is the default subdirectory name for this category.
	Dir string
}

// presetCategories holds the default filetype classification rules.
// Regexes are case-insensitive and match common file extensions.
var presetCategories = []PresetCategory{
	{
		Name:  "video",
		Regex: `(?i)\.(mp4|mkv|ts|avi|flv|mov|webm|wmv|rmvb|m2ts)$`,
		Dir:   "视频",
	},
	{
		Name:  "image",
		Regex: `(?i)\.(jpg|jpeg|png|gif|webp|bmp)$`,
		Dir:   "图片",
	},
	{
		Name:  "audio",
		Regex: `(?i)\.(mp3|flac|wav|aac|m4a|ogg)$`,
		Dir:   "音频",
	},
	{
		Name:  "document",
		Regex: `(?i)\.(pdf|doc|docx|xls|xlsx|ppt|pptx|txt|md|csv|epub|mobi|azw3|chm)$`,
		Dir:   "文档",
	},
	{
		Name:  "archive",
		Regex: `(?i)\.(zip|rar|7z|tar|gz|bz2|xz|r\d{1,3}|z\d{1,3}|\d{3}|part\d+\.rar|7z\.\d{3})$`,
		Dir:   "压缩包",
	},
}

// PresetCategories returns the built-in filetype classification rules with each
// category's directory joined under basePath. basePath may be empty.
func PresetCategories(basePath string) []PresetCategory {
	out := make([]PresetCategory, len(presetCategories))
	for i, c := range presetCategories {
		c.Dir = path.Join(basePath, c.Dir)
		out[i] = c
	}
	return out
}
