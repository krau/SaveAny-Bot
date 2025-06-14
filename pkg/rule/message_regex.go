package rule

import (
	"regexp"

	ruleenum "github.com/krau/SaveAny-Bot/pkg/enums/rule"
)

var _ RuleClass[string] = (*RuleMessageRegex)(nil)

type RuleMessageRegex struct {
	storInfo
	Regex *regexp.Regexp
}

func (r RuleMessageRegex) Type() ruleenum.RuleType {
	return ruleenum.MessageRegex
}

func (r RuleMessageRegex) Match(input string) (bool, error) {
	return r.Regex.MatchString(input), nil
}

func (r RuleMessageRegex) StorageName() string {
	return r.StorName
}
func (r RuleMessageRegex) StoragePath() string {
	return r.StorPath
}

func NewRuleMessageRegex(storName, storPath, regexStr string) (*RuleMessageRegex, error) {
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, err
	}
	return &RuleMessageRegex{
		storInfo: storInfo{
			StorName: storName,
			StorPath: storPath,
		},
		Regex: regex,
	}, nil
}
