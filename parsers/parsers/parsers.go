package parsers

import (
	"fmt"
	"sync"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

var (
	parsers       []parser.Parser
	mu            sync.Mutex
	configOnce    sync.Once
	configParsers = func() {
		mu.Lock()
		defer mu.Unlock()
		if len(parsers) == 0 {
			return
		}
		for _, pser := range parsers {
			if configurable, ok := pser.(parser.ConfigurableParser); ok {
				cfg := config.C().GetParserConfigByName(configurable.Name())
				if err := configurable.Configure(cfg); err != nil {
					fmt.Printf("Error configuring parser %s: %v\n", configurable.Name(), err)
				}
			}
		}
	}
)

func Add(p ...parser.Parser) {
	mu.Lock()
	defer mu.Unlock()
	parsers = append(parsers, p...)
}

func Get() []parser.Parser {
	configOnce.Do(configParsers)
	mu.Lock()
	defer mu.Unlock()
	return parsers
}
