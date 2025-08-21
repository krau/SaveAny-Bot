package op

import (
	"github.com/krau/SaveAny-Bot/parsers"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

type ParserConstructor func() parser.Parser

func RegisterParser(pser ParserConstructor) {
	p := pser()
	parsers.AddParser(p)
}
