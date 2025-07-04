package rule

import (
	ruleenum "github.com/krau/SaveAny-Bot/pkg/enums/rule"
)

var _ RuleClass[bool] = (*RuleMediaType)(nil)

type RuleMediaType struct {
	storInfo
	matchAlbum bool
}

func (r RuleMediaType) Type() ruleenum.RuleType {
	return ruleenum.IsAlbum
}

func (r RuleMediaType) Match(input bool) (bool, error) {
	return r.matchAlbum == input, nil
}

func (r RuleMediaType) StorageName() string {
	return r.storName
}

func (r RuleMediaType) StoragePath() string {
	return r.storPath
}

func NewRuleMediaType(storName, storPath string, matchAlbum bool) (*RuleMediaType, error) {
	return &RuleMediaType{
		storInfo: storInfo{
			storName: storName,
			storPath: storPath,
		},
		matchAlbum: matchAlbum,
	}, nil
}
