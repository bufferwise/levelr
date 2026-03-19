package cache

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go"
)

// Client wraps valkey.Client with typed helpers for all bot cache operations.
type Client struct {
	vk valkey.Client
}

// New creates a Valkey client connection.
func New(addr string) (*Client, error) {
	vk, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("valkey connect: %w", err)
	}
	slog.Info("valkey connected", slog.String("addr", addr))
	return &Client{vk: vk}, nil
}

// Close gracefully shuts down the Valkey client.
func (c *Client) Close() {
	c.vk.Close()
}

// --- Generic helpers ---

// Set stores a key-value pair with a TTL.
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.vk.Do(ctx, c.vk.B().Set().Key(key).Value(value).Ex(ttl).Build()).Error()
}

// Get retrieves a value by key. Returns empty string and false if not found.
func (c *Client) Get(ctx context.Context, key string) (string, bool) {
	val, err := c.vk.Do(ctx, c.vk.B().Get().Key(key).Build()).ToString()
	if err != nil {
		return "", false
	}
	return val, true
}

// Del removes a key.
func (c *Client) Del(ctx context.Context, key string) error {
	return c.vk.Do(ctx, c.vk.B().Del().Key(key).Build()).Error()
}

// Exists checks if a key exists. Returns true if it does.
func (c *Client) Exists(ctx context.Context, key string) bool {
	n, err := c.vk.Do(ctx, c.vk.B().Exists().Key(key).Build()).AsInt64()
	if err != nil {
		return false
	}
	return n > 0
}

// --- Cooldown helpers ---

// SetCooldown sets a cooldown flag for a user with the given duration.
func (c *Client) SetCooldown(ctx context.Context, userID uint64, seconds int) {
	key := fmt.Sprintf("cooldown:%d", userID)
	_ = c.Set(ctx, key, "1", time.Duration(seconds)*time.Second)
}

// CooldownActive returns true if the user's XP cooldown is still active.
func (c *Client) CooldownActive(ctx context.Context, userID uint64) bool {
	key := fmt.Sprintf("cooldown:%d", userID)
	return c.Exists(ctx, key)
}

// --- Voice session helpers ---

// voiceKey formats the Valkey key for a voice session.
func voiceKey(guildID, userID uint64) string {
	return fmt.Sprintf("voice:%d:%d", guildID, userID)
}

// SetVoiceSession records a user as active in a voice channel.
func (c *Client) SetVoiceSession(ctx context.Context, guildID, userID, channelID uint64, joinUnix int64) {
	key := voiceKey(guildID, userID)
	val := fmt.Sprintf("%d:%d", channelID, joinUnix)
	_ = c.Set(ctx, key, val, 8*time.Hour)
}

// DeleteVoiceSession removes a user's voice session entry.
func (c *Client) DeleteVoiceSession(ctx context.Context, guildID, userID uint64) {
	key := voiceKey(guildID, userID)
	_ = c.Del(ctx, key)
}

// GetVoiceSession returns details for a specific user's voice session.
func (c *Client) GetVoiceSession(ctx context.Context, guildID, userID uint64) (channelID uint64, joinUnix int64, ok bool) {
	val, ok := c.Get(ctx, voiceKey(guildID, userID))
	if !ok {
		return 0, 0, false
	}
	return parseVoiceVal(val)
}

func parseVoiceVal(val string) (uint64, int64, bool) {
	var cid uint64
	var ut int64
	n, err := fmt.Sscanf(val, "%d:%d", &cid, &ut)
	if err != nil || n != 2 {
		return 0, 0, false
	}
	return cid, ut, true
}

type VoiceSession struct {
	UserID    uint64
	ChannelID uint64
	JoinUnix  int64
}

// GetActiveVoiceSessions returns all active session details for a guild.
func (c *Client) GetActiveVoiceSessions(ctx context.Context, guildID uint64) ([]VoiceSession, error) {
	uids, err := c.ScanVoiceSessions(ctx, guildID)
	if err != nil {
		return nil, err
	}

	var sessions []VoiceSession
	for _, uid := range uids {
		cid, ut, ok := c.GetVoiceSession(ctx, guildID, uid)
		if ok {
			sessions = append(sessions, VoiceSession{UserID: uid, ChannelID: cid, JoinUnix: ut})
		}
	}
	return sessions, nil
}

// ScanVoiceSessions returns all active voice user IDs for a guild.
func (c *Client) ScanVoiceSessions(ctx context.Context, guildID uint64) ([]uint64, error) {
	pattern := fmt.Sprintf("voice:%d:*", guildID)
	prefix := fmt.Sprintf("voice:%d:", guildID)

	var userIDs []uint64
	var cursor uint64

	for {
		cmd := c.vk.B().Scan().Cursor(cursor).Match(pattern).Count(100).Build()
		res, err := c.vk.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			return nil, fmt.Errorf("voice scan: %w", err)
		}

		for _, key := range res.Elements {
			uidStr := key[len(prefix):]
			uid, err := strconv.ParseUint(uidStr, 10, 64)
			if err != nil {
				continue
			}
			userIDs = append(userIDs, uid)
		}

		cursor = res.Cursor
		if cursor == 0 {
			break
		}
	}

	return userIDs, nil
}

// --- AFK channel cache ---

func afkKey(guildID uint64) string {
	return fmt.Sprintf("afk:%d", guildID)
}

// SetAFKChannel caches the guild's AFK channel ID.
func (c *Client) SetAFKChannel(ctx context.Context, guildID, channelID uint64) {
	_ = c.Set(ctx, afkKey(guildID), strconv.FormatUint(channelID, 10), time.Hour)
}

// GetAFKChannel retrieves the cached AFK channel ID (0 if not found).
func (c *Client) GetAFKChannel(ctx context.Context, guildID uint64) uint64 {
	val, ok := c.Get(ctx, afkKey(guildID))
	if !ok {
		return 0
	}
	id, _ := strconv.ParseUint(val, 10, 64)
	return id
}

// --- Invalidation helpers ---

// InvalidateBlacklist removes a specific blacklist entry from cache.
func (c *Client) InvalidateBlacklist(ctx context.Context, entityType string, id uint64) {
	key := fmt.Sprintf("blacklist:%s:%d", entityType, id)
	_ = c.Del(ctx, key)
}

// InvalidateMultiplier removes a specific multiplier entry from cache.
func (c *Client) InvalidateMultiplier(ctx context.Context, entityType string, id uint64) {
	key := fmt.Sprintf("mult:%s:%d", entityType, id)
	_ = c.Del(ctx, key)
}
