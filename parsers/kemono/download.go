package kemono

import (
	"net/url"
	"strings"
)

type DownloadInfo struct {
	ServiceName string
	UserID      string
	PostID      string
}

func extractDownloadInfoFromURL(u string) *DownloadInfo {
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	url, err := url.Parse(u)
	if err != nil {
		return nil
	}
	parts := strings.Split(strings.Trim(url.Path, "/"), "/")
	if len(parts) == 3 {
		return &DownloadInfo{
			ServiceName: parts[0],
			UserID:      parts[2],
		}
	} else if len(parts) == 5 && parts[3] == "post" {
		return &DownloadInfo{
			ServiceName: parts[0],
			UserID:      parts[2],
			PostID:      parts[4],
		}
	}
	return nil
}
