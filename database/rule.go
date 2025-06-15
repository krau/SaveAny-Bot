package database

import "context"

func CreateRule(ctx context.Context, rule *Rule) error {
	return db.WithContext(ctx).Create(rule).Error
}

func DeleteRule(ctx context.Context, ruleID uint) error {
	return db.WithContext(ctx).Unscoped().Delete(&Rule{}, ruleID).Error
}

func UpdateUserApplyRule(ctx context.Context, chatID int64, applyRule bool) error {
	return db.WithContext(ctx).Model(&User{}).Where("chat_id = ?", chatID).Update("apply_rule", applyRule).Error
}

func GetRulesByUserChatID(ctx context.Context, chatID int64) ([]Rule, error) {
	var rules []Rule
	err := db.WithContext(ctx).Where("user_id = (SELECT id FROM users WHERE chat_id = ?)", chatID).Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}
