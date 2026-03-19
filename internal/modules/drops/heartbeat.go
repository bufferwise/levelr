package drops

import (
	"context"
	"log/slog"
	"time"

	"github.com/bufferwise/levelr/internal/config"
)

// StartDropTicker returns a worker routine that runs the auto-drop loop.
func StartDropTicker(cfg *config.Config, dropSvc *DropService) func(ctx context.Context) {
	return func(ctx context.Context) {
		if cfg.DropChannelID == 0 {
			slog.Warn("DROP_CHANNEL_ID not set — auto-drops disabled")
			return
		}

		interval := time.Duration(cfg.DropIntervalMin) * time.Minute
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		slog.Info("drop ticker started",
			slog.Uint64("channel_id", cfg.DropChannelID),
			slog.Int("interval_min", cfg.DropIntervalMin),
		)

		for {
			select {
			case <-ctx.Done():
				slog.Info("drop ticker stopped")
				return
			case <-ticker.C:
				err := dropSvc.SendDrop(ctx, cfg.DropChannelID, nil)
				if err != nil {
					slog.Error("auto-drop failed", slog.Any("err", err))
				}
			}
		}
	}
}
