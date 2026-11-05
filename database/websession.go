package database

import (
	"context"
	"time"
)

func CreateWebSession(ctx context.Context, session *WebSession) error {
	return db.WithContext(ctx).Create(session).Error
}

func GetWebSession(ctx context.Context, id string) (*WebSession, error) {
	var session WebSession
	if err := db.WithContext(ctx).Where("id = ? AND expires_at > ?", id, time.Now()).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func DeleteWebSession(ctx context.Context, id string) error {
	return db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&WebSession{}).Error
}

func DeleteOtherWebSessions(ctx context.Context, keepID string) error {
	return db.WithContext(ctx).Unscoped().Where("id <> ?", keepID).Delete(&WebSession{}).Error
}

func DeleteExpiredWebSessions(ctx context.Context) error {
	return db.WithContext(ctx).Unscoped().Where("expires_at <= ?", time.Now()).Delete(&WebSession{}).Error
}
