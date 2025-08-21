package twitter

import (
	"github.com/krau/SaveAny-Bot/common/op"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

type TwitterParser struct{}

func (p *TwitterParser) Parse(url string) (*parser.Item, error) {
	panic("TwitterParser.Parse not implemented")
}

func (p *TwitterParser) CanHandle(url string) bool {
	panic("TwitterParser.CanHandle not implemented")
}

func init() {
	op.RegisterParser(func() parser.Parser {
		return &TwitterParser{}
	})
}
