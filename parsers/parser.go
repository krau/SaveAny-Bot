package parsers

import (
	"context"
	"fmt"
	"sync"

	"github.com/krau/SaveAny-Bot/parsers/twitter"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

var (
	parsers   []parser.Parser
	parsersMu sync.Mutex
)

func GetParsers() []parser.Parser {
	parsersMu.Lock()
	defer parsersMu.Unlock()
	return parsers
}

func AddParser(p parser.Parser) {
	parsersMu.Lock()
	defer parsersMu.Unlock()
	parsers = append(parsers, p)
}

func init() {
	AddParser(new(twitter.TwitterParser))
}

var (
	ErrNoParserFound = fmt.Errorf("no parser found for the given URL")
)

func ParseWithContext(ctx context.Context, url string) (*parser.Item, error) {
	ch := make(chan *parser.Item, 1)
	errCh := make(chan error, 1)

	go func() {
		for _, pser := range parsers {
			if !pser.CanHandle(url) {
				continue
			}
			item, err := pser.Parse(url)
			if err != nil {
				errCh <- err
				return
			}
			ch <- item
			return
		}
		errCh <- ErrNoParserFound
	}()

	select {
	case item := <-ch:
		return item, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
