package rule

import (
	"regexp"

	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

type RuleFileNameRegex struct {
	storInfo
	regex *regexp.Regexp
}

var _ RuleClass[tfile.TGFile] = (*RuleFileNameRegex)(nil)

func (r RuleFileNameRegex) Type() RuleType {
	return FileNameRegex
}

func (r RuleFileNameRegex) Match(input tfile.TGFile) (bool, error) {
	return r.regex.MatchString(input.Name()), nil
}

func (r RuleFileNameRegex) StorageName() string {
	return r.storName
}

func (r RuleFileNameRegex) StoragePath() string {
	return r.storPath
}

func NewRuleFileNameRegex(storName, storPath, regexStr string) (*RuleFileNameRegex, error) {
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, err
	}
	return &RuleFileNameRegex{
		storInfo: storInfo{
			storName: storName,
			storPath: storPath,
		},
		regex: regex,
	}, nil
}
