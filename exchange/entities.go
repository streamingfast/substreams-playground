package exchange

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (t *Token) Sanitize() {
	t.Name = strings.ReplaceAll(t.Name, "\u0000", "")
	t.Symbol = strings.ReplaceAll(t.Symbol, "\u0000", "")
}

func (p *Pair) Sanitize() {
	p.Name = strings.ReplaceAll(p.Name, "\u0000", "")
}

func (e *PancakeDayData) IsFinal(blockNum uint64, blockTime time.Time) bool {
	dayId := blockTime.Unix() / 86400
	activeId := strconv.FormatInt(dayId, 10)

	return e.ID != activeId
}

func (p *PairHourData) IsFinal(blockNum uint64, blockTime time.Time) bool {
	hourId := blockTime.Unix() / 3600
	activeId := fmt.Sprintf("%s-%d", p.Pair, hourId)

	return p.ID != activeId
}

func (p *PairDayData) IsFinal(blockNum uint64, blockTime time.Time) bool {
	dayId := blockTime.Unix() / 86400
	activeId := fmt.Sprintf("%s-%d", p.PairAddress, dayId)

	return p.ID != activeId
}

func (p *TokenDayData) IsFinal(blockNum uint64, blockTime time.Time) bool {
	dayId := blockTime.Unix() / 86400
	activeId := fmt.Sprintf("%s-%d", p.Token, dayId)

	return p.ID != activeId
}

func (e *Transaction) IsFinal(blockNum uint64, blockTime time.Time) bool {
	return true
}

func (e *Swap) IsFinal(blockNum uint64, blockTime time.Time) bool {
	return true
}

func (*Mint) IsFinal(uint64, time.Time) bool {
	return true
}

func (*Burn) IsFinal(uint64, time.Time) bool {
	return true
}
