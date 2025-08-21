package parser

import (
	"context"
	"fmt"
)

type Parser interface {
	CanHandle(url string) bool
	Parse(url string) (*Item, error)
}

var parsers []Parser

func GetParsers() []Parser {
	return parsers
}

var (
	ErrNoParserFound = fmt.Errorf("no parser found for the given URL")
)

func ParseWithContext(ctx context.Context, url string) (*Item, error) {
	ch := make(chan *Item, 1)
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
