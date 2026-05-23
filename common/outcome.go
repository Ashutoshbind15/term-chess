package common

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
