package api

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gotd/td/tg"
	"github.com/krau/ffmpeg-go"
	ytdlp "github.com/lrstanley/go-ytdlp"
)

var (
	telegramMessageContextResolver = getMessageWithContext
	directMediaMetadataExtractor   = extractDirectMediaMetadata
	ytdlpMediaMetadataExtractor    = extractYTDLPMediaMetadata
)

func extractMediaMetadata(ctx context.Context, url string) (*MediaMetadataResponse, error) {
	if isValidMessageLink(url) {
		return extractTelegramMediaMetadata(ctx, url)
	}

	if resp, err := directMediaMetadataExtractor(ctx, url); err == nil {
		return resp, nil
	}

	return ytdlpMediaMetadataExtractor(ctx, url)
}

func extractTelegramMediaMetadata(ctx context.Context, link string) (*MediaMetadataResponse, error) {
	chatID, msgID, err := ParseMessageLink(ctx, link)
	if err != nil {
		return nil, fmt.Errorf("failed to parse telegram link: %w", err)
	}

	msgCtx, err := telegramMessageContextResolver(ctx, chatID, msgID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve telegram message: %w", err)
	}

	if msgCtx.Message == nil {
		return nil, fmt.Errorf("telegram message is nil")
	}

	if msgCtx.Message.Media == nil {
		return nil, fmt.Errorf("telegram message has no media")
	}

	return &MediaMetadataResponse{
		URL:             link,
		Title:           extractTelegramTitle(msgCtx.Message),
		Uploader:        extractTelegramUploader(msgCtx.Message),
		DurationSeconds: extractTelegramDuration(msgCtx.Message),
	}, nil
}

func extractTelegramDuration(msg *tg.Message) float64 {
	media := msg.Media

	documentMedia, ok := media.(*tg.MessageMediaDocument)
	if !ok || documentMedia == nil || documentMedia.Document == nil {
		// telegram media does not contain a document
		return 0
	}

	document, ok := documentMedia.Document.AsNotEmpty()
	if !ok {
		// telegram document is empty
		return 0
	}

	for _, attribute := range document.Attributes {
		switch attr := attribute.(type) {
		case *tg.DocumentAttributeVideo:
			if attr.Duration > 0 {
				return float64(attr.Duration)
			}
		case *tg.DocumentAttributeAudio:
			if attr.Duration > 0 {
				return float64(attr.Duration)
			}
		}
	}

	return 0
}

func extractTelegramTitle(msg *tg.Message) string {
	if msg == nil {
		return ""
	}

	if msg.Media != nil {
		if documentMedia, ok := msg.Media.(*tg.MessageMediaDocument); ok && documentMedia != nil && documentMedia.Document != nil {
			if document, ok := documentMedia.Document.AsNotEmpty(); ok {
				for _, attribute := range document.Attributes {
					switch attr := attribute.(type) {
					case *tg.DocumentAttributeAudio:
						if attr.Title != "" {
							return attr.Title
						}
					case *tg.DocumentAttributeFilename:
						if attr.FileName != "" {
							return attr.FileName
						}
					}
				}
			}
		}
	}

	return strings.TrimSpace(msg.Message)
}

func extractTelegramUploader(msg *tg.Message) string {
	if msg == nil {
		return ""
	}
	if msg.PostAuthor != "" {
		return msg.PostAuthor
	}
	if msg.FwdFrom.FromName != "" {
		return msg.FwdFrom.FromName
	}
	if msg.FwdFrom.PostAuthor != "" {
		return msg.FwdFrom.PostAuthor
	}

	if msg.Media != nil {
		if documentMedia, ok := msg.Media.(*tg.MessageMediaDocument); ok && documentMedia != nil && documentMedia.Document != nil {
			if document, ok := documentMedia.Document.AsNotEmpty(); ok {
				for _, attribute := range document.Attributes {
					switch attr := attribute.(type) {
					case *tg.DocumentAttributeAudio:
						if attr.Performer != "" {
							return attr.Performer
						}
					}
				}
			}
		}
	}

	return ""
}

