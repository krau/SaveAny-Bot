package s3

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// keyRoundTrip returns the wire and signed paths for key.
func keyRoundTrip(t *testing.T, c *Client, key string) (wire string, signed string) {
	t.Helper()

	raw, err := c.buildURL(key)
	if err != nil {
		t.Fatalf("buildURL(%q) failed: %v", key, err)
	}

	// net/http re-parses the string, so anything buildURL leaves ambiguous
	// (a bare '#' or '?') is resolved here, not inside buildURL.
	req, err := http.NewRequest(http.MethodPut, raw, nil)
	if err != nil {
		t.Fatalf("NewRequest for key %q failed: %v", key, err)
	}

	return req.URL.EscapedPath(), canonicalURI(req.URL.Path)
}

func TestBuildURLKeyEncoding(t *testing.T) {
	keys := []string{
		"plain.txt",
		"dir/sub/nested.txt",
		"with space.txt",
		"plus+sign.txt",
		"hash#fragment.txt",
		"question?mark.txt",
		"percent%sign.txt",
		"amp&equals=comma,.txt",
		"parens()[]{}.txt",
		"at@colon:semi;.txt",

		// Non-ASCII scripts
		"سورة الفاتحة.mp3",
		"أذان الفجر.mp3",
		"دعاء القنوت ١٤٤٧.mp3",
		"中文视频.mp4",
		"講座：入門篇.mp4",
		"日本語.txt",

		// Media titles like yt-dlp actually produces them
		"Playlist - Clip (Official Video) [4K].mp4",
		"Don't Stop Me Now.mp3",
		"Pembahasan Tauhid, Bagian 1.mp4",
		"تفسير سورة البقرة (الجزء الأول) [HD].mp4",
		"سورة الكهف & سورة مريم - المصحف المرتل.mp3",
		"خطبة الجمعة، المسجد الحرام @ مكة.mp4",
		"纪录片 & 访谈.mp3",
		"Live，第二集 @ 吉隆坡.mp4",
	}

	clients := map[string]*Client{
		"path-style":    {endpoint: "https://s3.example.com", bucket: "bucket", pathStyle: true},
		"virtual-style": {endpoint: "https://s3.example.com", bucket: "bucket"},
	}

	for name, c := range clients {
		t.Run(name, func(t *testing.T) {
			for _, key := range keys {
				t.Run(key, func(t *testing.T) {
					wire, signed := keyRoundTrip(t, c, key)

					// A mismatch here is a SignatureDoesNotMatch at runtime:
					// we would sign one URI and send another.
					if wire != signed {
						t.Errorf("wire URI %q does not match signed URI %q", wire, signed)
					}

					// And the server must decode our path back to the key we
					// asked for, rather than a truncated or mangled one.
					decoded, err := url.PathUnescape(wire)
					if err != nil {
						t.Fatalf("wire URI %q is not valid encoding: %v", wire, err)
					}
					got := strings.TrimPrefix(strings.TrimPrefix(decoded, "/bucket"), "/")
					if got != key {
						t.Errorf("key round-tripped as %q, want %q (wire %q)", got, key, wire)
					}
				})
			}
		})
	}
}

// TestBuildURLEveryPrintableASCII checks every printable key byte so reserved
// characters cannot make the request URI differ from the URI being signed.
func TestBuildURLEveryPrintableASCII(t *testing.T) {
	clients := map[string]*Client{
		"path-style":    {endpoint: "https://s3.example.com", bucket: "bucket", pathStyle: true},
		"virtual-style": {endpoint: "https://s3.example.com", bucket: "bucket"},
	}

	for name, c := range clients {
		t.Run(name, func(t *testing.T) {
			for b := byte(0x20); b < 0x7f; b++ {
				if b == '/' {
					continue // a separator, not a key byte
				}
				key := "a" + string(b) + "b.mp4"

				wire, signed := keyRoundTrip(t, c, key)
				if wire != signed {
					t.Errorf("byte %q (0x%02X): wire URI %q does not match signed URI %q", b, b, wire, signed)
				}

				decoded, err := url.PathUnescape(wire)
				if err != nil {
					t.Errorf("byte %q (0x%02X): wire URI %q is not valid encoding: %v", b, b, wire, err)
					continue
				}
				got := strings.TrimPrefix(strings.TrimPrefix(decoded, "/bucket"), "/")
				if got != key {
					t.Errorf("byte %q (0x%02X): key round-tripped as %q, want %q", b, b, got, key)
				}
			}
		})
	}
}

func TestBuildURLBucketPlacement(t *testing.T) {
	cases := []struct {
		name   string
		client *Client
		want   string
	}{
		{
			name:   "path-style puts the bucket in the path",
			client: &Client{endpoint: "https://s3.example.com", bucket: "bucket", pathStyle: true},
			want:   "https://s3.example.com/bucket/key.txt",
		},
		{
			name:   "virtual style puts the bucket in the host",
			client: &Client{endpoint: "https://s3.example.com", bucket: "bucket"},
			want:   "https://bucket.s3.example.com/key.txt",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.client.buildURL("key.txt")
			if err != nil {
				t.Fatalf("buildURL failed: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEscapePathMatchesCanonicalURI(t *testing.T) {
	// canonicalURI is the signing side and escapePath the request side; if they
	// ever diverge, every key containing a reserved byte breaks.
	cases := []string{
		"/dir/sub",
		"/a+b",
		"/a b",
		"/日本語",
		"/الرسمية",
		"/a&b=c,d",
	}

	for _, path := range cases {
		got, want := canonicalURI(path), escapePath(path)
		if got != want {
			t.Errorf("canonicalURI(%q) = %q, escapePath = %q", path, got, want)
		}
	}

	if got := canonicalURI(""); got != "/" {
		t.Errorf(`canonicalURI("") = %q, want "/"`, got)
	}
}
