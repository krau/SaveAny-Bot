package tfile

import (
	"testing"

	"github.com/gotd/td/tg"
	tfilepkg "github.com/krau/SaveAny-Bot/pkg/tfile"
)

func TestSourceCaption(t *testing.T) {
	tests := []struct {
		name string
		file tfilepkg.TGFile
		want string
		ok   bool
	}{
		{
			name: "original caption",
			file: tfilepkg.NewTGFile(nil, nil, 0, "video.mov", tfilepkg.WithMessage(&tg.Message{Message: "original caption"})),
			want: "original caption",
			ok:   true,
		},
		{
			name: "empty caption suppresses storage fallback",
			file: tfilepkg.NewTGFile(nil, nil, 0, "video.mov", tfilepkg.WithMessage(&tg.Message{})),
			ok:   true,
		},
		{
			name: "file without source message",
			file: tfilepkg.NewTGFile(nil, nil, 0, "video.mov"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sourceCaption(tt.file)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("sourceCaption() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}
