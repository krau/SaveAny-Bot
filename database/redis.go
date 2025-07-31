package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Redis client instance
var rdb *redis.Client

// Redis key prefixes for different data types
const (
	userKeyPrefix       = "users:"        // users:{chatId}
	dirKeyPrefix        = "dirs:"         // dirs:{userId}:{dirId}
	ruleKeyPrefix       = "rules:"        // rules:{userId}:{ruleId}
	userDirsKeyPrefix   = "user_dirs:"    // user_dirs:{userId} -> set of dirIds
	userRulesKeyPrefix  = "user_rules:"   // user_rules:{userId} -> set of ruleIds
	counterKeyPrefix    = "counters:"     // counters:dir_id, counters:rule_id for auto-increment
)

// RedisUser represents a user stored in Redis (similar to User model)
type RedisUser struct {
	ID             uint      `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	ChatID         int64     `json:"chat_id"`
	Silent         bool      `json:"silent"`
	DefaultStorage string    `json:"default_storage"`
	ApplyRule      bool      `json:"apply_rule"`
}

// RedisDir represents a directory stored in Redis (similar to Dir model)
type RedisDir struct {
	ID          uint       `json:"id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	UserID      uint       `json:"user_id"`
	StorageName string     `json:"storage_name"`
	Path        string     `json:"path"`
}

