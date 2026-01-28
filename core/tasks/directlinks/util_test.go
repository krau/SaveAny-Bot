package directlinks

import (
	"testing"
)

func TestFilenameFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple file",
			url:      "https://example.com/file.zip",
			expected: "file.zip",
		},
		{
			name:     "file with path",
			url:      "https://example.com/path/to/document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "url with query params",
			url:      "https://example.com/file.mp4?token=abc123",
			expected: "file.mp4",
		},
		{
			name:     "url with fragment",
			url:      "https://example.com/file.txt#section",
			expected: "file.txt",
		},
		{
			name:     "url encoded filename",
			url:      "https://example.com/%E6%B5%8B%E8%AF%95.zip",
			expected: "测试.zip",
		},
		{
			name:     "url encoded Chinese filename",
			url:      "https://example.com/10%E6%9C%8817%E6%97%A5(6).mp4",
			expected: "10月17日(6).mp4",
		},
		{
			name:     "root path only",
			url:      "https://example.com/",
			expected: "",
		},
		{
			name:     "no path",
			url:      "https://example.com",
			expected: "",
		},
		{
			name:     "empty url",
			url:      "",
			expected: "",
		},
		{
			name:     "file with spaces encoded",
			url:      "https://example.com/my%20file.txt",
			expected: "my file.txt",
		},
		{
			name:     "complex path with multiple slashes",
			url:      "https://cdn.example.com/a/b/c/d/e/video.mkv",
			expected: "video.mkv",
		},
		{
			name:     "malformed url with invalid characters",
			url:      "://invalid url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filenameFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("filenameFromURL(%q) = %q, expected %q", tt.url, result, tt.expected)
			}
		})
	}
}
