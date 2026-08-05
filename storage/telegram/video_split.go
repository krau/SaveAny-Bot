package telegram

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message"
	"github.com/rs/xid"

	"github.com/krau/SaveAny-Bot/config"
)

const (
	videoPartTargetRatio  = 0.95
	videoSplitAttempts    = 4
	minSegmentDuration    = 1.0
	maxLosslessVideoParts = 10
)

type losslessVideoPart struct {
	Path string
	Name string
	Size int64
}

type mediaToolRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

var runMediaTool mediaToolRunner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", err, message)
	}
	return output, nil
}

func createLosslessVideoParts(
	ctx context.Context,
	r io.ReadSeeker,
	filename string,
	fileSize, maxPartSize int64,
) ([]losslessVideoPart, func(), error) {
	inputPath, sourceCleanup, err := sourceFile(r)
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to prepare seekable video source: %w", err)
	}

	tempBase := config.C().Temp.BasePath
	if err := os.MkdirAll(tempBase, 0o755); err != nil {
		sourceCleanup()
		return nil, func() {}, fmt.Errorf("failed to create video split base directory: %w", err)
	}
	tempDir, err := os.MkdirTemp(tempBase, "telegram-video-split-*")
	if err != nil {
		sourceCleanup()
		return nil, func() {}, fmt.Errorf("failed to create video split directory: %w", err)
	}
	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			log.FromContext(ctx).Warnf("Failed to clean lossless video parts: %s", err)
		}
		sourceCleanup()
	}

	parts, err := splitLosslessVideo(ctx, inputPath, tempDir, filename, fileSize, maxPartSize)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	return parts, cleanup, nil
}

func splitLosslessVideo(
	ctx context.Context,
	inputPath, outputDir, filename string,
	fileSize, maxPartSize int64,
) ([]losslessVideoPart, error) {
	if fileSize <= maxPartSize {
		return nil, fmt.Errorf("video size %d does not exceed part limit %d", fileSize, maxPartSize)
	}
	if maxPartSize <= 0 {
		return nil, fmt.Errorf("invalid video part limit: %d", maxPartSize)
	}

	duration, err := probeMediaDuration(ctx, inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to probe source video duration: %w", err)
	}
	if duration < minSegmentDuration {
		return nil, fmt.Errorf("invalid source video duration: %.3f", duration)
	}

	targetSize := int64(float64(maxPartSize) * videoPartTargetRatio)
	segmentDuration := initialSegmentDuration(duration, fileSize, targetSize)
	extension := videoPartExtension(filename)
	outputPattern := filepath.Join(outputDir, "part-%03d"+extension)

	var lastOversize int64
	for attempt := 0; attempt < videoSplitAttempts; attempt++ {
		if err := clearVideoParts(outputDir); err != nil {
			return nil, err
		}
		if err := runFFmpegSegment(ctx, inputPath, outputPattern, extension, segmentDuration); err != nil {
			return nil, fmt.Errorf("failed to losslessly split video: %w", err)
		}

		parts, largest, err := collectVideoParts(ctx, outputDir, filename)
		if err != nil {
			return nil, err
		}
		if len(parts) < 2 {
			return nil, fmt.Errorf("video split produced %d part(s), expected at least 2", len(parts))
		}
		if len(parts) > maxLosslessVideoParts {
			return nil, fmt.Errorf(
				"video split produced %d parts, exceeding the single-album limit of %d",
				len(parts),
				maxLosslessVideoParts,
			)
		}
		if largest <= maxPartSize {
			return parts, nil
		}

		lastOversize = largest
		segmentDuration *= float64(targetSize) / float64(largest)
		if segmentDuration < minSegmentDuration {
			break
		}
	}

	return nil, fmt.Errorf(
		"unable to keep lossless video parts below %d bytes; largest part was %d bytes",
		maxPartSize,
		lastOversize,
	)
}

func initialSegmentDuration(duration float64, fileSize, targetSize int64) float64 {
	partCount := math.Ceil(float64(fileSize) / float64(targetSize))
	if partCount < 2 {
		partCount = 2
	}
	segmentDuration := duration / partCount
	if segmentDuration < minSegmentDuration {
		return minSegmentDuration
	}
	return segmentDuration
}

func videoPartExtension(filename string) string {
	extension := strings.ToLower(filepath.Ext(filepath.Base(filename)))
	switch extension {
	case ".avi", ".m4v", ".mkv", ".mov", ".mp4", ".ts", ".webm":
		return extension
	default:
		return ".mp4"
	}
}

