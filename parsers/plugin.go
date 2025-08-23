package parsers

import "github.com/blang/semver"

var (
	LatestParserVersion  = semver.MustParse("1.0.0")
	MinimumParserVersion = semver.MustParse("1.0.0")
)

type PluginMeta struct {
	Name        string `json:"name"`
	Version     string `json:"version"` // [TODO] 分版本解析, 但是我们现在只有 v1 所以先不写
	Description string `json:"description"`
	Author      string `json:"author"`
}

type ParserMethod uint

const (
	_ ParserMethod = iota
	ParserMethodCanHandle
	ParserMethodParse
)
