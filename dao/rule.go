package dao

func CreateRule(rule *Rule) error {
	return db.Create(rule).Error
}

func DeleteRule(ruleID uint) error {
	return db.Unscoped().Delete(&Rule{}, ruleID).Error
}

func UpdateUserApplyRule(chatID int64, applyRule bool) error {
	return db.Model(&User{}).Where("chat_id = ?", chatID).Update("apply_rule", applyRule).Error
}

func GetRulesByUserChatID(chatID int64) ([]Rule, error) {
	var rules []Rule
	err := db.Where("user_id = (SELECT id FROM users WHERE chat_id = ?)", chatID).Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}
