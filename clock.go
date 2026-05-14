package main

import (
	"time"

	"github.com/charmbracelet/log"
	"github.com/notnil/chess"

	"github.com/Ashutoshbind15/ssh-chess/managers"
)

// ClockUpdateMsg / TimeForfeitMsg are page messages emitted by the clock
// ticker. They live in the cmd package next to the rest of the page msg
// types because the ticker that produces them is itself a cmd-level
// composition of GameManager + DataManager + SessionManager.
type ClockUpdateMsg struct {
	WhiteTime time.Duration
	BlackTime time.Duration
	GameID    string
}

type TimeForfeitMsg struct {
	GameID     string
	LoserColor chess.Color
}

// runClockTicker drives in-progress game clocks. It is intentionally cmd
// code rather than a manager: it stitches GameManager (state), DataManager
// (persistence) and SessionManager (delivery) together, but none of those
// know about each other. Stops when stopCh is closed.
func runClockTicker(stopCh <-chan struct{}) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			tickClocks()
		}
	}
}

func tickClocks() {
	for _, game := range gameManager.AllInProgressGames() {
		if expired, loserColor := game.IsTimeExpired(); expired {
			handleTimeForfeit(game, loserColor)
			continue
		}

		whiteTime, blackTime := game.CurrentClocks()
		msg := ClockUpdateMsg{
			WhiteTime: whiteTime,
			BlackTime: blackTime,
			GameID:    game.ID(),
		}

		whiteFP, blackFP := game.Fingerprints()
		for _, fp := range []string{whiteFP, blackFP} {
			if fp == "" {
				continue
			}
			if prog := sessionManager.GetProgram(fp); prog != nil {
				prog.Send(msg)
			}
		}
	}
}

func handleTimeForfeit(game *managers.Game, loserColor chess.Color) {
	gameID := game.ID()
	if finished := gameManager.EndByTimeForfeit(gameID, loserColor); finished == nil {
		return
	}

	whiteFP, blackFP := game.Fingerprints()

	record := gameManager.BuildGameRecord(gameID)
	record.Method = managers.MethodTimeForfeit
	if err := dataManager.AddGame(record); err != nil {
		log.Error("failed to persist time-forfeit game", "id", gameID, "error", err)
	}
	gameManager.RemoveGame(gameID)

	log.Info("Time forfeit", "game", gameID, "loser_color", loserColor)

	forfeitMsg := TimeForfeitMsg{
		GameID:     gameID,
		LoserColor: loserColor,
	}
	for _, fp := range []string{whiteFP, blackFP} {
		if fp == "" {
			continue
		}
		if prog := sessionManager.GetProgram(fp); prog != nil {
			prog.Send(forfeitMsg)
		}
	}
}
