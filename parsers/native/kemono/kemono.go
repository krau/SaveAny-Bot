package kemono

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/netutil"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

type KemonoParser struct{}

var (
	kemonoDomains = []string{
		"kemono.su",
		"kemono.cr",
	}
	ErrFailedToExtractInfo = errors.New("failed to extract download info from URL")
)

const (
	kemonoApiBase = "https://kemono.cr/api/v1"
)

func (k *KemonoParser) CanHandle(text string) bool {
	text = strings.TrimPrefix(text, "https://")
	text = strings.TrimPrefix(text, "http://")

	var matchesDomain bool
	for _, domain := range kemonoDomains {
		if strings.Contains(text, domain) {
			matchesDomain = true
			break
		}
	}
	if !matchesDomain {
		return false
	}

	var path string
	for _, domain := range kemonoDomains {
		if idx := strings.Index(text, domain); idx != -1 {
			remaining := text[idx+len(domain):]
			if len(remaining) > 0 && remaining[0] == '/' {
				path = remaining[1:]
			}
			break
		}
	}

	if path == "" {
		return false
	}

	parts := strings.Split(path, "/")
	// servicename/user/id (user profile page)
	// servicename/user/id/post/id (post page)
	return len(parts) == 3 || (len(parts) == 5 && parts[3] == "post")
}

func (k *KemonoParser) Parse(ctx context.Context, u string) (*parser.Item, error) {
	info := extractDownloadInfoFromURL(u)
	if info == nil {
		return nil, ErrFailedToExtractInfo
	}
	if info.PostID != "" {
		return k.parseOne(ctx, info)
	}
	return k.parseUserPage(ctx, info)
}

func (k *KemonoParser) parseOne(ctx context.Context, info *DownloadInfo) (*parser.Item, error) {
	client := netutil.DefaultParserHTTPClient()
	endpoint := fmt.Sprintf("%s/%s/user/%s/post/%s", kemonoApiBase, info.ServiceName, info.UserID, info.PostID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to Kemono API: %w", err)
	}
	req.Header.Set("Accept", "text/css")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Kemono API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Kemono API, status code: %d", resp.StatusCode)
	}
	var postInfo PostInfo
	if err := json.NewDecoder(resp.Body).Decode(&postInfo); err != nil {
		return nil, fmt.Errorf("failed to decode Kemono API response: %w", err)
	}
	item := &parser.Item{
		Site:        "kemono",
		Title:       postInfo.Post.Title,
		URL:         fmt.Sprintf("https://kemono.cr/%s/user/%s/post/%s", info.ServiceName, info.UserID, info.PostID),
		Author:      postInfo.Post.User, // [TODO] request user profile
		Description: postInfo.Post.Content,
		Tags: func() []string {
			if postInfo.Post.Tags != nil {
				return *postInfo.Post.Tags
			}
			return nil
		}(),
	}
	resources := make([]parser.Resource, 0)
	for _, attachment := range postInfo.Attachments {
		if attachment.Server == nil || attachment.Path == nil || attachment.Name == nil {
			continue
		}
		var size int64
		fileUrl := fmt.Sprintf("%s/data%s", *attachment.Server, *attachment.Path)
		headReq, err := http.NewRequestWithContext(ctx, http.MethodHead, fileUrl, nil)
		if err == nil {
			resp, err := client.Do(headReq)
			if err == nil {
				size = resp.ContentLength
				resp.Body.Close()
			}
		}
		resources = append(resources, parser.Resource{
			URL:      fmt.Sprintf("%s/data%s", *attachment.Server, *attachment.Path),
			Filename: *attachment.Name,
			Size:     size,
		})
	}
	picCdnMap := make(map[string]string)
	for _, preview := range postInfo.Previews {
		if preview.Type == nil || *preview.Type != "thumbnail" {
			continue
		}
		picCdnMap[*preview.Path] = *preview.Server
	}
	for _, attachment := range postInfo.Post.Attachments {
		if !isImageExt(*attachment.Path) {
			continue
		}
		picUrl, err := url.JoinPath(picCdnMap[*attachment.Path], "data", *attachment.Path)
		if err != nil {
			continue
		}
		var size int64
		headReq, err := http.NewRequestWithContext(ctx, http.MethodHead, picUrl, nil)
		if err == nil {
			resp, err := client.Do(headReq)
			if err == nil {
				size = resp.ContentLength
				resp.Body.Close()
			}
		}
		resources = append(resources, parser.Resource{
			URL:      picUrl,
			Filename: *attachment.Name,
			Size:     size,
		})
	}
	item.Resources = resources
	return item, nil
}

func (k *KemonoParser) parseUserPage(_ context.Context, _ *DownloadInfo) (*parser.Item, error) {
	return nil, errors.New("kemono user page not implemented")
}

func isImageExt(attachmentPath string) bool {
	return strutil.HasSuffixAny(path.Ext(strings.Split(attachmentPath, "?")[0]), []string{".jpg", ".jpeg", ".png", ".webp"})
}
