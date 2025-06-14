package rule

import (
	"regexp"

	ruleenum "github.com/krau/SaveAny-Bot/pkg/enums/rule"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

type RuleFileNameRegex struct {
	storInfo
	Regex *regexp.Regexp
}

var _ RuleClass[tfile.TGFile] = (*RuleFileNameRegex)(nil)

func (r RuleFileNameRegex) Type() ruleenum.RuleType {
	return ruleenum.FileNameRegex
}

func (r RuleFileNameRegex) Match(input tfile.TGFile) (bool, error) {
	return r.Regex.MatchString(input.Name()), nil
}

func (r RuleFileNameRegex) StorageName() string {
	return r.StorName
}

func (r RuleFileNameRegex) StoragePath() string {
	return r.StorPath
}

func NewRuleFileNameRegex(storName, storPath, regexStr string) (*RuleFileNameRegex, error) {
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, err
	}
	return &RuleFileNameRegex{
		storInfo: storInfo{
			StorName: storName,
			StorPath: storPath,
		},
		Regex: regex,
	}, nil
}
