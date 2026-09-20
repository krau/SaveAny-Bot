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

func TestOutputTemplatePath(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{name: "file name", template: "%(title)s.%(ext)s", want: "/dl/%(title)s.%(ext)s"},
		{name: "subdirectory", template: "sub/%(title)s.%(ext)s", want: "/dl/sub/%(title)s.%(ext)s"},
		{name: "absolute template stays inside", template: "/var/tmp/%(title)s.%(ext)s", want: "/dl/var/tmp/%(title)s.%(ext)s"},
		{name: "type prefix stays outside", template: "subtitle:subs/%(title)s.%(ext)s", want: "subtitle:/dl/subs/%(title)s.%(ext)s"},
		{name: "combined type prefix", template: "subtitle+thumbnail:subs/%(title)s.%(ext)s", want: "subtitle+thumbnail:/dl/subs/%(title)s.%(ext)s"},
		{name: "unknown key is a directory name", template: "season:1/%(title)s.%(ext)s", want: "/dl/season:1/%(title)s.%(ext)s"},
		{name: "escape", template: "../escaped/%(title)s.%(ext)s", wantErr: true},
		{name: "escape below a subdirectory", template: "sub/../../escaped/%(title)s.%(ext)s", wantErr: true},
		{name: "directory itself", template: "sub/..", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := outputTemplatePath("/dl", tt.template)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("outputTemplatePath(%q) = %q, want an error", tt.template, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("outputTemplatePath(%q) failed: %v", tt.template, err)
			}
			if got != tt.want {
				t.Errorf("outputTemplatePath(%q) = %q, want %q", tt.template, got, tt.want)
			}
		})
	}
}
