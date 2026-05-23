package common

import "strings"

// PlayerOutcomeLabel returns won, lost, or draw from the player's perspective.
func PlayerOutcomeLabel(outcome, playerColor string) string {
	switch outcome {
	case "1/2-1/2":
		return "draw"
	case "1-0":
		if playerColor == "white" {
			return "won"
		}
		return "lost"
	case "0-1":
		if playerColor == "black" {
			return "won"
		}
		return "lost"
	default:
		return outcome
	}
}

// HumanizeMethod turns chess library method names into readable phrases.
func HumanizeMethod(method string) string {
	switch method {
	case "", "NoMethod":
		return ""
	case "Checkmate":
		return "Checkmate"
	case "Resignation":
		return "Resignation"
	case "DrawOffer":
		return "Draw by agreement"
	case "Stalemate":
		return "Stalemate"
	case "ThreefoldRepetition":
		return "Threefold repetition"
	case "FivefoldRepetition":
		return "Fivefold repetition"
	case "FiftyMoveRule":
		return "Fifty-move rule"
	case "SeventyFiveMoveRule":
		return "Seventy-five-move rule"
	case "InsufficientMaterial":
		return "Insufficient material"
	case "time forfeit":
		return "Time forfeit"
	default:
		var b strings.Builder
		for i, r := range method {
			if i > 0 && r >= 'A' && r <= 'Z' {
				b.WriteByte(' ')
			}
			if i == 0 {
				b.WriteRune(r)
				continue
			}
			if r >= 'A' && r <= 'Z' {
				b.WriteRune(r + ('a' - 'A'))
			} else {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
}

// GameResultSummary returns a player-facing end-of-game line.
func GameResultSummary(outcome, method, playerColor string) string {
	if outcome == "" || outcome == "*" {
		return "Game over."
	}
	label := PlayerOutcomeLabel(outcome, playerColor)
	methodHuman := HumanizeMethod(method)
	switch label {
	case "won":
		if methodHuman != "" {
			return methodHuman + " — you win!"
		}
		return "You win!"
	case "lost":
		if methodHuman != "" {
			return methodHuman + " — you lose."
		}
		return "You lose."
	case "draw":
		if methodHuman != "" {
			return methodHuman + " — draw."
		}
		return "Draw."
	default:
		return "Game over: " + outcome
	}
}
