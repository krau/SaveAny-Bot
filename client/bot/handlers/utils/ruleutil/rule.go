package ruleutil

import (
	"context"

	"github.com/duke-git/lancet/v2/convertor"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/consts"
	ruleenum "github.com/krau/SaveAny-Bot/pkg/enums/rule"
	"github.com/krau/SaveAny-Bot/pkg/rule"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

type ruleInput struct {
	File tfile.TGFileMessage
}

type ruleInputOption func(*ruleInput)

func NewInput(file tfile.TGFileMessage, opts ...ruleInputOption) *ruleInput {
	input := &ruleInput{
		File: file,
	}
	for _, opt := range opts {
		opt(input)
	}
	return input
}

type matchedStorName string

func (m matchedStorName) String() string {
	return string(m)
}

// can we use this storage name directly?
func (m matchedStorName) IsUsable() bool {
	return m != "" && m != consts.RuleStorNameChosen
}

type MatchedDirPath string

func (m MatchedDirPath) String() string {
	return string(m)
}

func (m MatchedDirPath) NeedNewForAlbum() bool {
	return m != "" && m == consts.RuleDirPathNewForAlbum
}

func ApplyRule(ctx context.Context, rules []database.Rule, inputs *ruleInput) (matchedStorageName matchedStorName, dirPath MatchedDirPath) {
	if inputs == nil || len(rules) == 0 {
		return "", ""
	}
	logger := log.FromContext(ctx)
	for _, ur := range rules {
		switch ur.Type {
		case ruleenum.FileNameRegex.String():
			ru, err := rule.NewRuleFileNameRegex(ur.StorageName, ur.DirPath, ur.Data)
			if err != nil {
				logger.Errorf("Failed to create rule: %s", err)
				continue
			}
			ok, err := ru.Match(inputs.File)
			if err != nil {
				logger.Errorf("Failed to match rule: %s", err)
				continue
			}
			if ok {
				dirPath = MatchedDirPath(ru.StoragePath())
				matchedStorageName = matchedStorName(ru.StorageName())
			}
		case ruleenum.MessageRegex.String():
			ru, err := rule.NewRuleMessageRegex(ur.StorageName, ur.DirPath, ur.Data)
			if err != nil {
				logger.Errorf("Failed to create rule: %s", err)
				continue
			}
			ok, err := ru.Match(inputs.File.Message().GetMessage())
			if err != nil {
				logger.Errorf("Failed to match rule: %s", err)
				continue
			}
			if ok {
				dirPath = MatchedDirPath(ru.StoragePath())
				matchedStorageName = matchedStorName(ru.StorageName())
			}
		case ruleenum.IsAlbum.String():
			matchAlbum, err := convertor.ToBool(ur.Data)
			if err != nil {
				matchAlbum = false
			}
			ru, err := rule.NewRuleMediaType(ur.StorageName, ur.DirPath, matchAlbum)
			if err != nil {
				logger.Errorf("Failed to create rule: %s", err)
				continue
			}
			ok, err := ru.Match(inputs.File.Message().GroupedID != 0)
			if err != nil {
				logger.Errorf("Failed to match rule: %s", err)
				continue
			}
			if ok {
				dirPath = MatchedDirPath(ru.StoragePath())
				matchedStorageName = matchedStorName(ru.StorageName())
			}
		}
	}
	return
}
