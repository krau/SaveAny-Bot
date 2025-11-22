package telegram

import (
	"fmt"
	"io"

	"github.com/yapingcat/gomedia/go-mp4"
)

type MP4Info struct {
	Duration int
	Width    int
	Height   int
}

func getMP4Info(r io.ReadSeeker) (*MP4Info, error) {
	d := mp4.CreateMp4Demuxer(r)

	tracks, err := d.ReadHead()
	if err != nil {
		return nil, err
	}

	for _, track := range tracks {
		if track.Cid == mp4.MP4_CODEC_H264 {
			info := d.GetMp4Info()
			return &MP4Info{
				Duration: int(info.Duration / info.Timescale),
				Width:    int(track.Width),
				Height:   int(track.Height),
			}, nil
		}
	}

	return nil, fmt.Errorf("no h264 track found")
}
