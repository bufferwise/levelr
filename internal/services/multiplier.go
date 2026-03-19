package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	db "github.com/bufferwise/levelr/internal/db/sqlc"
)

type MultiplierCache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

type MultiplierService struct {
	queries *db.Queries
	cache   MultiplierCache
}

func NewMultiplierService(queries *db.Queries, cache MultiplierCache) *MultiplierService {
	return &MultiplierService{queries: queries, cache: cache}
}

// GetEntityMultiplier returns the multiplier for a single entity (role or channel).
func (s *MultiplierService) GetEntityMultiplier(ctx context.Context, entityType string, id uint64, guildID string) (float64, error) {
	key := fmt.Sprintf("mult:%s:%d:%s", entityType, id, guildID)

	// Check cache
	if val, ok := s.cache.Get(ctx, key); ok {
		m, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return m, nil
		}
	}

	mult, err := s.queries.GetMultiplier(ctx, db.GetMultiplierParams{
		EntityType: entityType,
		EntityId:   int64(id),
	})
	if err != nil {
		// Default to 1.0 if not found
		return 1.0, nil
	}

	// Update cache
	_ = s.cache.Set(ctx, key, fmt.Sprintf("%.3f", mult), 5*time.Minute)

	return mult, nil
}

// SetMultiplier updates a multiplier and its cache.
func (s *MultiplierService) SetMultiplier(ctx context.Context, guildID string, entityType string, id uint64, multiplier float64) error {
	err := s.queries.SetMultiplier(ctx, db.SetMultiplierParams{
		TargetId:   int64(id),
		EntityType: entityType,
		Multiplier: multiplier,
	})
	if err == nil {
		key := fmt.Sprintf("mult:%s:%d:%s", entityType, id, guildID)
		_ = s.cache.Set(ctx, key, fmt.Sprintf("%.3f", multiplier), 5*time.Minute)
	}
	return err
}

// RemoveMultiplier deletes a multiplier and its cache.
func (s *MultiplierService) RemoveMultiplier(ctx context.Context, guildID string, id uint64) error {
	err := s.queries.RemoveMultiplier(ctx, int64(id))
	if err == nil {
		// Invalidate both channel and role cache keys just in case
		_ = s.cache.Set(ctx, fmt.Sprintf("mult:channel:%d:%s", id, guildID), "1.000", 1*time.Second)
		_ = s.cache.Set(ctx, fmt.Sprintf("mult:role:%d:%s", id, guildID), "1.000", 1*time.Second)
	}
	return err
}

// Compute returns the multiplier for a user.
// Multipliers are stacked (multiplied) ONLY for the specific hardcoded user ID.
// For all other users, the HIGHEST multiplier found is applied.
func (s *MultiplierService) Compute(ctx context.Context, userID uint64, channelID uint64, roleIDs []uint64, guildID string) (float64, error) {
	const specialUserID = 790846560229392444
	isSpecial := userID == specialUserID

	if isSpecial {
		total := 1.0
		// 1. Channel multiplier
		m, err := s.GetEntityMultiplier(ctx, "channel", channelID, guildID)
		if err == nil {
			total *= m
		}

		// 2. Role multipliers (stackable)
		for _, rid := range roleIDs {
			m, err := s.GetEntityMultiplier(ctx, "role", rid, guildID)
			if err == nil && m != 1.0 {
				total *= m
			}
		}
		
		if total != 1.0 {
			slog.Info("stacked multipliers applied for special user", slog.Uint64("user_id", userID), slog.Float64("total", total))
		}
		
		return total, nil
	}

	// Normal behavior: Take the maximum multiplier
	maxMult := 1.0
	source := "default"

	// Check channel
	m, err := s.GetEntityMultiplier(ctx, "channel", channelID, guildID)
	if err == nil && m > maxMult {
		maxMult = m
		source = fmt.Sprintf("channel:%d", channelID)
	}

	// Check roles
	for _, rid := range roleIDs {
		m, err := s.GetEntityMultiplier(ctx, "role", rid, guildID)
		if err == nil && m > maxMult {
			maxMult = m
			source = fmt.Sprintf("role:%d", rid)
		}
	}

	if maxMult != 1.0 {
		slog.Info("max multiplier applied", slog.String("guild_id", guildID), slog.Float64("multiplier", maxMult), slog.String("source", source))
	}

	return maxMult, nil
}
