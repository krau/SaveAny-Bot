package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/utils/tphutil"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
)

// ExtractTelegraphImages 从 Telegraph URL 提取图片
func ExtractTelegraphImages(ctx context.Context, pageURL string) ([]string, string, error) {
	logger := log.FromContext(ctx)

	// 验证 URL 格式
	if !isValidTelegraphURL(pageURL) {
		return nil, "", fmt.Errorf("invalid telegraph URL format: %s", pageURL)
	}

	// 解析 URL 获取页面路径
	pagepath, err := parseTelegraphPath(pageURL)
	if err != nil {
		return nil, "", err
	}

	logger.Debugf("Fetching telegraph page: %s", pagepath)

	client := telegraph.NewClient()
	page, err := client.GetPage(ctx, pagepath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get telegraph page: %w", err)
	}

	var imgs []string
	for _, elem := range page.Content {
		imgs = append(imgs, tphutil.GetNodeImages(elem)...)
	}

	if len(imgs) == 0 {
		return nil, "", fmt.Errorf("no images found in telegraph page")
	}

	return imgs, pagepath, nil
}

// parseTelegraphPath 解析 Telegraph URL 获取页面路径
func parseTelegraphPath(pageURL string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("invalid telegraph URL: %w", err)
	}

	if !strings.HasSuffix(u.Host, "telegra.ph") && !strings.HasSuffix(u.Host, "telegraph.co") {
		return "", fmt.Errorf("invalid telegraph URL host: %s", u.Host)
	}

	paths := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(paths) == 0 || paths[0] == "" {
		return "", fmt.Errorf("invalid telegraph URL path: %s", u.Path)
	}

	pagepath := paths[len(paths)-1]
	pagepath, err = url.PathUnescape(pagepath)
	if err != nil {
		return "", fmt.Errorf("failed to unescape telegraph path: %w", err)
	}

	return strings.TrimSpace(pagepath), nil
}

// isValidTelegraphURL 检查是否是有效的 Telegraph URL
func isValidTelegraphURL(url string) bool {
	return strings.HasPrefix(url, "https://telegra.ph/") ||
		strings.HasPrefix(url, "http://telegra.ph/") ||
		strings.HasPrefix(url, "https://telegraph.co/") ||
		strings.HasPrefix(url, "http://telegraph.co/")
}
