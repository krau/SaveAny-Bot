package parsers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/krau/SaveAny-Bot/pkg/parser"
)

type stubParser struct {
	name    string
	handles bool
	item    *parser.Item
	err     error
	calls   *int
	cancel  context.CancelFunc
}

func (p *stubParser) CanHandle(string) bool { return p.handles }

func (p *stubParser) Name() string { return p.name }

func (p *stubParser) Parse(context.Context, string) (*parser.Item, error) {
	if p.calls != nil {
		(*p.calls)++
	}
	if p.cancel != nil {
		p.cancel()
	}
	return p.item, p.err
}

func TestParseWithContextFallsBackToNextParser(t *testing.T) {
	want := &parser.Item{Title: "from second parser"}
	var firstCalls, secondCalls int
	got, err := parseWithContext(t.Context(), "https://example.com/item", []parser.Parser{
		&stubParser{
			name:    "first",
			handles: true,
			err:     errors.New("first failed"),
			calls:   &firstCalls,
		},
		&stubParser{
			name:    "second",
			handles: true,
			item:    want,
			calls:   &secondCalls,
		},
	})
	if err != nil {
		t.Fatalf("parseWithContext: %v", err)
	}
	if got != want {
		t.Fatalf("item = %#v, want %#v", got, want)
	}
	if firstCalls != 1 || secondCalls != 1 {
		t.Fatalf("calls = first:%d second:%d, want 1 each", firstCalls, secondCalls)
	}
}

func TestParseWithContextReportsAllParserFailures(t *testing.T) {
	firstErr := errors.New("first failed")
	secondErr := errors.New("second failed")
	_, err := parseWithContext(t.Context(), "https://example.com/item", []parser.Parser{
		&stubParser{name: "first", handles: true, err: firstErr},
		&stubParser{name: "second", handles: true, err: secondErr},
	})
	if err == nil {
		t.Fatal("parseWithContext returned no error")
	}
	for _, want := range []error{firstErr, secondErr} {
		if !errors.Is(err, want) {
			t.Fatalf("error %v does not wrap %v", err, want)
		}
	}
	for _, name := range []string{"first", "second"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("error %q does not name parser %q", err, name)
		}
	}
}

func TestParseWithContextTreatsNilItemAsFailure(t *testing.T) {
	want := &parser.Item{Title: "from second parser"}
	got, err := parseWithContext(t.Context(), "https://example.com/item", []parser.Parser{
		&stubParser{name: "empty", handles: true},
		&stubParser{name: "second", handles: true, item: want},
	})
	if err != nil {
		t.Fatalf("parseWithContext: %v", err)
	}
	if got != want {
		t.Fatalf("item = %#v, want %#v", got, want)
	}
}

func TestParseWithContextStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	var secondCalls int
	_, err := parseWithContext(ctx, "https://example.com/item", []parser.Parser{
		&stubParser{
			name:    "canceling",
			handles: true,
			err:     context.Canceled,
			cancel:  cancel,
		},
		&stubParser{
			name:    "second",
			handles: true,
			item:    &parser.Item{Title: "must not run"},
			calls:   &secondCalls,
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if secondCalls != 0 {
		t.Fatalf("second parser calls = %d, want 0", secondCalls)
	}
}

func TestParseWithContextNoMatchingParser(t *testing.T) {
	_, err := parseWithContext(t.Context(), "https://example.com/item", []parser.Parser{
		&stubParser{name: "ignored"},
	})
	if !errors.Is(err, ErrNoParserFound) {
		t.Fatalf("error = %v, want ErrNoParserFound", err)
	}
}