func runFFmpegSegment(
	ctx context.Context,
	inputPath, outputPattern, extension string,
	segmentDuration float64,
) error {
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-y",
		"-i", inputPath,
		"-map", "0",
		"-map_metadata", "0",
		"-c", "copy",
		"-f", "segment",
		"-segment_time", strconv.FormatFloat(segmentDuration, 'f', 3, 64),
		"-segment_start_number", "1",
		"-reset_timestamps", "1",
		"-avoid_negative_ts", "make_zero",
	}
	if extension == ".mp4" || extension == ".m4v" || extension == ".mov" {
		args = append(args, "-segment_format_options", "movflags=+faststart")
	}
	args = append(args, outputPattern)
	_, err := runMediaTool(ctx, "ffmpeg", args...)
	return err
}

func probeMediaDuration(ctx context.Context, filePath string) (float64, error) {
	output, err := runMediaTool(
		ctx,
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath,
	)
	if err != nil {
		return 0, err
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ffprobe duration %q: %w", strings.TrimSpace(string(output)), err)
	}
	return duration, nil
}

func clearVideoParts(outputDir string) error {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("failed to read video split directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(outputDir, entry.Name())); err != nil {
			return fmt.Errorf("failed to clear old video part %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func collectVideoParts(
	ctx context.Context,
	outputDir, filename string,
) ([]losslessVideoPart, int64, error) {
	matches, err := filepath.Glob(filepath.Join(outputDir, "part-*"))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list video parts: %w", err)
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return nil, 0, fmt.Errorf("ffmpeg did not produce any video parts")
	}

	extension := videoPartExtension(filename)
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filepath.Base(filename)))
	if base == "" || base == "." {
		base = xid.New().String()
	}
	parts := make([]losslessVideoPart, 0, len(matches))
	var largest int64
	for index, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to stat video part %s: %w", match, err)
		}
		if info.Size() <= 0 {
			return nil, 0, fmt.Errorf("video part %s is empty", match)
		}
		if _, err := probeMediaDuration(ctx, match); err != nil {
			return nil, 0, fmt.Errorf("failed to validate video part %s: %w", match, err)
		}
		if info.Size() > largest {
			largest = info.Size()
		}
		parts = append(parts, losslessVideoPart{
			Path: match,
			Name: fmt.Sprintf("%s.part%03d%s", base, index+1, extension),
			Size: info.Size(),
		})
	}
	return parts, largest, nil
}

func partStoragePath(storagePath, partName string) string {
	directory := path.Dir(path.Clean(storagePath))
	if directory == "." || directory == "/" {
		return partName
	}
	return path.Join(directory, partName)
}

func (t *Telegram) uploadLosslessVideoParts(
	ctx context.Context,
	tctx *ext.Context,
	storagePath string,
	parts []losslessVideoPart,
	sourceCaption *string,
) error {
	if len(parts) == 0 {
		return fmt.Errorf("no lossless video parts to upload")
	}
	if len(parts) > maxLosslessVideoParts {
		return fmt.Errorf(
			"refusing to upload %d lossless video parts as multiple albums; maximum is %d",
			len(parts),
			maxLosslessVideoParts,
		)
	}

	prepared := make([]preparedMedia, 0, len(parts))
	for index, part := range parts {
		partFile, err := os.Open(part.Path)
		if err != nil {
			return fmt.Errorf("failed to open video part %s: %w", part.Name, err)
		}
		media, prepareErr := t.prepareMedia(
			ctx,
			tctx,
			partFile,
			partStoragePath(storagePath, part.Name),
			part.Size,
			videoPartCaption(sourceCaption, index),
		)
		closeErr := partFile.Close()
		if prepareErr != nil {
			return fmt.Errorf("failed to prepare video part %s: %w", part.Name, prepareErr)
		}
		if closeErr != nil {
			return fmt.Errorf("failed to close video part %s: %w", part.Name, closeErr)
		}
		prepared = append(prepared, *media)
	}

	builder := tctx.Sender.WithUploader(prepared[0].uploader).To(prepared[0].peer)
	if len(prepared) == 1 {
		if _, err := builder.Media(ctx, prepared[0].media); err != nil {
			return fmt.Errorf("failed to send video part: %w", err)
		}
		return nil
	}
	media := make([]message.MultiMediaOption, len(prepared))
	for index := range prepared {
		media[index] = prepared[index].media
	}
	if _, err := builder.Album(ctx, media[0], media[1:]...); err != nil {
		return fmt.Errorf("failed to send video parts as album: %w", err)
	}
	return nil
}

func videoPartCaption(sourceCaption *string, index int) *string {
	if sourceCaption == nil {
		return nil
	}
	if index == 0 {
		return sourceCaption
	}
	empty := ""
	return &empty
}
