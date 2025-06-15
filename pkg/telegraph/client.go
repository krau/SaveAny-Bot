// https://github.com/celestix/telegraph-go

package telegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Page object represents a page on Telegraph.
type Page struct {
	// Path to the page.
	Path string `json:"path"`
	// URL of the page.
	Url string `json:"url"`
	// Title of the page.
	Title string `json:"title"`
	// Description of the page.
	Description string `json:"description"`
	// Optional. Name of the author, displayed below the title.
	AuthorName string `json:"author_name,omitempty"`
	// Optional. Profile link, opened when users click on the author's name below the title.  Can be any link, not necessarily to a Telegram profile or channel.
	AuthorUrl string `json:"author_url,omitempty"`
	// Optional. Image URL of the page.
	ImageUrl string `json:"image_url,omitempty"`
	// Optional. Content of the page.
	Content []Node `json:"content,omitempty"`
	// Number of page views for the page.
	Views int64 `json:"views"`
	// Optional. Only returned if access_token passed. True, if the target Telegraph account can edit the page.
	CanEdit bool `json:"can_edit,omitempty"`
}

// Node is abstract object represents a DOM Node. It can be a String which represents a DOM text node or a
// NodeElement object.
type Node any

// NodeElement represents a DOM element node.
type NodeElement struct {
	// Name of the DOM element. Available tags: a, aside, b, blockquote, br, code, em, figcaption, figure,
	// h3, h4, hr, i, iframe, img, li, ol, p, pre, s, strong, u, ul, video.Client
	Tag string `json:"tag"`

	// Attributes of the DOM element. Key of object represents name of attribute, value represents value
	// of attribute. Available attributes: href, src.
	Attrs map[string]string `json:"attrs,omitempty"`

	// List of child nodes for the DOM element.
	Children []Node `json:"children,omitempty"`
}

type Client struct {
	client *http.Client
}

type Body struct {
	// Ok: if true, request was successful, and result can be found in the Result field.
	// If false, error can be explained in Error field.
	Ok bool `json:"ok"`
	// Error: contains a human-readable description of the error result.
	Error string `json:"error"`
	// Result: result of requests (if Ok)
	Result json.RawMessage `json:"result"`
}

const (
	ApiUrl = "https://api.telegra.ph/"
)

func (c *Client) InvokeRequest(ctx context.Context, method string, params url.Values) (json.RawMessage, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, ApiUrl+method, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to build POST request to %s: %w", method, err)
	}

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to execute POST request to %s: %w", method, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var b Body
	if err = json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return nil, fmt.Errorf("failed to parse response from %s: %w", method, err)
	}
	if !b.Ok {
		return nil, fmt.Errorf("failed to %s: %s", method, b.Error)
	}
	return b.Result, nil
}

func (c *Client) GetPage(ctx context.Context, phpath string) (*Page, error) {
	var (
		u = url.Values{}
		a Page
	)
	u.Add("path", phpath)
	u.Add("return_content", "true")
	r, err := c.InvokeRequest(ctx, "getPage", u)
	if err != nil {
		return nil, err
	}
	return &a, json.Unmarshal(r, &a)
}

// Helper to use the client(*http.Client) to download a file from a given URL.
func (c *Client) Download(ctx context.Context, durl string) (io.ReadCloser, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, durl, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file from %s: %s", durl, resp.Status)
	}
	return resp.Body, nil
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{},
	}
}

func NewClientWithProxy(proxyUrl string) (*Client, error) {
	u, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}
	p := http.ProxyURL(u)
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: p,
		},
	}
	return &Client{
		client: httpClient,
	}, nil
}
