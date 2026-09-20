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
		wantErr  bool
	}{
		{name: "no flags", flags: nil, want: "", wantRest: nil},
		{name: "unrelated flags", flags: []string{"-f", "best"}, want: "", wantRest: []string{"-f", "best"}},
		{name: "short flag with separate value", flags: []string{"-o", "%(title)s.%(ext)s"}, want: "%(title)s.%(ext)s", wantRest: []string{}},
		{name: "long flag with separate value", flags: []string{"--output", "%(title)s.%(ext)s"}, want: "%(title)s.%(ext)s", wantRest: []string{}},
		{name: "long flag with equals", flags: []string{"--output=%(id)s.%(ext)s"}, want: "%(id)s.%(ext)s", wantRest: []string{}},
		{name: "short flag with attached value", flags: []string{"-o%(id)s.%(ext)s"}, want: "%(id)s.%(ext)s", wantRest: []string{}},
		{name: "last template wins", flags: []string{"-o", "%(id)s.%(ext)s", "--output=%(title)s.%(ext)s"}, want: "%(title)s.%(ext)s", wantRest: []string{}},
		{name: "remaining flags keep their order", flags: []string{"-f", "best", "-o", "%(id)s.%(ext)s", "--extract-audio"}, want: "%(id)s.%(ext)s", wantRest: []string{"-f", "best", "--extract-audio"}},
		{name: "dangling flag", flags: []string{"-f", "best", "-o"}, wantErr: true},
		{name: "flag as template", flags: []string{"-o", "--extract-audio"}, wantErr: true},
		{name: "capital O is not an output template", flags: []string{"-O", "%(id)s"}, want: "", wantRest: []string{"-O", "%(id)s"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rest, err := splitOutputTemplate(tt.flags)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("splitOutputTemplate(%q) = (%q, %q), want an error", tt.flags, got, rest)
				}
				return
			}
			if err != nil {
				t.Fatalf("splitOutputTemplate(%q) failed: %v", tt.flags, err)
			}
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
