package common

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/notnil/chess"
)

func InitTextInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Guest"
	ti.Focus()
	ti.Prompt = ">"
	return ti
}

func InitSpinner() spinner.Model {
	return spinner.New(spinner.WithSpinner(spinner.MiniDot))
}

// --- board history helpers ---

const (
	HistoryFromBorderColor = "39"  // sky blue — origin square
	HistoryToBorderColor   = "208" // orange — destination square
)

// BoardPieceMode controls how occupied squares are drawn on the board.
type BoardPieceMode int

const (
	BoardPieceUnicode BoardPieceMode = iota // ♔♕♖… symbols
	BoardPieceBare                          // FEN letters (K, q, …)
)

func (m BoardPieceMode) Toggle() BoardPieceMode {
	if m == BoardPieceUnicode {
		return BoardPieceBare
	}
	return BoardPieceUnicode
}

func (m BoardPieceMode) Label() string {
	switch m {
	case BoardPieceBare:
		return "bare"
	default:
		return "unicode"
	}
}

func PieceStyleHelpLine(mode BoardPieceMode) string {
	return fmt.Sprintf("Piece style: ctrl+b · %s", mode.Label())
}

// BoardHistoryView describes the board position and move highlight for one
// ply in the move list. Ply 0 is the starting position; ply N is after N
// half-moves. When ply == len(moves) the live FEN from the server is used.
type BoardHistoryView struct {
	Ply      int
	Total    int
	FEN      string
	MoveFrom string
	MoveTo   string
	MoveUCI  string
	AtLive   bool
}

func BoardHistoryViewFor(moves []string, liveFEN string, ply int) BoardHistoryView {
	total := len(moves)
	if ply < 0 {
		ply = total
	}
	if ply > total {
		ply = total
	}

	v := BoardHistoryView{
		Ply:    ply,
		Total:  total,
		AtLive: ply == total,
	}

	g := chess.NewGame(chess.UseNotation(chess.UCINotation{}))
	if ply == 0 {
		v.FEN = g.FEN()
		return v
	}

	for i := 0; i < ply; i++ {
		_ = g.MoveStr(moves[i])
	}
	if v.AtLive && liveFEN != "" {
		v.FEN = liveFEN
	} else {
		v.FEN = g.FEN()
	}

	if ply > 0 {
		uci := moves[ply-1]
		v.MoveUCI = uci
		if len(uci) >= 4 {
			v.MoveFrom = uci[:2]
			v.MoveTo = uci[2:4]
		}
	}
	return v
}

func HistoryStatusLine(v BoardHistoryView) string {
	if v.Total == 0 {
		return ""
	}
	if v.Ply == 0 {
		if v.AtLive {
			return "Starting position."
		}
		return fmt.Sprintf("Starting position (0/%d) — press → for next", v.Total)
	}
	if v.AtLive {
		return fmt.Sprintf("Move %d/%d: %s", v.Ply, v.Total, v.MoveUCI)
	}
	return fmt.Sprintf("Move %d/%d: %s — press → to resume live position", v.Ply, v.Total, v.MoveUCI)
}

func AdjustHistoryPly(ply, total int, key string) (int, bool) {
	if total == 0 {
		return ply, false
	}
	switch key {
	case "left":
		if ply > 0 {
			return ply - 1, true
		}
		return ply, true
	case "right":
		if ply < total {
			return ply + 1, true
		}
		return ply, true
	}
	return ply, false
}
