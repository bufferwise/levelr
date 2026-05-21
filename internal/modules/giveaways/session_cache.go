package giveaways

import (
	"fmt"
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

type DraftSession struct {
	GuildID          snowflake.ID
	ChannelID        snowflake.ID
	HostID           snowflake.ID
	Prize            string
	Duration         time.Duration
	WinnerCount      int
	RequiredRole     *snowflake.ID
	RequiredLevel    int
	MinAccountDays   int
	RoleMultipliers  map[snowflake.ID]float64
	State            string // "main" or "multipliers"
	LastInteraction  time.Time
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*DraftSession
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*DraftSession),
	}
	go sm.startGC()
	return sm
}

func (sm *SessionManager) Get(guildID, userID snowflake.ID) (*DraftSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", guildID, userID)
	s, ok := sm.sessions[key]
	if ok {
		s.LastInteraction = time.Now()
	}
	return s, ok
}

func (sm *SessionManager) Set(guildID, userID snowflake.ID, s *DraftSession) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	key := fmt.Sprintf("%s:%s", guildID, userID)
	s.LastInteraction = time.Now()
	sm.sessions[key] = s
}

func (sm *SessionManager) Delete(guildID, userID snowflake.ID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	key := fmt.Sprintf("%s:%s", guildID, userID)
	delete(sm.sessions, key)
}

func (sm *SessionManager) startGC() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for k, s := range sm.sessions {
			if now.Sub(s.LastInteraction) > 15*time.Minute {
				delete(sm.sessions, k)
			}
		}
		sm.mu.Unlock()
	}
}
