package webdav

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	BaseURL    string
	Username   string
	Password   string
	httpClient *http.Client
}

func NewClient(baseURL, username, password string, httpClient *http.Client) *Client {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		BaseURL:    baseURL,
		Username:   username,
		Password:   password,
		httpClient: httpClient,
	}
}

func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
	return c.httpClient.Do(req)
}

func (c *Client) MkDir(ctx context.Context, dirPath string) error {
	url := c.BaseURL + dirPath
	resp, err := c.doRequest(ctx, "MKCOL", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("MKCOL: %s", resp.Status)
}

func (c *Client) WriteFile(ctx context.Context, remotePath string, content io.Reader) error {
	url := c.BaseURL + remotePath
	resp, err := c.doRequest(ctx, "PUT", url, content)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("PUT: %s", resp.Status)
}
