package handlers

import (
	"net/url"
	"strings"
	"testing"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := strings.Split(tt.input, " ")

			// Simulate the parsing logic from handleYtdlpCmd
			var urls []string
			var flags []string

			for i := 1; i < len(args); i++ {
				arg := strings.TrimSpace(args[i])
				if arg == "" {
					continue
				}

				// Check if it's a flag (starts with - or --)
				if strings.HasPrefix(arg, "-") {
					flags = append(flags, arg)
					// Check if the next argument might be a value for this flag
					if i+1 < len(args) {
						nextArg := strings.TrimSpace(args[i+1])
						if nextArg != "" && !strings.HasPrefix(nextArg, "-") {
							// Check if it's clearly a URL (has ://)
							if strings.Contains(nextArg, "://") {
								// It's a URL, don't consume it as a flag value
								continue
							}
							// Otherwise, treat it as a flag value
							flags = append(flags, nextArg)
							i++ // Skip the next argument as it's been consumed
						}
					}
				} else {
					// Try to parse as URL
					u, err := url.Parse(arg)
					if err != nil || u.Scheme == "" || u.Host == "" {
						continue
					}
					urls = append(urls, arg)
				}
			}

			// Verify URLs
			if len(urls) != len(tt.expectedURLs) {
				t.Errorf("Expected %d URLs, got %d", len(tt.expectedURLs), len(urls))
			}
			for i, expectedURL := range tt.expectedURLs {
				if i >= len(urls) || urls[i] != expectedURL {
					t.Errorf("Expected URL[%d] to be '%s', got '%s'", i, expectedURL, urls[i])
				}
			}

			// Verify flags
			if len(flags) != len(tt.expectedFlags) {
				t.Errorf("Expected %d flags, got %d", len(tt.expectedFlags), len(flags))
			}
			for i, expectedFlag := range tt.expectedFlags {
				if i >= len(flags) || flags[i] != expectedFlag {
					t.Errorf("Expected flag[%d] to be '%s', got '%s'", i, expectedFlag, flags[i])
				}
			}
		})
	}
}
