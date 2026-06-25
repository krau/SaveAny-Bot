package ytdlp

import "testing"

func TestBuildFormatSelector(t *testing.T) {
	tests := []struct {
		name      string
		maxHeight int
		want      string
	}{
		{"no limit", 0, ""},
		{"negative", -1, ""},
		{"1080p", 1080, "bv*[height<=1080]+ba/b[height<=1080]/b"},
		{"720p", 720, "bv*[height<=720]+ba/b[height<=720]/b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildFormatSelector(tt.maxHeight); got != tt.want {
				t.Errorf("buildFormatSelector(%d) = %q, want %q", tt.maxHeight, got, tt.want)
			}
		})
	}
}
