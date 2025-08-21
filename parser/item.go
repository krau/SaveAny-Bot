package parser

// Resource is a single downloadable resource with metadata.
type Resource struct {
	URL       string            `json:"url"`
	Filename  string            `json:"filename"`
	MimeType  string            `json:"mime_type"`
	Extension string            `json:"extension"`
	Size      int64             `json:"size"`    // -1 when unknown
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
