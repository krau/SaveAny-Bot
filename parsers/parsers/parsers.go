package parsers

import (
	"slices"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

var (
	parsers []parser.Parser
	mu      sync.Mutex
)

func Add(p ...parser.Parser) {
	mu.Lock()
	defer mu.Unlock()
	parsers = append(parsers, p...)
}

// Get returns a snapshot of the registered parsers without reconfiguring them.
// Call Configure after changing parser settings or loading plugins.
func Get() []parser.Parser {
	mu.Lock()
	defer mu.Unlock()
	return slices.Clone(parsers)
}

// Configure applies the current parser settings to every configurable parser.
// It is safe to call at startup and after a settings reload.
func Configure() {
	mu.Lock()
	snapshot := slices.Clone(parsers)
	mu.Unlock()
	for _, pser := range snapshot {
		configurable, ok := pser.(parser.ConfigurableParser)
		if !ok {
			continue
		}
		cfg := config.C().GetParserConfigByName(configurable.Name())
		if err := configurable.Configure(cfg); err != nil {
			log.Errorf("Failed to configure parser %s: %v", configurable.Name(), err)
		}
	}
}