func extractDirectMediaMetadata(ctx context.Context, url string) (*MediaMetadataResponse, error) {
	probeResult, err := ffmpeg.ProbeWithTimeout(
		url,
		10*time.Second,
		ffmpeg.KwArgs{
			"show_entries": "format=duration",
			"of":           "json",
			"v":            "error",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to probe direct media: %w", err)
	}

	var data struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal([]byte(probeResult), &data); err != nil {
		return nil, fmt.Errorf("failed to parse direct media metadata: %w", err)
	}

	var duration float64
	if _, err := fmt.Sscanf(strings.TrimSpace(data.Format.Duration), "%f", &duration); err != nil || duration <= 0 {
		return nil, fmt.Errorf("duration is not available for direct media")
	}

	return &MediaMetadataResponse{
		URL:             url,
		Title:           deriveTitleFromURL(url),
		DurationSeconds: duration,
	}, nil
}

func extractYTDLPMediaMetadata(ctx context.Context, url string) (*MediaMetadataResponse, error) {
	cmd := ytdlp.New().DumpSingleJSON()
	result, err := cmd.Run(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect media: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to inspect media: %s", result.Stderr)
	}

	var e *ytdlp.ExtractedInfo
	infoList := make([]*ytdlp.ExtractedInfo, 0)

	for _, log := range result.OutputLogs {
		e, err = parseYTDLPExtractedInfoLog(log.Line, log.JSON)
		if err != nil {
			continue
		}
		if e == nil {
			continue
		}

		infoList = append(infoList, e)
	}

	if len(infoList) == 0 || infoList[0] == nil {
		return nil, fmt.Errorf("no media metadata found for url")
	}

	if infoList[0].Duration == nil {
		return nil, fmt.Errorf("duration is not available for url")
	}

	return &MediaMetadataResponse{
		URL:             url,
		Title:           firstString(infoList[0].Title, infoList[0].AltTitle),
		Thumbnail:       firstString(infoList[0].Thumbnail),
		Uploader:        firstString(infoList[0].Uploader, infoList[0].Channel, infoList[0].Creator, infoList[0].Artist, infoList[0].AlbumArtist),
		DurationSeconds: *infoList[0].Duration,
	}, nil
}

func deriveTitleFromURL(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	trimmed = strings.TrimRight(trimmed, "/")
	base := path.Base(trimmed)
	if base == "." || base == "/" || base == "" {
		return ""
	}
	return filepath.Base(base)
}

func firstString(values ...*string) string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			return strings.TrimSpace(*value)
		}
	}
	return ""
}

func parseYTDLPExtractedInfoLog(line string, rawJSON *json.RawMessage) (*ytdlp.ExtractedInfo, error) {
	if rawJSON != nil && len(*rawJSON) > 0 {
		return parseYTDLPExtractedInfoBytes(*rawJSON)
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil, nil
	}

	info, lineErr := parseYTDLPExtractedInfoBytes(json.RawMessage([]byte(trimmed)))
	if info != nil {
		return info, nil
	}

	start := strings.IndexByte(trimmed, '{')
	end := strings.LastIndexByte(trimmed, '}')
	if start == -1 {
		return nil, nil
	}
	if end == -1 || start >= end {
		return nil, lineErr
	}

	info, err := parseYTDLPExtractedInfoBytes(json.RawMessage([]byte(trimmed[start : end+1])))
	if err != nil {
		return nil, err
	}
	if info != nil {
		return info, nil
	}

	return nil, lineErr
}

func parseYTDLPExtractedInfoBytes(data json.RawMessage) (*ytdlp.ExtractedInfo, error) {
	e, err := ytdlp.ParseExtractedInfo(&data)
	if err != nil {
		return nil, err
	}
	if e.Type == "" {
		return nil, nil
	}
	return e, nil
}
