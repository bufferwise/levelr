package leveling

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/services"
	"github.com/disgoorg/snowflake/v2"
)

func StartWeeklyReportWorker(queries *db.Queries, notifier services.Notifier, mainGuildID uint64) func(ctx context.Context) {
	return func(ctx context.Context) {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		slog.Info("starting weekly report worker")

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now().UTC()
				// We want to send the report for the *last* week on Monday 00:00 (or as soon as possible after that)
				// Actually, let's just check if it's Monday.
				if now.Weekday() != time.Monday {
					continue
				}

				weekStart := services.WeekStart(now.AddDate(0, 0, -1)) // Get last week's start
				weekStr := services.WeekStartString(weekStart)

				// Check if already reported
				lastReported, _ := queries.GetConfig(ctx, "last_weekly_report")
				if lastReported == weekStr {
					continue
				}

				slog.Info("generating weekly report", slog.String("week", weekStr))

				// Fetch top 10 messages
				topMsg, err := queries.GetWeeklyMsgLeaderboard(ctx, db.GetWeeklyMsgLeaderboardParams{
					YearWeek: weekStr,
					Limit:    10,
					Offset:   0,
				})
				if err != nil {
					slog.Error("failed to fetch weekly msg leaderboard", slog.Any("err", err))
					continue
				}

				// Fetch top 10 voice
				topVC, err := queries.GetWeeklyVCLeaderboard(ctx, db.GetWeeklyVCLeaderboardParams{
					YearWeek: weekStr,
					Limit:    10,
					Offset:   0,
				})
				if err != nil {
					slog.Error("failed to fetch weekly vc leaderboard", slog.Any("err", err))
					continue
				}

				if len(topMsg) == 0 && len(topVC) == 0 {
					slog.Info("no activity this week, skipping report")
					_ = queries.SetConfig(ctx, db.SetConfigParams{Key: "last_weekly_report", Value: weekStr})
					continue
				}

				// Format message
				report := fmt.Sprintf("## 🏆 Weekly Activity Report (%s)\n\n", weekStr)

				report += "### 💬 Top Message Grinders\n"
				if len(topMsg) == 0 {
					report += "No messages sent this week.\n"
				} else {
					for i, u := range topMsg {
						report += fmt.Sprintf("%d. <@%s> — **%d** messages\n", i+1, u.Userid, u.Count)
					}
				}

				report += "\n### 🎙️ Top Voice Spiders\n"
				if len(topVC) == 0 {
					report += "No voice activity this week.\n"
				} else {
					for i, u := range topVC {
						report += fmt.Sprintf("%d. <@%s> — **%d** minutes\n", i+1, u.Userid, u.Minutes)
					}
				}

				report += "\n> <:Spider_Sparkle:1336641144674717696> **Keep Grinding! Next report on next Monday!**"

				// Send report
				notifier.SendWeeklyReport(ctx, snowflake.ID(mainGuildID), report)

				// Mark as reported
				_ = queries.SetConfig(ctx, db.SetConfigParams{Key: "last_weekly_report", Value: weekStr})
			}
		}
	}
}
