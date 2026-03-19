package services

import (
	"context"
	"fmt"
	"time"

	db "github.com/bufferwise/levelr/internal/db/sqlc"
)

type BlacklistCache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

type BlacklistService struct {
	queries *db.Queries
	cache   BlacklistCache
}

func NewBlacklistService(queries *db.Queries, cache BlacklistCache) *BlacklistService {
	return &BlacklistService{queries: queries, cache: cache}
}

// AddBlacklist adds an entity to the blacklist and updates cache.
func (s *BlacklistService) AddBlacklist(ctx context.Context, guildID string, entityType string, id uint64) error {
	err := s.queries.AddBlacklist(ctx, db.AddBlacklistParams{
		TargetId:   int64(id),
		EntityType: entityType,
	})
	if err == nil {
		key := fmt.Sprintf("blacklist:%s:%d:%s", entityType, id, guildID)
		_ = s.cache.Set(ctx, key, "1", 5*time.Minute)
	}
	return err
}

// RemoveBlacklist removes an entity from the blacklist and updates cache.
func (s *BlacklistService) RemoveBlacklist(ctx context.Context, guildID string, id uint64) error {
	err := s.queries.RemoveBlacklist(ctx, int64(id))
	if err == nil {
		// Invalidate all potential cache keys
		_ = s.cache.Set(ctx, fmt.Sprintf("blacklist:user:%d:%s", id, guildID), "0", 1*time.Second)
		_ = s.cache.Set(ctx, fmt.Sprintf("blacklist:role:%d:%s", id, guildID), "0", 1*time.Second)
		_ = s.cache.Set(ctx, fmt.Sprintf("blacklist:channel:%d:%s", id, guildID), "0", 1*time.Second)
	}
	return err
}

// IsEntityBlacklisted checks if an entity (user, role, or channel) is blacklisted.
func (s *BlacklistService) IsEntityBlacklisted(ctx context.Context, entityType string, id uint64, guildID string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s:%d:%s", entityType, id, guildID)

	// Check cache
	if val, ok := s.cache.Get(ctx, key); ok {
		return val == "1", nil
	}

	// Check DB
	res, err := s.queries.IsBlacklisted(ctx, db.IsBlacklistedParams{
		EntityType: entityType,
		EntityId:   int64(id),
	})
	if err != nil {
		return false, err
	}

	isBlacklisted := res != 0

	// Update cache
	val := "0"
	if isBlacklisted {
		val = "1"
	}
	_ = s.cache.Set(ctx, key, val, 5*time.Minute)

	return isBlacklisted, nil
}

// IsUserBlacklisted checks user, channel, and all user roles for blacklist status.
func (s *BlacklistService) IsUserBlacklisted(ctx context.Context, userID, channelID uint64, roleIDs []uint64, guildID string) (bool, error) {
	// 1. Check user
	if blacklisted, _ := s.IsEntityBlacklisted(ctx, "user", userID, guildID); blacklisted {
		return true, nil
	}

	// 2. Check channel
	if blacklisted, _ := s.IsEntityBlacklisted(ctx, "channel", channelID, guildID); blacklisted {
		return true, nil
	}

	// 3. Check roles
	for _, rid := range roleIDs {
		if blacklisted, _ := s.IsEntityBlacklisted(ctx, "role", rid, guildID); blacklisted {
			return true, nil
		}
	}

	return false, nil
}
