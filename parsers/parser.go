package parsers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/parsers/js"
	"github.com/krau/SaveAny-Bot/parsers/native/kemono"
	"github.com/krau/SaveAny-Bot/parsers/native/twitter"
	"github.com/krau/SaveAny-Bot/parsers/parsers"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

func init() {
	parsers.Add(new(twitter.TwitterParser), new(kemono.KemonoParser))
}

var (
	ErrNoParserFound = fmt.Errorf("no parser found for the given URL")
)

// ParseWithContext tries every matching parser in registration order. A parser
// failure is returned only after all later matching parsers also fail.
func ParseWithContext(ctx context.Context, url string) (*parser.Item, error) {
	return parseWithContext(ctx, url, parsers.Get())
}

func parseWithContext(ctx context.Context, url string, candidates []parser.Parser) (*parser.Item, error) {
	ch := make(chan *parser.Item, 1)
	errCh := make(chan error, 1)

	go func() {
		var parseErrors []error
		for _, pser := range candidates {
			if err := ctx.Err(); err != nil {
				errCh <- err
				return
			}
			if !pser.CanHandle(url) {
				continue
			}
			item, err := pser.Parse(ctx, url)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					errCh <- ctxErr
					return
				}
				name := parserName(pser)
				wrapped := fmt.Errorf("parser %s: %w", name, err)
				log.FromContext(ctx).Debugf("Parser %s failed: %v", name, err)
				parseErrors = append(parseErrors, wrapped)
				continue
			}
			if item == nil {
				name := parserName(pser)
				err := fmt.Errorf("parser %s returned no item", name)
				log.FromContext(ctx).Debugf("%v", err)
				parseErrors = append(parseErrors, err)
				continue
			}
			if err := ctx.Err(); err != nil {
				errCh <- err
				return
			}
			ch <- item
			return
		}
		if len(parseErrors) > 0 {
			errCh <- errors.Join(parseErrors...)
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

func parserName(pser parser.Parser) string {
	if named, ok := pser.(parser.NamedParser); ok {
		if name := strings.TrimSpace(named.Name()); name != "" {
			return name
		}
	}
	return fmt.Sprintf("%T", pser)
}

// CanHandle checks if any registered parser can handle the given URL and returns the parser if found.
func CanHandle(url string) (bool, parser.Parser) {
	for _, pser := range parsers.Get() {
		if pser.CanHandle(url) {
			return true, pser
		}
	}
	return false, nil
}

// Configure reapplies parser settings after a settings or plugin reload.
func Configure() {
	parsers.Configure()
}

func LoadPlugins(ctx context.Context, dir string) error {
	return js.LoadPlugins(ctx, dir)
}

func AddPlugin(ctx context.Context, code string, name string) error {
	return js.AddPlugin(ctx, code, name)
}
