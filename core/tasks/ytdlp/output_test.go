package ytdlp

import (
	"slices"
	"testing"

	"github.com/krau/SaveAny-Bot/config"
)

func TestRewriteOutputTemplates(t *testing.T) {
	const dir = "/dl"
	tests := []struct {
		name    string
		flags   []string
		want    []string
		wantErr bool
	}{
		{name: "no flags", flags: nil, want: []string{}},
		{name: "unrelated flags", flags: []string{"-f", "best"}, want: []string{"-f", "best"}},
		{name: "short flag with separate value", flags: []string{"-o", "%(title)s.%(ext)s"}, want: []string{"-o", "/dl/%(title)s.%(ext)s"}},
		{name: "long flag with separate value", flags: []string{"--output", "sub/%(title)s.%(ext)s"}, want: []string{"--output", "/dl/sub/%(title)s.%(ext)s"}},
		{name: "long flag with equals", flags: []string{"--output=%(id)s.%(ext)s"}, want: []string{"--output=/dl/%(id)s.%(ext)s"}},
		{name: "short flag with attached value", flags: []string{"-o%(id)s.%(ext)s"}, want: []string{"-o/dl/%(id)s.%(ext)s"}},
		{
			name:  "repeated templates keep their order",
			flags: []string{"-o", "%(id)s.%(ext)s", "--extract-audio", "--output=%(title)s.%(ext)s"},
			want:  []string{"-o", "/dl/%(id)s.%(ext)s", "--extract-audio", "--output=/dl/%(title)s.%(ext)s"},
		},
		{name: "type prefix stays outside", flags: []string{"-o", "subtitle:subs/%(title)s.%(ext)s"}, want: []string{"-o", "subtitle:/dl/subs/%(title)s.%(ext)s"}},
		{name: "dangling flag", flags: []string{"-f", "best", "-o"}, wantErr: true},
		{name: "flag as template", flags: []string{"-o", "--extract-audio"}, wantErr: true},
		{name: "escape", flags: []string{"-o", "../out/%(title)s.%(ext)s"}, wantErr: true},
		{name: "empty template", flags: []string{"--output="}, wantErr: true},
		{name: "capital O is not an output template", flags: []string{"-O", "%(id)s"}, want: []string{"-O", "%(id)s"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewriteOutputTemplates(tt.flags, dir)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("rewriteOutputTemplates(%q) = %q, want an error", tt.flags, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("rewriteOutputTemplates(%q) failed: %v", tt.flags, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("rewriteOutputTemplates(%q) = %q, want %q", tt.flags, got, tt.want)
			}
		})
	}
}

func TestResolveFilenameTemplate(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.YtdlpConfig
		want string
	}{
		{"default", config.YtdlpConfig{}, config.DefaultYtdlpFilenameTemplate},
		{"config template", config.YtdlpConfig{FilenameTemplate: "%(uploader)s - %(title)s.%(ext)s"}, "%(uploader)s - %(title)s.%(ext)s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveFilenameTemplate(tt.cfg); got != tt.want {
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
		{name: "empty", template: "", wantErr: true},
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
