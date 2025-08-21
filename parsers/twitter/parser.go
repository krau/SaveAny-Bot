package twitter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/krau/SaveAny-Bot/pkg/parser"
)

type TwitterParser struct {
	client http.Client
}

const (
	FxTwitterApi = "api.fxtwitter.com"
)

var _ parser.Parser = (*TwitterParser)(nil)

var (
	twitterSourceURLRegexp *regexp.Regexp = regexp.MustCompile(`(?:twitter|x)\.com/([^/]+)/status/(\d+)`)
)

func getTweetID(sourceURL string) string {
	matches := twitterSourceURLRegexp.FindStringSubmatch(sourceURL)
	if len(matches) < 3 {
		return ""
	}
	return matches[2]
}

func (p *TwitterParser) Parse(u string) (*parser.Item, error) {
	id := getTweetID(u)
	if id == "" {
		return nil, errors.New("invalid Twitter URL")
	}
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
	return twitterSourceURLRegexp.MatchString(u)
}
