package core

import (
	"fmt"
	"path"
	"regexp"

	"github.com/celestix/gotgproto/ext"
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func getStorageAndPathForTask(task *types.Task) (storage.Storage, string, error) {
	user, err := dao.GetUserByChatID(task.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user by chat ID: %w", err)
	}
	if task.StoragePath == "" {
		task.StoragePath = task.FileName()
	}
	taskStorage, err := storage.GetStorageByUserIDAndName(task.UserID, task.StorageName)
	if err != nil {
		return nil, "", err
	}
	storagePath := taskStorage.JoinStoragePath(*task)
	if !user.ApplyRule || user.Rules == nil {
		return taskStorage, storagePath, nil
	}
	var ruleTaskStorage storage.Storage
	var ruleStoragePath string
	for _, rule := range user.Rules {
		matchStorage, matchStoragePath := applyRule(&rule, *task)
		if matchStorage != nil && matchStoragePath != "" {
			ruleTaskStorage = matchStorage
			ruleStoragePath = matchStoragePath
		}
	}
	if ruleStoragePath == "" || ruleTaskStorage == nil {
		return taskStorage, storagePath, nil
	}
	common.Log.Debugf("Rule matched: %s, %s", ruleTaskStorage.Name(), ruleStoragePath)
	return ruleTaskStorage, ruleStoragePath, nil
}

func applyRule(rule *dao.Rule, task types.Task) (storage.Storage, string) {
	var DirPath, StorageName string
	switch rule.Type {
	case string(types.RuleTypeFileNameRegex):
		ruleRegex, err := regexp.Compile(rule.Data)
		if err != nil {
			common.Log.Errorf("failed to compile regex: %s", err)
			return nil, ""
		}
		if !ruleRegex.MatchString(task.FileName()) {
			return nil, ""
		}
		DirPath = rule.DirPath
		StorageName = rule.StorageName
	case string(types.RuleTypeMessageRegex):
		ruleRegex, err := regexp.Compile(rule.Data)
		if err != nil {
			common.Log.Errorf("failed to compile regex: %s", err)
			return nil, ""
		}
		ctx, ok := task.Ctx.(*ext.Context)
		if !ok {
			common.Log.Fatalf("context is not *ext.Context: %T", task.Ctx)
			return nil, ""
		}
		msg, err := bot.GetTGMessage(ctx, task.FileChatID, task.FileMessageID)
		if err != nil {
			common.Log.Errorf("failed to get message: %s", err)
			return nil, ""
		}
		if msg == nil {
			return nil, ""
		}
		if !ruleRegex.MatchString(msg.GetMessage()) {
			return nil, ""
		}
		DirPath = rule.DirPath
		StorageName = rule.StorageName
	default:
		common.Log.Errorf("unknown rule type: %s", rule.Type)
		return nil, ""
	}
	taskStorageName := func() string {
		if StorageName == "" || StorageName == "CHOSEN" {
			return task.StorageName
		}
		return StorageName
	}()
	taskStorage, err := storage.GetStorageByUserIDAndName(task.UserID, taskStorageName)
	if err != nil {
		common.Log.Errorf("failed to get storage: %s", err)
		return nil, ""
	}
	task.StoragePath = path.Join(DirPath, task.StoragePath)
	return taskStorage, taskStorage.JoinStoragePath(task)
}
