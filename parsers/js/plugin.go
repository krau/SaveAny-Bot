//go:build !no_jsparser

package js

import "github.com/blang/semver"

var (
	LatestParserVersion  = semver.MustParse("1.0.0")
	MinimumParserVersion = semver.MustParse("1.0.0")
)

type PluginMeta struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
}

type ParserMethod uint

const (
	_ ParserMethod = iota
	ParserMethodCanHandle
	ParserMethodParse
)