// RedisRule represents a rule stored in Redis (similar to Rule model)
type RedisRule struct {
	ID          uint       `json:"id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	UserID      uint       `json:"user_id"`
	Type        string     `json:"type"`
	Data        string     `json:"data"`
	StorageName string     `json:"storage_name"`
	DirPath     string     `json:"dir_path"`
}

// initRedis initializes the Redis client connection
func initRedis(ctx context.Context) error {
	logger := log.FromContext(ctx)
	
	// Create Redis client options with configuration from config
	opts := &redis.Options{
		Addr:     config.Cfg.DB.RedisAddr,
		Password: config.Cfg.DB.RedisPassword,
		DB:       config.Cfg.DB.RedisDB,
	}
	
	// Set username for Redis ACL authentication if provided
	if config.Cfg.DB.RedisUser != "" {
		opts.Username = config.Cfg.DB.RedisUser
		logger.Debug("Redis ACL username configured", "username", config.Cfg.DB.RedisUser)
	}
	
	// Create Redis client with the configured options
	rdb = redis.NewClient(opts)

	// Test the connection
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	logger.Debug("Redis connected", "ping", pong)
	logger.Info("Redis database initialized")
	return nil
}

// generateID generates a new auto-increment ID for the given entity type
func generateID(ctx context.Context, entityType string) (uint, error) {
	counterKey := counterKeyPrefix + entityType
	id, err := rdb.Incr(ctx, counterKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to generate ID for %s: %w", entityType, err)
	}
	return uint(id), nil
}

// convertUserToRedisUser converts a GORM User to RedisUser
func convertUserToRedisUser(user *User) *RedisUser {
	var deletedAt *time.Time
	if user.DeletedAt.Valid {
		deletedAt = &user.DeletedAt.Time
	}
	
	return &RedisUser{
		ID:             user.ID,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
		DeletedAt:      deletedAt,
		ChatID:         user.ChatID,
		Silent:         user.Silent,
		DefaultStorage: user.DefaultStorage,
		ApplyRule:      user.ApplyRule,
	}
}

// convertRedisUserToUser converts a RedisUser to GORM User
func convertRedisUserToUser(redisUser *RedisUser) *User {
	var deletedAt gorm.DeletedAt
	if redisUser.DeletedAt != nil {
		deletedAt.Time = *redisUser.DeletedAt
		deletedAt.Valid = true
	}
	
	return &User{
		Model: gorm.Model{
			ID:        redisUser.ID,
			CreatedAt: redisUser.CreatedAt,
			UpdatedAt: redisUser.UpdatedAt,
			DeletedAt: deletedAt,
		},
		ChatID:         redisUser.ChatID,
		Silent:         redisUser.Silent,
		DefaultStorage: redisUser.DefaultStorage,
		ApplyRule:      redisUser.ApplyRule,
	}
}

// convertDirToRedisDir converts a GORM Dir to RedisDir
func convertDirToRedisDir(dir *Dir) *RedisDir {
	var deletedAt *time.Time
	if dir.DeletedAt.Valid {
		deletedAt = &dir.DeletedAt.Time
	}
	
	return &RedisDir{
		ID:          dir.ID,
		CreatedAt:   dir.CreatedAt,
		UpdatedAt:   dir.UpdatedAt,
		DeletedAt:   deletedAt,
		UserID:      dir.UserID,
		StorageName: dir.StorageName,
		Path:        dir.Path,
	}
}

// convertRedisDirToDir converts a RedisDir to GORM Dir
func convertRedisDirToDir(redisDir *RedisDir) *Dir {
	var deletedAt gorm.DeletedAt
	if redisDir.DeletedAt != nil {
		deletedAt.Time = *redisDir.DeletedAt
		deletedAt.Valid = true
	}
	
	return &Dir{
		Model: gorm.Model{
			ID:        redisDir.ID,
			CreatedAt: redisDir.CreatedAt,
			UpdatedAt: redisDir.UpdatedAt,
			DeletedAt: deletedAt,
		},
		UserID:      redisDir.UserID,
		StorageName: redisDir.StorageName,
		Path:        redisDir.Path,
	}
}

// convertRuleToRedisRule converts a GORM Rule to RedisRule
func convertRuleToRedisRule(rule *Rule) *RedisRule {
	var deletedAt *time.Time
	if rule.DeletedAt.Valid {
		deletedAt = &rule.DeletedAt.Time
	}
	
	return &RedisRule{
		ID:          rule.ID,
		CreatedAt:   rule.CreatedAt,
		UpdatedAt:   rule.UpdatedAt,
		DeletedAt:   deletedAt,
		UserID:      rule.UserID,
		Type:        rule.Type,
		Data:        rule.Data,
		StorageName: rule.StorageName,
		DirPath:     rule.DirPath,
	}
}

// convertRedisRuleToRule converts a RedisRule to GORM Rule
func convertRedisRuleToRule(redisRule *RedisRule) *Rule {
	var deletedAt gorm.DeletedAt
	if redisRule.DeletedAt != nil {
		deletedAt.Time = *redisRule.DeletedAt
		deletedAt.Valid = true
	}
	
	return &Rule{
		Model: gorm.Model{
			ID:        redisRule.ID,
			CreatedAt: redisRule.CreatedAt,
			UpdatedAt: redisRule.UpdatedAt,
			DeletedAt: deletedAt,
		},
		UserID:      redisRule.UserID,
		Type:        redisRule.Type,
		Data:        redisRule.Data,
		StorageName: redisRule.StorageName,
		DirPath:     redisRule.DirPath,
	}
}

// Redis implementations of user operations

// redisCreateUser creates a new user in Redis
func redisCreateUser(ctx context.Context, chatID int64) error {
	// Check if user already exists
	userKey := userKeyPrefix + strconv.FormatInt(chatID, 10)
	exists, err := rdb.Exists(ctx, userKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists > 0 {
		return nil // User already exists
	}

	// Generate new ID
	id, err := generateID(ctx, "user")
	if err != nil {
		return err
	}

	// Create user object
	now := time.Now()
	user := &RedisUser{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		ChatID:    chatID,
	}

	// Serialize to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}

	// Store in Redis
	err = rdb.Set(ctx, userKey, userData, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to store user in Redis: %w", err)
	}

	return nil
}

// redisGetAllUsers retrieves all users from Redis
func redisGetAllUsers(ctx context.Context) ([]User, error) {
	// Get all user keys
	userKeys, err := rdb.Keys(ctx, userKeyPrefix+"*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user keys: %w", err)
	}

	users := make([]User, 0, len(userKeys))
	for _, key := range userKeys {
		// Get user data
		userData, err := rdb.Get(ctx, key).Result()
		if err != nil {
			continue // Skip errors for individual users
		}

		var redisUser RedisUser
		if err := json.Unmarshal([]byte(userData), &redisUser); err != nil {
			continue // Skip invalid data
		}

		// Convert to GORM User and get associated data
		user := convertRedisUserToUser(&redisUser)
		
		// Load associated dirs and rules
		dirs, _ := redisGetUserDirs(ctx, user.ID)
		rules, _ := redisGetRulesByUserID(ctx, user.ID)
		
		user.Dirs = dirs
		user.Rules = rules
		
		users = append(users, *user)
	}

	return users, nil
}

// redisGetUserByChatID retrieves a user by chat ID from Redis
func redisGetUserByChatID(ctx context.Context, chatID int64) (*User, error) {
	userKey := userKeyPrefix + strconv.FormatInt(chatID, 10)
	
	userData, err := rdb.Get(ctx, userKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get user from Redis: %w", err)
	}

	var redisUser RedisUser
	if err := json.Unmarshal([]byte(userData), &redisUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	// Convert to GORM User
	user := convertRedisUserToUser(&redisUser)
	
	// Load associated dirs and rules
	dirs, _ := redisGetUserDirs(ctx, user.ID)
	rules, _ := redisGetRulesByUserID(ctx, user.ID)
	
	user.Dirs = dirs
	user.Rules = rules

	return user, nil
}

// redisUpdateUser updates a user in Redis
func redisUpdateUser(ctx context.Context, user *User) error {
	userKey := userKeyPrefix + strconv.FormatInt(user.ChatID, 10)
	
	// Check if user exists
	exists, err := rdb.Exists(ctx, userKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists == 0 {
		return gorm.ErrRecordNotFound
	}

	// Convert and update timestamp
	redisUser := convertUserToRedisUser(user)
	redisUser.UpdatedAt = time.Now()

	// Serialize to JSON
	userData, err := json.Marshal(redisUser)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}

	// Update in Redis
	err = rdb.Set(ctx, userKey, userData, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to update user in Redis: %w", err)
	}

	return nil
}

// redisDeleteUser deletes a user and associated data from Redis
func redisDeleteUser(ctx context.Context, user *User) error {
	userKey := userKeyPrefix + strconv.FormatInt(user.ChatID, 10)
	
	// Delete user data
	err := rdb.Del(ctx, userKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete user from Redis: %w", err)
	}

	// Delete associated dirs and rules
	userDirsKey := userDirsKeyPrefix + strconv.Itoa(int(user.ID))
	userRulesKey := userRulesKeyPrefix + strconv.Itoa(int(user.ID))
	
	// Get all dir and rule IDs
	dirIDs, _ := rdb.SMembers(ctx, userDirsKey).Result()
	ruleIDs, _ := rdb.SMembers(ctx, userRulesKey).Result()
	
	// Delete individual dirs and rules
	for _, dirID := range dirIDs {
		dirKey := dirKeyPrefix + strconv.Itoa(int(user.ID)) + ":" + dirID
		rdb.Del(ctx, dirKey)
	}
	for _, ruleID := range ruleIDs {
		ruleKey := ruleKeyPrefix + strconv.Itoa(int(user.ID)) + ":" + ruleID
		rdb.Del(ctx, ruleKey)
	}
	
	// Delete index sets
	rdb.Del(ctx, userDirsKey)
	rdb.Del(ctx, userRulesKey)

	return nil
}

// Redis implementations of dir operations

func redisGetUserDirs(ctx context.Context, userID uint) ([]Dir, error) {
	userDirsKey := userDirsKeyPrefix + strconv.Itoa(int(userID))
	
	dirIDs, err := rdb.SMembers(ctx, userDirsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user dirs: %w", err)
	}

	dirs := make([]Dir, 0, len(dirIDs))
	for _, dirID := range dirIDs {
		dirKey := dirKeyPrefix + strconv.Itoa(int(userID)) + ":" + dirID
		
		dirData, err := rdb.Get(ctx, dirKey).Result()
		if err != nil {
			continue // Skip errors
		}

		var redisDir RedisDir
		if err := json.Unmarshal([]byte(dirData), &redisDir); err != nil {
			continue
		}

		dirs = append(dirs, *convertRedisDirToDir(&redisDir))
	}

	return dirs, nil
}

// Redis implementations of rule operations

func redisGetRulesByUserID(ctx context.Context, userID uint) ([]Rule, error) {
	userRulesKey := userRulesKeyPrefix + strconv.Itoa(int(userID))
	
	ruleIDs, err := rdb.SMembers(ctx, userRulesKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user rules: %w", err)
	}

	rules := make([]Rule, 0, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		ruleKey := ruleKeyPrefix + strconv.Itoa(int(userID)) + ":" + ruleID
		
		ruleData, err := rdb.Get(ctx, ruleKey).Result()
		if err != nil {
			continue // Skip errors
		}

		var redisRule RedisRule
		if err := json.Unmarshal([]byte(ruleData), &redisRule); err != nil {
			continue
		}

		rules = append(rules, *convertRedisRuleToRule(&redisRule))
	}

	return rules, nil
}

// redisCreateDirForUser creates a directory for a user in Redis
func redisCreateDirForUser(ctx context.Context, userID uint, storageName, path string) error {
	// Generate new ID
	id, err := generateID(ctx, "dir")
	if err != nil {
		return err
	}

	// Create dir object
	now := time.Now()
	dir := &RedisDir{
		ID:          id,
		CreatedAt:   now,
		UpdatedAt:   now,
		UserID:      userID,
		StorageName: storageName,
		Path:        path,
	}

	// Serialize to JSON
	dirData, err := json.Marshal(dir)
	if err != nil {
		return fmt.Errorf("failed to marshal dir data: %w", err)
	}

	// Store in Redis
	dirKey := dirKeyPrefix + strconv.Itoa(int(userID)) + ":" + strconv.Itoa(int(id))
	err = rdb.Set(ctx, dirKey, dirData, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to store dir in Redis: %w", err)
	}

	// Add to user's dirs index
	userDirsKey := userDirsKeyPrefix + strconv.Itoa(int(userID))
	err = rdb.SAdd(ctx, userDirsKey, strconv.Itoa(int(id))).Err()
	if err != nil {
		return fmt.Errorf("failed to add dir to user index: %w", err)
	}

	return nil
}

// redisGetDirByID retrieves a directory by ID from Redis
func redisGetDirByID(ctx context.Context, userID, id uint) (*Dir, error) {
	dirKey := dirKeyPrefix + strconv.Itoa(int(userID)) + ":" + strconv.Itoa(int(id))
	
	dirData, err := rdb.Get(ctx, dirKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get dir from Redis: %w", err)
	}

	var redisDir RedisDir
	if err := json.Unmarshal([]byte(dirData), &redisDir); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dir data: %w", err)
	}

	return convertRedisDirToDir(&redisDir), nil
}

// redisGetUserDirsByChatID retrieves directories for a user by chat ID from Redis
func redisGetUserDirsByChatID(ctx context.Context, chatID int64) ([]Dir, error) {
	user, err := redisGetUserByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return redisGetUserDirs(ctx, user.ID)
}

// redisGetDirsByUserIDAndStorageName retrieves directories by user ID and storage name from Redis
func redisGetDirsByUserIDAndStorageName(ctx context.Context, userID uint, storageName string) ([]Dir, error) {
	dirs, err := redisGetUserDirs(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Filter by storage name
	filteredDirs := make([]Dir, 0)
	for _, dir := range dirs {
		if dir.StorageName == storageName {
			filteredDirs = append(filteredDirs, dir)
		}
	}

	return filteredDirs, nil
}

// redisGetDirsByUserChatIDAndStorageName retrieves directories by user chat ID and storage name from Redis
func redisGetDirsByUserChatIDAndStorageName(ctx context.Context, chatID int64, storageName string) ([]Dir, error) {
	user, err := redisGetUserByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return redisGetDirsByUserIDAndStorageName(ctx, user.ID, storageName)
}

// redisDeleteDirForUser deletes a directory for a user from Redis
func redisDeleteDirForUser(ctx context.Context, userID uint, storageName, path string) error {
	// Find the dir by userID, storageName, and path
	dirs, err := redisGetUserDirs(ctx, userID)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		if dir.UserID == userID && dir.StorageName == storageName && dir.Path == path {
			return redisDeleteDirByID(ctx, dir.ID)
		}
	}

	return nil // Not found, but don't return error
}

// redisDeleteDirByID deletes a directory by ID from Redis
func redisDeleteDirByID(ctx context.Context, id uint) error {
	// Find the dir to get userID
	userDirsKeys, err := rdb.Keys(ctx, userDirsKeyPrefix+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to get user dirs keys: %w", err)
	}

	var userID uint
	idStr := strconv.Itoa(int(id))
	
	// Find which user this dir belongs to
	for _, userDirsKey := range userDirsKeys {
		isMember, err := rdb.SIsMember(ctx, userDirsKey, idStr).Result()
		if err != nil {
			continue
		}
		if isMember {
			// Extract userID from key
			userIDStr := userDirsKey[len(userDirsKeyPrefix):]
			if uid, err := strconv.Atoi(userIDStr); err == nil {
				userID = uint(uid)
				break
			}
		}
	}

	if userID == 0 {
		return gorm.ErrRecordNotFound
	}

	// Delete the dir
	dirKey := dirKeyPrefix + strconv.Itoa(int(userID)) + ":" + idStr
	err = rdb.Del(ctx, dirKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete dir from Redis: %w", err)
	}

	// Remove from user's dirs index
	userDirsKey := userDirsKeyPrefix + strconv.Itoa(int(userID))
	err = rdb.SRem(ctx, userDirsKey, idStr).Err()
	if err != nil {
		return fmt.Errorf("failed to remove dir from user index: %w", err)
	}

	return nil
}

// redisCreateRule creates a rule in Redis
func redisCreateRule(ctx context.Context, rule *Rule) error {
	// Generate new ID if not set
	var id uint
	var err error
	if rule.ID == 0 {
		id, err = generateID(ctx, "rule")
		if err != nil {
			return err
		}
	} else {
		id = rule.ID
	}

	// Create rule object
	now := time.Now()
	redisRule := &RedisRule{
		ID:          id,
		CreatedAt:   now,
		UpdatedAt:   now,
		UserID:      rule.UserID,
		Type:        rule.Type,
		Data:        rule.Data,
		StorageName: rule.StorageName,
		DirPath:     rule.DirPath,
	}

	// Serialize to JSON
	ruleData, err := json.Marshal(redisRule)
	if err != nil {
		return fmt.Errorf("failed to marshal rule data: %w", err)
	}

	// Store in Redis
	ruleKey := ruleKeyPrefix + strconv.Itoa(int(rule.UserID)) + ":" + strconv.Itoa(int(id))
	err = rdb.Set(ctx, ruleKey, ruleData, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to store rule in Redis: %w", err)
	}

	// Add to user's rules index
	userRulesKey := userRulesKeyPrefix + strconv.Itoa(int(rule.UserID))
	err = rdb.SAdd(ctx, userRulesKey, strconv.Itoa(int(id))).Err()
	if err != nil {
		return fmt.Errorf("failed to add rule to user index: %w", err)
	}

	return nil
}

// redisDeleteRule deletes a rule by ID from Redis
func redisDeleteRule(ctx context.Context, ruleID uint) error {
	// Find the rule to get userID
	userRulesKeys, err := rdb.Keys(ctx, userRulesKeyPrefix+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to get user rules keys: %w", err)
	}

	var userID uint
	idStr := strconv.Itoa(int(ruleID))
	
	// Find which user this rule belongs to
	for _, userRulesKey := range userRulesKeys {
		isMember, err := rdb.SIsMember(ctx, userRulesKey, idStr).Result()
		if err != nil {
			continue
		}
		if isMember {
			// Extract userID from key
			userIDStr := userRulesKey[len(userRulesKeyPrefix):]
			if uid, err := strconv.Atoi(userIDStr); err == nil {
				userID = uint(uid)
				break
			}
		}
	}

	if userID == 0 {
		return gorm.ErrRecordNotFound
	}

	// Delete the rule
	ruleKey := ruleKeyPrefix + strconv.Itoa(int(userID)) + ":" + idStr
	err = rdb.Del(ctx, ruleKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete rule from Redis: %w", err)
	}

	// Remove from user's rules index
	userRulesKey := userRulesKeyPrefix + strconv.Itoa(int(userID))
	err = rdb.SRem(ctx, userRulesKey, idStr).Err()
	if err != nil {
		return fmt.Errorf("failed to remove rule from user index: %w", err)
	}

	return nil
}

// redisUpdateUserApplyRule updates the apply_rule field for a user in Redis
func redisUpdateUserApplyRule(ctx context.Context, chatID int64, applyRule bool) error {
	user, err := redisGetUserByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	user.ApplyRule = applyRule
	return redisUpdateUser(ctx, user)
}

// redisGetRulesByUserChatID retrieves rules for a user by chat ID from Redis
func redisGetRulesByUserChatID(ctx context.Context, chatID int64) ([]Rule, error) {
	user, err := redisGetUserByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return redisGetRulesByUserID(ctx, user.ID)
}