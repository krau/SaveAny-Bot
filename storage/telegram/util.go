package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/constant"
	"github.com/gotd/td/tg"
	"github.com/krau/ffmpeg-go"
	"github.com/yapingcat/gomedia/go-mp4"
)

type VideoMetadata struct {
	Duration int
	Width    int
	Height   int
}

// a go native way to get mp4 video metadata
func getMP4Meta(rs io.ReadSeeker) (metadata *VideoMetadata, err error) {
	// Recover from panics in the gomedia library (e.g., "no vosdata" panic)
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while parsing MP4: %v", r)
		}
	}()

	d := mp4.CreateMp4Demuxer(rs)

	tracks, e := d.ReadHead()
	if e != nil {
		return nil, e
	}

	for _, track := range tracks {
		if track.Cid == mp4.MP4_CODEC_H264 {
			info := d.GetMp4Info()
			return &VideoMetadata{
				Duration: int(info.Duration / info.Timescale),
				Width:    int(track.Width),
				Height:   int(track.Height),
			}, nil
		}
	}

	return nil, fmt.Errorf("no h264 track found")
}

// getVideoMetadata uses ffprobe to get video metadata
func getVideoMetadata(rs io.ReadSeeker) (*VideoMetadata, error) {
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		defer pipeWriter.Close()
		rs.Seek(0, io.SeekStart)
		io.Copy(pipeWriter, rs)
	}()

	result, err := ffmpeg.ProbeReaderWithTimeout(
		pipeReader,
		time.Second*10,
		ffmpeg.KwArgs{
			"select_streams": "v:0",
			"show_entries":   "stream=width,height:format=duration",
			"of":             "json",
		},
	)
	if err != nil {
		return nil, err
	}

	var data struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return nil, err
	}

	// 转换 duration
	var durationFloat float64
	if data.Format.Duration != "" {
		fmt.Sscanf(data.Format.Duration, "%f", &durationFloat)
	}

	meta := &VideoMetadata{
		Duration: int(durationFloat),
	}

	if len(data.Streams) > 0 {
		meta.Width = data.Streams[0].Width
		meta.Height = data.Streams[0].Height
	}

	return meta, nil
}

func extractThumbFrame(rs io.ReadSeeker) ([]byte, error) {
	data, err := extractFrameAt(rs, 1.0)
	if err == nil && len(data) > 0 {
		return data, nil
	}
	return extractFrameAt(rs, 0.0)
}

func extractFrameAt(rs io.ReadSeeker, timestamp float64) ([]byte, error) {
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		defer pipeWriter.Close()
		rs.Seek(0, io.SeekStart)
		io.Copy(pipeWriter, rs)
	}()

	var out bytes.Buffer

	err := ffmpeg.
		Input("pipe:0", ffmpeg.KwArgs{
			"ss": fmt.Sprintf("%.3f", timestamp),
		}).
		Output("pipe:1", ffmpeg.KwArgs{
			"vframes": 1,
			"f":       "mjpeg",
		}).
		WithInput(pipeReader).
		WithOutput(&out).
		OverWriteOutput().
		Run()

	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func tryGetInputPeer(ctx *ext.Context, chatID int64) tg.InputPeerClass {
	peer := ctx.PeerStorage.GetInputPeerById(chatID)
	if peer != nil && !peer.Zero() {
		return peer
	}
	id := constant.TDLibPeerID(chatID)
	plain := id.ToPlain()
	var channel constant.TDLibPeerID
	channel.Channel(plain)
	peer = ctx.PeerStorage.GetInputPeerById(int64(channel))
	if peer != nil && !peer.Zero() {
		return peer
	}
	var chat constant.TDLibPeerID
	chat.Chat(plain)
	peer = ctx.PeerStorage.GetInputPeerById(int64(chat))
	if peer != nil && !peer.Zero() {
		return peer
	}
	var user constant.TDLibPeerID
	user.User(plain)
	peer = ctx.PeerStorage.GetInputPeerById(int64(user))
	return peer
}
