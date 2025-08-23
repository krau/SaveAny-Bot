package rule

import (
	"regexp"
)

var _ RuleClass[string] = (*RuleMessageRegex)(nil)

type RuleMessageRegex struct {
	storInfo
	regex *regexp.Regexp
}

func (r RuleMessageRegex) Type() RuleType {
	return MessageRegex
}

func (r RuleMessageRegex) Match(input string) (bool, error) {
	return r.regex.MatchString(input), nil
}

func (r RuleMessageRegex) StorageName() string {
	return r.storName
}
func (r RuleMessageRegex) StoragePath() string {
	return r.storPath
}

func NewRuleMessageRegex(storName, storPath, regexStr string) (*RuleMessageRegex, error) {
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, err
	}
	return &RuleMessageRegex{
		storInfo: storInfo{
			storName: storName,
			storPath: storPath,
		},
		regex: regex,
	}, nil
}
