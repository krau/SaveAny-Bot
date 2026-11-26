package webdav

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
)

type Client struct {
	BaseURL          string
	Username         string
	Password         string
	httpClient       *http.Client
	uploadHTTPClient *http.Client
}

type WebdavMethod string

const (
	WebdavMethodMkcol    WebdavMethod = "MKCOL"
	WebdavMethodPropfind WebdavMethod = "PROPFIND"
	WebdavMethodPut      WebdavMethod = "PUT"
	WebdavMethodGet      WebdavMethod = "GET"
)

// WebDAV XML structures for PROPFIND response
type Multistatus struct {
	XMLName   xml.Name   `xml:"multistatus"`
	Responses []Response `xml:"response"`
}

type Response struct {
	Href     string   `xml:"href"`
	Propstat Propstat `xml:"propstat"`
}

type Propstat struct {
	Prop   Prop   `xml:"prop"`
	Status string `xml:"status"`
}

type Prop struct {
	ResourceType     ResourceType `xml:"resourcetype"`
	GetContentLength int64        `xml:"getcontentlength"`
	GetLastModified  string       `xml:"getlastmodified"`
	DisplayName      string       `xml:"displayname"`
}

type ResourceType struct {
	Collection *struct{} `xml:"collection"`
}

func (rt ResourceType) IsCollection() bool {
	return rt.Collection != nil
}

func NewClient(baseURL, username, password string, httpClient *http.Client) *Client {
	return NewClientWithUploadHTTPClient(baseURL, username, password, httpClient, httpClient)
}

// NewClientWithUploadHTTPClient uses a separate client for PUT requests so
// uploads can tolerate a backend that finalizes data after receiving the body.
func NewClientWithUploadHTTPClient(
	baseURL, username, password string,
	httpClient, uploadHTTPClient *http.Client,
) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if uploadHTTPClient == nil {
		uploadHTTPClient = httpClient
	}
	return &Client{
		BaseURL:          baseURL,
		Username:         username,
		Password:         password,
		httpClient:       httpClient,
		uploadHTTPClient: uploadHTTPClient,
	}
}

func (c *Client) newRequest(ctx context.Context, method WebdavMethod, remotePath string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	if u.Path == "" {
		u.Path = "/"
	}
	if remotePath == "" {
		remotePath = "/"
	}
	escapedPath := (&url.URL{Path: remotePath}).EscapedPath()
	u = u.JoinPath(escapedPath)
	req, err := http.NewRequestWithContext(ctx, string(method), u.String(), body)
	if err != nil {
		return nil, err
	}
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
	if method == WebdavMethodPropfind {
		req.Header.Set("Depth", "1")
	}
	if method == WebdavMethodPut {
		if length, ok := ctx.Value(ctxkey.ContentLength).(int64); ok {
			req.ContentLength = length
		}
	}
	return req, nil
}

func (c *Client) Exists(ctx context.Context, remotePath string) (bool, error) {
	req, err := c.newRequest(ctx, WebdavMethodPropfind, remotePath, nil)
	if err != nil {
		return false, err
	}
	resp, err := c.httpClient.Do(req)
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
	if dirPath == "" || dirPath == "." {
		return nil
	}
	parts := strings.Split(dirPath, "/")
	var currentPath strings.Builder
	for i, part := range parts {
		if i > 0 {
			currentPath.WriteString("/")
		}
		currentPath.WriteString(part)
		current := currentPath.String()

		exists, err := c.Exists(ctx, current)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		req, err := c.newRequest(ctx, WebdavMethodMkcol, current, nil)
		if err != nil {
			return err
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("MKCOL %s: %s", current, resp.Status)
		}
	}
	return nil
}

func (c *Client) WriteFile(ctx context.Context, remotePath string, content io.Reader) error {
	return c.writeFile(ctx, remotePath, content, false)
}

// WriteFileIfAbsent creates a file without replacing an existing one.
func (c *Client) WriteFileIfAbsent(ctx context.Context, remotePath string, content io.Reader) error {
	return c.writeFile(ctx, remotePath, content, true)
}

func (c *Client) writeFile(ctx context.Context, remotePath string, content io.Reader, ifAbsent bool) error {
	req, err := c.newRequest(ctx, WebdavMethodPut, strings.Trim(remotePath, "/"), content)
	if err != nil {
		return err
	}
	if ifAbsent {
		req.Header.Set("If-None-Match", "*")
	}
	resp, err := c.uploadHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if ifAbsent && (resp.StatusCode == http.StatusPreconditionFailed || resp.StatusCode == http.StatusConflict) {
		return storagetypes.ErrAlreadyExists
	}
	return fmt.Errorf("PUT: %s", resp.Status)
}

func (c *Client) ListDir(ctx context.Context, dirPath string) ([]Response, error) {
	req, err := c.newRequest(ctx, WebdavMethodPropfind, dirPath+"/", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("PROPFIND: %s", resp.Status)
	}

	var multistatus Multistatus
	if err := xml.NewDecoder(resp.Body).Decode(&multistatus); err != nil {
		return nil, fmt.Errorf("failed to decode PROPFIND response: %w", err)
	}

	var results []Response
	basePath := req.URL.Path
	for _, r := range multistatus.Responses {
		decodedHref, err := url.PathUnescape(r.Href)
		if err != nil {
			decodedHref = r.Href
		}
		if strings.TrimSuffix(decodedHref, "/") == strings.TrimSuffix(basePath, "/") {
			continue
		}
		results = append(results, r)
	}

	return results, nil
}

func (c *Client) ReadFile(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	req, err := c.newRequest(ctx, WebdavMethodGet, strings.Trim(filePath, "/"), nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("GET: %s", resp.Status)
	}

	return resp.Body, resp.ContentLength, nil
}
