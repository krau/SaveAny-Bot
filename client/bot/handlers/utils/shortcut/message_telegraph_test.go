package shortcut

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestFindTelegraphURL(t *testing.T) {
	tests := []struct {
		name string
		msg  *tg.Message
		want string
	}{
		{
			name: "single URL entity",
			msg: &tg.Message{
				Message: "https://telegra.ph/Example-01-02",
				Entities: []tg.MessageEntityClass{
					&tg.MessageEntityURL{Offset: 0, Length: 32},
				},
			},
			want: "https://telegra.ph/Example-01-02",
		},
		{
			name: "Telegraph URL before another URL",
			msg: &tg.Message{
				Message: "https://telegra.ph/Example-01-02 https://example.com/",
				Entities: []tg.MessageEntityClass{
					&tg.MessageEntityURL{Offset: 0, Length: 32},
					&tg.MessageEntityURL{Offset: 33, Length: 20},
				},
			},
			want: "https://telegra.ph/Example-01-02",
		},
		{
			name: "hidden Telegraph URL",
			msg: &tg.Message{
				Message: "article",
				Entities: []tg.MessageEntityClass{
					&tg.MessageEntityTextURL{
						Offset: 0,
						Length: 7,
						URL:    "https://telegra.ph/Hidden-01-02",
					},
				},
			},
			want: "https://telegra.ph/Hidden-01-02",
		},
		{
			name: "URL entity after non-BMP character",
			msg: &tg.Message{
				Message: "😀 https://telegra.ph/Emoji-01-02",
				Entities: []tg.MessageEntityClass{
					&tg.MessageEntityURL{Offset: 3, Length: 30},
				},
			},
			want: "https://telegra.ph/Emoji-01-02",
		},
		{
			name: "valid Telegraph URL after invalid candidate",
			msg: &tg.Message{
				Message: "https://telegra.ph/nested/Bad https://telegra.ph/Valid-01-02",
				Entities: []tg.MessageEntityClass{
					&tg.MessageEntityURL{Offset: 0, Length: 29},
					&tg.MessageEntityURL{Offset: 30, Length: 30},
				},
			},
			want: "https://telegra.ph/Valid-01-02",
		},
		{
			name: "plain message fallback",
			msg:  &tg.Message{Message: "read https://telegra.ph/Plain-01-02 now"},
			want: "https://telegra.ph/Plain-01-02",
		},
		{
			name: "nil message",
			msg:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findTelegraphURL(tt.msg); got != tt.want {
				t.Fatalf("findTelegraphURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseTelegraphPagePath(t *testing.T) {
	tests := []struct {
		name    string
		pageURL string
		want    string
		wantErr bool
	}{
		{
			name:    "plain path",
			pageURL: "https://telegra.ph/Example-01-02",
			want:    "Example-01-02",
		},
		{
			name:    "escaped path with query and fragment",
			pageURL: "https://telegra.ph/%E6%B5%8B%E8%AF%95-01-02?source=telegram#top",
			want:    "测试-01-02",
		},
		{
			name:    "trailing slash",
			pageURL: "https://telegra.ph/Example-01-02/",
			want:    "Example-01-02",
		},
		{
			name:    "root URL",
			pageURL: "https://telegra.ph/",
			wantErr: true,
		},
		{
			name:    "wrong host",
			pageURL: "https://example.com/Example-01-02",
			wantErr: true,
		},
		{
			name:    "wrong scheme",
			pageURL: "http://telegra.ph/Example-01-02",
			wantErr: true,
		},
		{
			name:    "invalid percent escape",
			pageURL: "https://telegra.ph/Invalid-%zz",
			wantErr: true,
		},
		{
			name:    "nested path",
			pageURL: "https://telegra.ph/nested/Example-01-02",
			wantErr: true,
		},
		{
			name:    "encoded slash",
			pageURL: "https://telegra.ph/nested%2FExample-01-02",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTelegraphPagePath(tt.pageURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseTelegraphPagePath(%q) returned no error", tt.pageURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTelegraphPagePath(%q) failed: %v", tt.pageURL, err)
			}
			if got != tt.want {
				t.Fatalf("parseTelegraphPagePath(%q) = %q, want %q", tt.pageURL, got, tt.want)
			}
		})
	}
}
