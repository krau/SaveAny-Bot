package config

type YtdlpConfig struct {
	// MaxHeight limits the video resolution by height in pixels (e.g. 1080, 720).
	// 0 means no limit (best available). Ignored when Format is set.
	MaxHeight int `toml:"max_height" mapstructure:"max_height" json:"max_height"`
	// Format is a raw yt-dlp format selector (-f). When set, it takes precedence
	// over MaxHeight and gives the user full control.
	Format string `toml:"format" mapstructure:"format" json:"format"`
	// Recode is the target video container yt-dlp recodes into (e.g. mp4).
	// Empty disables recoding.
	Recode string `toml:"recode" mapstructure:"recode" json:"recode"`
}
