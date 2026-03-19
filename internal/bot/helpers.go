package bot

import (
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"

	"github.com/bufferwise/levelr/internal/config"
)

// IsMainGuild checks if the given id matches the main guild ID.
func IsMainGuild(id snowflake.ID, cfg *config.Config) bool {
	return uint64(id) == cfg.MainGuildID
}

// snowflakeFromUint64 converts a uint64 to a disgo snowflake.
func snowflakeFromUint64(id uint64) snowflake.ID {
	return snowflake.ID(id)
}

// SnowflakeFromUint64 is the exported version for use by other packages.
func SnowflakeFromUint64(id uint64) snowflake.ID {
	return snowflake.ID(id)
}

// timeNowUnix returns the current UTC unix timestamp.
func timeNowUnix() int64 {
	return time.Now().UTC().Unix()
}

// Ensure discord import is used (type alias for convenience).
type VoiceState = discord.VoiceState

func Ptr[T any](v T) *T {
	return &v
}

func BoolPtr(v bool) *bool {
	return &v
}

// SnowflakeSliceToUint64 converts a slice of snowflake.IDs to a slice of uint64s.
func SnowflakeSliceToUint64(ids []snowflake.ID) []uint64 {
	res := make([]uint64, len(ids))
	for i, id := range ids {
		res[i] = uint64(id)
	}
	return res
}
