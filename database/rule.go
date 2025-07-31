package database

import "context"

// CreateRule creates a rule in the database (SQLite or Redis)
func CreateRule(ctx context.Context, rule *Rule) error {
	if useRedis {
		return redisCreateRule(ctx, rule)
	}
	
	// SQLite implementation (original)
	return db.WithContext(ctx).Create(rule).Error
}

// DeleteRule deletes a rule by ID from the database (SQLite or Redis)
func DeleteRule(ctx context.Context, ruleID uint) error {
	if useRedis {
		return redisDeleteRule(ctx, ruleID)
	}
	
	// SQLite implementation (original)
	return db.WithContext(ctx).Unscoped().Delete(&Rule{}, ruleID).Error
}

// UpdateUserApplyRule updates the apply_rule field for a user in the database (SQLite or Redis)
func UpdateUserApplyRule(ctx context.Context, chatID int64, applyRule bool) error {
	if useRedis {
		return redisUpdateUserApplyRule(ctx, chatID, applyRule)
	}
	
	// SQLite implementation (original)
	return db.WithContext(ctx).Model(&User{}).Where("chat_id = ?", chatID).Update("apply_rule", applyRule).Error
}

// GetRulesByUserChatID retrieves rules for a user by chat ID from the database (SQLite or Redis)
func GetRulesByUserChatID(ctx context.Context, chatID int64) ([]Rule, error) {
	if useRedis {
		return redisGetRulesByUserChatID(ctx, chatID)
	}
	
	// SQLite implementation (original)
	var rules []Rule
	err := db.WithContext(ctx).Where("user_id = (SELECT id FROM users WHERE chat_id = ?)", chatID).Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}
