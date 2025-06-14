package rule

import (
	ruleenum "github.com/krau/SaveAny-Bot/pkg/enums/rule"
)

type RuleClass[InputType any] interface {
	Type() ruleenum.RuleType
	Match(input InputType) (bool, error)
	StorageName() string
	StoragePath() string
}

type storInfo struct {
	StorName string
	StorPath string
}
