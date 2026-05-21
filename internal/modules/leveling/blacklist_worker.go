package leveling

import (
	"context"
	"log/slog"
	"time"

	"github.com/bufferwise/levelr/internal/services"
)

func StartBlacklistExpirationWorker(blSvc *services.BlacklistService) func(ctx context.Context) {
	return func(ctx context.Context) {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		slog.Info("starting blacklist expiration worker")

		// Run pruning once on startup
		if count, err := blSvc.PruneExpiredBlacklists(ctx); err == nil && count > 0 {
			slog.Info("pruned expired blacklists on startup", slog.Int64("count", count))
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count, err := blSvc.PruneExpiredBlacklists(ctx)
				if err != nil {
					slog.Error("failed to prune expired blacklists", slog.Any("err", err))
				} else if count > 0 {
					slog.Info("successfully pruned expired blacklists", slog.Int64("count", count))
				}
			}
		}
	}
}
