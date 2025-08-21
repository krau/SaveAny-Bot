package parser

type Parser interface {
	CanHandle(url string) bool
	Parse(url string) (*Item, error)
}

// Resource is a single downloadable resource with metadata.
type Resource struct {
	URL       string            `json:"url"`
	Filename  string            `json:"filename"` // with ext
	MimeType  string            `json:"mime_type"`
	Extension string            `json:"extension"`
	Size      int64             `json:"size"`    // -1 when unknown
	Hash      map[string]string `json:"hash"`    // {"md5": "...", "sha256": "..."}
	Headers   map[string]string `json:"headers"` // HTTP headers when downloading
	Extra     map[string]any    `json:"extra"`
}

func (r *Resource) FileName() string {
	return r.Filename
}

func (r *Resource) FileSize() int64 {
	return r.Size
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
