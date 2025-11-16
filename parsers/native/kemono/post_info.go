// https://github.com/kemono-rs/kemono

package kemono

type PostInfo struct {
	Post        Post             `json:"post"`
	Attachments []AttachmentLike `json:"attachments"`
	Previews    []AttachmentLike `json:"previews"`
}

type AttachmentLike struct {
	Type   *string `json:"type,omitempty"`
	Server *string `json:"server,omitempty"`
	Name   *string `json:"name,omitempty"`
	Path   *string `json:"path,omitempty"`
}

type Post struct {
	ID          string           `json:"id"`
	User        string           `json:"user"`
	Service     string           `json:"service"`
	Title       string           `json:"title"`
	Content     string           `json:"content"`
	Embed       Embed            `json:"embed"`
	SharedFile  bool             `json:"shared_file"`
	Added       *string          `json:"added,omitempty"`
	Published   string           `json:"published"`
	Edited      *string          `json:"edited,omitempty"`
	File        File             `json:"file"`
	Attachments []AttachmentLike `json:"attachments"`
	Poll        *Poll            `json:"poll,omitempty"`
	Captions    *string          `json:"captions,omitempty"`
	Tags        *[]string        `json:"tags,omitempty"`
	Next        *string          `json:"next,omitempty"`
	Prev        *string          `json:"prev,omitempty"`
}

type File struct {
	Name *string `json:"name,omitempty"`
	Path *string `json:"path,omitempty"`
}

type Embed struct {
	URL         *string `json:"url,omitempty"`
	Subject     *string `json:"subject,omitempty"`
	Description *string `json:"description,omitempty"`
}

type Poll struct {
	Title          string   `json:"title"`
	Choices        []Choice `json:"choices"`
	ClosesAt       *string  `json:"closes_at,omitempty"`
	CreatedAt      string   `json:"created_at"`
	Description    *string  `json:"description,omitempty"`
	AllowsMultiple bool     `json:"allows_multiple"`
	TotalVotes     int64    `json:"total_votes"`
}

type Choice struct {
	Text  string `json:"text"`
	Votes int64  `json:"votes"`
}
