package giveaways

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/disgoorg/snowflake/v2"
)

// DrawWinnersCSPRNG draws up to count distinct winners using a cryptographically secure random ticket pool.
// It dynamically purges already selected winners from the active pool to ensure fair, non-duplicate drawing.
func DrawWinnersCSPRNG(tickets []snowflake.ID, count int) []snowflake.ID {
	if len(tickets) == 0 || count <= 0 {
		return nil
	}

	winners := make([]snowflake.ID, 0, count)
	seen := make(map[snowflake.ID]bool)

	for len(winners) < count {
		var activeTickets []snowflake.ID
		for _, ticket := range tickets {
			if !seen[ticket] {
				activeTickets = append(activeTickets, ticket)
			}
		}

		if len(activeTickets) == 0 {
			break
		}

		var b [4]byte
		_, err := rand.Read(b[:])
		if err != nil {
			break
		}

		val := binary.BigEndian.Uint32(b[:])
		index := int(val % uint32(len(activeTickets)))
		winnerID := activeTickets[index]

		seen[winnerID] = true
		winners = append(winners, winnerID)
	}

	return winners
}

// RandomColor generates a cryptographically secure random color for Discord embeds (0x000000 - 0xFFFFFF)
func RandomColor() int {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
}
