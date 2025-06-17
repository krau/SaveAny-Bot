package webdav

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
)

type Client struct {
	BaseURL    string
	Username   string
	Password   string
	httpClient *http.Client
}

type WebdavMethod string

const (
	WebdavMethodMkcol    WebdavMethod = "MKCOL"
	WebdavMethodPropfind WebdavMethod = "PROPFIND"
	WebdavMethodPut      WebdavMethod = "PUT"
)

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

func (c *Client) doRequest(ctx context.Context, method WebdavMethod, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, string(method), url, body)
	if err != nil {
		return nil, err
	}
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
	if method == WebdavMethodPropfind {
		req.Header.Set("Depth", "1")
	}
	if method == WebdavMethodPut && ctx != nil {
		if length := ctx.Value(ctxkey.ContentLength); length != nil {
			if l, ok := length.(int64); ok {
				req.ContentLength = l
			}
		}
	}
	return c.httpClient.Do(req)
}

func (c *Client) Exists(ctx context.Context, remotePath string) (bool, error) {
	url := c.BaseURL + remotePath
	resp, err := c.doRequest(ctx, WebdavMethodPropfind, url, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("PROPFIND: %s", resp.Status)
}

func (c *Client) MkDir(ctx context.Context, dirPath string) error {
	dirPath = strings.Trim(dirPath, "/")
	if dirPath == "" {
		return nil
	}
	parts := strings.Split(dirPath, "/")
	currentPath := ""
	for i, part := range parts {
		if i > 0 {
			currentPath += "/"
		}
		currentPath += part

		exists, err := c.Exists(ctx, currentPath)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		url := c.BaseURL + currentPath
		resp, err := c.doRequest(ctx, WebdavMethodMkcol, url, nil)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("MKCOL %s: %s", currentPath, resp.Status)
		}
	}
	return nil
}

func (c *Client) WriteFile(ctx context.Context, remotePath string, content io.Reader) error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	parts := strings.Split(strings.Trim(remotePath, "/"), "/")
	u.Path = path.Join(u.Path, strings.Join(parts, "/"))
	resp, err := c.doRequest(ctx, WebdavMethodPut, u.String(), content)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("PUT: %s", resp.Status)

}
