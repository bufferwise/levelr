package presence

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bufferwise/levelr/internal/config"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/gateway"
)

type PresenceService struct {
	cfg    *config.Config
	client *bot.Client
}

func NewPresenceService(cfg *config.Config, client *bot.Client) *PresenceService {
	return &PresenceService{
		cfg:    cfg,
		client: client,
	}
}

type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Entries []Entry  `xml:"entry"`
}

type Entry struct {
	Title string `xml:"title"`
}

func (s *PresenceService) StartTicker(ctx context.Context) {
	// Wait momentarily to ensure the gateway connection is fully open and the shard is ready.
	// Otherwise, calling SetPresence will error with "shard is not ready".
	time.Sleep(15 * time.Second)

	// Run once immediately on startup
	s.updatePresence(ctx)

	// Refresh every 6 hours as requested
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Debug("youtube_presence_ticker shutting down")
			return
		case <-ticker.C:
			s.updatePresence(ctx)
		}
	}
}

func (s *PresenceService) updatePresence(ctx context.Context) {
	handles := strings.Split(s.cfg.YoutubeHandles, ",")
	if len(handles) == 0 || s.cfg.YoutubeHandles == "" {
		slog.Warn("youtube presence enabled but no handles configured")
		return
	}

	// Just use the first handler for now or loop over them.
	// Since Discord presence only takes one activity text, we fetch the latest video from the first valid handle.
	for _, handle := range handles {
		handle = strings.TrimSpace(handle)
		if handle == "" {
			continue
		}

		channelID, err := getChannelID(handle)
		if err != nil {
			slog.Warn("failed to get channel ID for presence", slog.String("handle", handle), slog.Any("err", err))
			continue
		}

		title, err := getLatestVideoTitle(channelID)
		if err != nil {
			slog.Warn("failed to get latest video title for presence", slog.String("channel_id", channelID), slog.Any("err", err))
			continue
		}

		if title != "" {
			err = (*s.client).SetPresence(ctx, gateway.WithStreamingActivity(title, "https://twitch.tv/discord"))
			if err != nil {
				slog.Error("failed to update presence", slog.Any("err", err))
			} else {
				slog.Info("updated youtube presence", slog.String("title", title))
			}
			return // Stop processing after successfully updating with the latest video
		}
	}
}

func getChannelID(handle string) (string, error) {
	// If it already looks like a Channel ID, return it
	if strings.HasPrefix(handle, "UC") && len(handle) >= 24 {
		return handle, nil
	}

	// Ensure handle starts with @ if it's a handle
	urlHandle := handle
	if !strings.HasPrefix(handle, "@") && !strings.HasPrefix(handle, "channel/") && !strings.HasPrefix(handle, "user/") {
		urlHandle = "@" + handle
	}

	resp, err := http.Get("https://www.youtube.com/" + urlHandle)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	html := string(body)

	// Pattern 1: <meta itemprop="channelId" content="...">
	re1 := regexp.MustCompile(`itemprop="channelId" content="([^"]+)"`)
	if matches := re1.FindStringSubmatch(html); len(matches) > 1 {
		return matches[1], nil
	}

	// Pattern 2: <link rel="canonical" href=".../channel/UC...">
	re2 := regexp.MustCompile(`channel/(UC[a-zA-Z0-9_-]{22})`)
	if matches := re2.FindStringSubmatch(html); len(matches) > 1 {
		return matches[1], nil
	}

	// Pattern 3: Existing pattern channel_id=...
	re3 := regexp.MustCompile(`channel_id=([^"&?]+)`)
	if matches := re3.FindStringSubmatch(html); len(matches) > 1 {
		return matches[1], nil
	}

	// Pattern 4: meta product:availability? Wait, let's try externalId
	re4 := regexp.MustCompile(`"externalId":"(UC[a-zA-Z0-9_-]{22})"`)
	if matches := re4.FindStringSubmatch(html); len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("channel ID not found for handle %s", handle)
}

func getLatestVideoTitle(channelID string) (string, error) {
	resp, err := http.Get("https://www.youtube.com/feeds/videos.xml?channel_id=" + channelID)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch RSS feed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var feed Feed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return "", err
	}

	if len(feed.Entries) > 0 {
		return feed.Entries[0].Title, nil
	}

	return "", nil
}
