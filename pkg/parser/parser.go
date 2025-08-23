package parser

import (
	"context"
	"crypto/md5"
	"fmt"
)

type Parser interface {
	CanHandle(url string) bool
	Parse(ctx context.Context, url string) (*Item, error)
}

type ConfigurableParser interface {
	Parser
	Configure(config map[string]any) error
	Name() string
}

// Resource is a single downloadable resource with metadata.
type Resource struct {
	URL       string            `json:"url"`
	Filename  string            `json:"filename"` // with ext
	MimeType  string            `json:"mime_type"`
	Extension string            `json:"extension"` // e.g. "mp4"
	Size      int64             `json:"size"`    // 0 when unknown
	Hash      map[string]string `json:"hash"`    // {"md5": "...", "sha256": "..."}
	Headers   map[string]string `json:"headers"` // HTTP headers when downloading
	Extra     map[string]any    `json:"extra"`
}

type Item struct {
	Site        string         `json:"site"`
	URL         string         `json:"url"` // original URL of the item
	Title       string         `json:"title"`
	Author      string         `json:"author"`
	Description string         `json:"description"`
	Tags        []string       `json:"tags"`
	Resources   []Resource     `json:"resources"`
	Extra       map[string]any `json:"extra"`
}

func (r *Resource) FileName() string {
	return r.Filename
}

func (r *Resource) FileSize() int64 {
	return r.Size
}

func (r *Resource) ID() string {
	h := md5.New()
	h.Write([]byte(r.URL))
	h.Write([]byte(r.Filename))
	h.Write([]byte(r.MimeType))
	h.Write([]byte(r.Extension))
	h.Write([]byte(fmt.Sprintf("%d", r.Size)))

	for k, v := range r.Hash {
		h.Write([]byte(k))
		h.Write([]byte(v))
	}

	for k, v := range r.Headers {
		h.Write([]byte(k))
		h.Write([]byte(v))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
