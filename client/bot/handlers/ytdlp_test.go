package handlers

import (
	"slices"
	"testing"

	"github.com/charmbracelet/log"
)

// TestYtdlpArgumentParsing tests the URL and flag separation logic
func TestYtdlpArgumentParsing(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedURLs  []string
		expectedFlags []string
	}{
		{
			name:          "Single URL without flags",
			input:         "/ytdlp https://example.com/video",
			expectedURLs:  []string{"https://example.com/video"},
			expectedFlags: []string{},
		},
		{
			name:          "Multiple URLs without flags",
			input:         "/ytdlp https://example.com/v1 https://example.com/v2",
			expectedURLs:  []string{"https://example.com/v1", "https://example.com/v2"},
			expectedFlags: []string{},
		},
		{
			name:          "URL with format flag",
			input:         "/ytdlp --format best https://example.com/video",
			expectedURLs:  []string{"https://example.com/video"},
			expectedFlags: []string{"--format", "best"},
		},
		{
			name:          "URL with extract-audio flag",
			input:         "/ytdlp --extract-audio --audio-format mp3 https://example.com/video",
			expectedURLs:  []string{"https://example.com/video"},
			expectedFlags: []string{"--extract-audio", "--audio-format", "mp3"},
		},
		{
			name:          "Multiple URLs with flags",
			input:         "/ytdlp --format best https://example.com/v1 https://example.com/v2",
			expectedURLs:  []string{"https://example.com/v1", "https://example.com/v2"},
			expectedFlags: []string{"--format", "best"},
		},
		{
			name:          "Flags mixed with URLs",
			input:         "/ytdlp https://example.com/v1 --format best https://example.com/v2",
			expectedURLs:  []string{"https://example.com/v1", "https://example.com/v2"},
			expectedFlags: []string{"--format", "best"},
		},
		{
			name:          "Short flag",
			input:         "/ytdlp -f best https://example.com/video",
			expectedURLs:  []string{"https://example.com/video"},
			expectedFlags: []string{"-f", "best"},
		},
		{
			name:          "Boolean flag",
			input:         "/ytdlp --extract-audio https://example.com/video",
			expectedURLs:  []string{"https://example.com/video"},
			expectedFlags: []string{"--extract-audio"},
		},
		{
			name:          "Output template flag with spaces",
			input:         `/ytdlp -o "%(uploader)s - %(title)s.%(ext)s" https://example.com/video`,
			expectedURLs:  []string{"https://example.com/video"},
			expectedFlags: []string{"-o", "%(uploader)s - %(title)s.%(ext)s"},
		},
		{
			name:          "Single quoted template",
			input:         `/ytdlp -o '%(title)s.%(ext)s' https://example.com/video`,
			expectedURLs:  []string{"https://example.com/video"},
			expectedFlags: []string{"-o", "%(title)s.%(ext)s"},
		},
		{
			name:          "Invalid URL is dropped",
			input:         "/ytdlp not-a-url -f best",
			expectedURLs:  []string{},
			expectedFlags: []string{"-f", "best"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := splitQuotedArgs(tt.input)[1:]
			urls, flags := parseYtdlpArgs(log.FromContext(t.Context()), args)

			if !slices.Equal(urls, tt.expectedURLs) {
				t.Errorf("urls = %q, want %q", urls, tt.expectedURLs)
			}
			if !slices.Equal(flags, tt.expectedFlags) {
				t.Errorf("flags = %q, want %q", flags, tt.expectedFlags)
			}
		})
	}
}
