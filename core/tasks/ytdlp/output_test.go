package ytdlp

import (
	"slices"
	"testing"

	"github.com/krau/SaveAny-Bot/config"
)

func TestSplitOutputTemplate(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		want     string
		wantRest []string
	}{
		{"no flags", nil, "", nil},
		{"unrelated flags", []string{"-f", "best"}, "", []string{"-f", "best"}},
		{"short flag with separate value", []string{"-o", "%(title)s.%(ext)s"}, "%(title)s.%(ext)s", []string{}},
		{"long flag with separate value", []string{"--output", "%(title)s.%(ext)s"}, "%(title)s.%(ext)s", []string{}},
		{"long flag with equals", []string{"--output=%(id)s.%(ext)s"}, "%(id)s.%(ext)s", []string{}},
		{"short flag with attached value", []string{"-o%(id)s.%(ext)s"}, "%(id)s.%(ext)s", []string{}},
		{"last template wins", []string{"-o", "%(id)s.%(ext)s", "--output=%(title)s.%(ext)s"}, "%(title)s.%(ext)s", []string{}},
		{"remaining flags keep their order", []string{"-f", "best", "-o", "%(id)s.%(ext)s", "--extract-audio"}, "%(id)s.%(ext)s", []string{"-f", "best", "--extract-audio"}},
		{"dangling flag is kept for yt-dlp", []string{"-f", "best", "-o"}, "", []string{"-f", "best", "-o"}},
		{"capital O is not an output template", []string{"-O", "%(id)s"}, "", []string{"-O", "%(id)s"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rest := splitOutputTemplate(tt.flags)
			if got != tt.want {
				t.Errorf("template = %q, want %q", got, tt.want)
			}
			if !slices.Equal(rest, tt.wantRest) {
				t.Errorf("remaining flags = %q, want %q", rest, tt.wantRest)
			}
		})
	}
}

func TestResolveFilenameTemplate(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.YtdlpConfig
		userTemplate string
		want         string
	}{
		{"default", config.YtdlpConfig{}, "", config.DefaultYtdlpFilenameTemplate},
		{"config template", config.YtdlpConfig{FilenameTemplate: "%(uploader)s - %(title)s.%(ext)s"}, "", "%(uploader)s - %(title)s.%(ext)s"},
		{"user template wins over config", config.YtdlpConfig{FilenameTemplate: "%(id)s.%(ext)s"}, "%(title)s.%(ext)s", "%(title)s.%(ext)s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveFilenameTemplate(tt.cfg, tt.userTemplate); got != tt.want {
				t.Errorf("resolveFilenameTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}
