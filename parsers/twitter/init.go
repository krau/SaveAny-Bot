package twitter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/krau/SaveAny-Bot/common/op"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

type TwitterParser struct {
	client http.Client
}

const (
	FxTwitterApi = "api.fxtwitter.com"
)

func (p *TwitterParser) Parse(u string) (*parser.Item, error) {
	parts := strings.Split(u, "/")
	if len(parts) < 4 || parts[3] != "status" {
		return nil, errors.New("invalid Twitter URL")
	}
	id := parts[4]
	apiUrl := fmt.Sprintf("https://%s/_/status/%s", FxTwitterApi, id)
	resp, err := p.client.Get(apiUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Twitter API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Twitter API, status code: %d", resp.StatusCode)
	}
	var fxResp FxTwitterApiResp
	if err := json.NewDecoder(resp.Body).Decode(&fxResp); err != nil {
		return nil, fmt.Errorf("failed to decode Twitter API response: %w", err)
	}
	if fxResp.Code != 200 {
		return nil, fmt.Errorf("request twitter API error: %s", fxResp.Message)
	}
	if len(fxResp.Tweet.Media.All) == 0 {
		return nil, errors.New("no media found in the tweet")
	}
	resources := make([]parser.Resource, 0, len(fxResp.Tweet.Media.All))
	for _, media := range fxResp.Tweet.Media.All {
		resources = append(resources, parser.Resource{
			URL:      media.URL,
			Filename: path.Base(strings.Split(media.URL, "?")[0]),
		})
	}
	item := &parser.Item{
		Site:        "Twitter",
		Title:       fmt.Sprintf("Tweet/%s", id),
		URL:         fxResp.Tweet.URL,
		Description: fxResp.Tweet.Text,
		Author:      fxResp.Tweet.Author.Name,
		Tags:        make([]string, 0),
		Extra:       make(map[string]any),
		Resources:   resources,
	}
	return item, nil
}

func (p *TwitterParser) CanHandle(u string) bool {
	url1, err := url.Parse(u)
	if err != nil {
		return false
	}
	if url1.Host == "twitter.com" || url1.Host == "x.com" {
		path := strings.TrimPrefix(url1.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) >= 3 && parts[1] == "status" {
			return true
		}
	}
	return false
}

func init() {
	op.RegisterParser(func() parser.Parser {
		return &TwitterParser{
			client: http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyFromEnvironment,
				},
			},
		}
	})
}
