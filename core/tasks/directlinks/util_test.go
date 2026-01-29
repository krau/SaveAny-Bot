package directlinks

import (
	"testing"
)

func TestParseFilenameFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple filename",
			url:      "https://example.com/files/document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "filename with encoded characters",
			url:      "https://example.com/files/%E6%B5%8B%E8%AF%95.zip",
			expected: "测试.zip",
		},
		{
			name:     "filename with query string in URL",
			url:      "https://example.com/files/image.png?token=abc123",
			expected: "image.png",
		},
		{
			name:     "nested path",
			url:      "https://example.com/a/b/c/file.txt",
			expected: "file.txt",
		},
		{
			name:     "URL with port",
			url:      "https://example.com:8080/downloads/archive.tar.gz",
			expected: "archive.tar.gz",
		},
		{
			name:     "empty path",
			url:      "https://example.com",
			expected: "",
		},
		{
			name:     "root path only",
			url:      "https://example.com/",
			expected: "",
		},
		{
			name:     "filename with spaces encoded",
			url:      "https://example.com/my%20file%20name.pdf",
			expected: "my file name.pdf",
		},
		{
			name:     "complex encoded filename",
			url:      "https://example.com/downloads/%E4%B8%AD%E6%96%87%E6%96%87%E4%BB%B6.docx",
			expected: "中文文件.docx",
		},
		{
			name:     "invalid URL",
			url:      "://invalid-url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFilenameFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("parseFilenameFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}
