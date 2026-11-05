package tphutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
)

var defaultClient = struct {
	sync.Mutex
	key    string
	client *telegraph.Client
}{}

func DefaultClient() *telegraph.Client {
	cfg := config.C().Telegram.Proxy
	key := "direct"
	if cfg.Enable && cfg.URL != "" {
		key = cfg.URL
	}
	defaultClient.Lock()
	defer defaultClient.Unlock()
	if defaultClient.client != nil && defaultClient.key == key {
		return defaultClient.client
	}
	client, err := newClient(key)
	if err != nil {
		log.Warn("Failed to create Telegraph client with proxy; using direct client", "error", err)
		client = telegraph.NewClient()
	}
	defaultClient.client = client
	defaultClient.key = key
	return client
}

func newClient(key string) (*telegraph.Client, error) {
	if key == "direct" {
		return telegraph.NewClient(), nil
	}
	return telegraph.NewClientWithProxy(key)
}

func GetNodeImages(node telegraph.Node) []string {
	var srcs []string

	var nodeElement telegraph.NodeElement
	data, err := json.Marshal(node)
	if err != nil {
		return srcs
	}
	err = json.Unmarshal(data, &nodeElement)
	if err != nil {
		return srcs
	}

	if nodeElement.Tag == "img" {
		if src, exists := nodeElement.Attrs["src"]; exists {
			if strings.HasPrefix(src, "/file/") {
				src = "https://telegra.ph" + src
			}
			srcs = append(srcs, src)
		}
	}
	for _, child := range nodeElement.Children {
		srcs = append(srcs, GetNodeImages(child)...)
	}
	return srcs
}

// ParsePagePath validates a Telegraph page URL and returns its path. Bot
// messages require HTTPS; the HTTP API also accepts legacy HTTP links.
func ParsePagePath(pageURL string) (string, error) {
	return parsePagePath(pageURL, false)
}

func parsePagePath(pageURL string, allowHTTP bool) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("invalid telegraph URL: %w", err)
	}
	if u.Scheme != "https" && !(allowHTTP && u.Scheme == "http") {
		return "", fmt.Errorf("invalid telegraph URL scheme: %s", u.Scheme)
	}
	if !strings.EqualFold(u.Hostname(), "telegra.ph") && !strings.EqualFold(u.Hostname(), "telegraph.co") {
		return "", fmt.Errorf("invalid telegraph URL host: %s", u.Host)
	}
	pagePath := strings.Trim(u.EscapedPath(), "/")
	if pagePath == "" || strings.Contains(pagePath, "/") {
		return "", fmt.Errorf("invalid telegraph URL path: %s", u.Path)
	}
	pagePath, err = url.PathUnescape(pagePath)
	if err != nil {
		return "", fmt.Errorf("failed to unescape telegraph path: %w", err)
	}
	pagePath = strings.TrimSpace(pagePath)
	if pagePath == "" || strings.Contains(pagePath, "/") {
		return "", fmt.Errorf("invalid telegraph URL path: %s", u.Path)
	}
	return pagePath, nil
}

// ExtractImages fetches a Telegraph page and returns its image URLs and path.
func ExtractImages(ctx context.Context, pageURL string) ([]string, string, error) {
	pagePath, err := parsePagePath(pageURL, true)
	if err != nil {
		return nil, "", err
	}
	page, err := DefaultClient().GetPage(ctx, pagePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get telegraph page: %w", err)
	}
	images := make([]string, 0)
	for _, elem := range page.Content {
		images = append(images, GetNodeImages(elem)...)
	}
	if len(images) == 0 {
		return nil, "", errors.New("no images found in telegraph page")
	}
	return images, pagePath, nil
}
