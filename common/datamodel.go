package common

import (
	"time"

	"gorm.io/gorm"
)

type Player struct {
	gorm.Model
	Fingerprint string `gorm:"uniqueIndex"`
	Username    string
}

type Game struct {
	gorm.Model
	GameID           string `gorm:"uniqueIndex"`
	WhiteFingerprint string `gorm:"index"`
	WhiteUsername    string
	BlackFingerprint string `gorm:"index"`
	BlackUsername    string
	PGN              string
	Outcome          string
	Method           string
	TimeControl      int
}

type BotGame struct {
	gorm.Model
	BotGameID         string `gorm:"uniqueIndex"`
	PlayerFingerprint string `gorm:"index"`
	PlayerUsername    string
	PlayerColor       string // "white" or "black"
	BotLevel          int
	PGN               string
	Outcome           string
	Method            string
}

type LiveGameEvent struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	GameID    string `gorm:"index:idx_live_game_version,unique;not null"`
	Version int    `gorm:"index:idx_live_game_version,unique;not null"`
	Payload []byte `gorm:"type:jsonb;not null"`
}

// LiveGamePayload is the JSON snapshot stored in LiveGameEvent.Payload.
type LiveGamePayload struct {
	GameID           string   `json:"game_id"`
	Code             string   `json:"code"`
	Status           string   `json:"status"`
	TimeControl      int      `json:"time_control"`
	WhiteFingerprint string   `json:"white_fingerprint"`
	BlackFingerprint string   `json:"black_fingerprint"`
	WhiteUsername    string   `json:"white_username"`
	BlackUsername    string   `json:"black_username"`
	Moves            []string `json:"moves"`
	WhiteTimeLeftMs  int64    `json:"white_time_left_ms"`
	BlackTimeLeftMs  int64    `json:"black_time_left_ms"`
	TurnStartedAt    *string  `json:"turn_started_at,omitempty"`
}
