package pcs

import (
	"encoding/json"
	"fmt"
)

type PCSPairs []PCSPair

func (p PCSPairs) Print() {
	if len(p) == 0 {
		return
	}
	fmt.Println("Pairs updates:")
	cnt, _ := json.MarshalIndent(p, "", "  ")
	fmt.Println(string(cnt))
}

// sf.pancakeswap
type PCSPair struct {
	Address               string
	Token0                ERC20Token
	Token1                ERC20Token
	CreationTransactionID string
	BlockNum              uint64

	LogOrdinal uint64
}

func (p PCSPair) GetOrdinal() uint64 { return p.LogOrdinal }

type ERC20Tokens []ERC20Token

type ERC20Token struct {
	Address  string
	Name     string
	Symbol   string
	Decimals int64
}

type PCSReserveUpdates []PCSReserveUpdate

func (p PCSReserveUpdates) Print() {
	if len(p) == 0 {
		return
	}
	fmt.Println("Reserve updates:")
	cnt, _ := json.MarshalIndent(p, "", "  ")
	fmt.Println(string(cnt))
}

type PCSReserveUpdate struct {
	PairAddress string
	Reserve0    string
	Reserve1    string

	LogOrdinal uint64

	Token0Price string
	Token1Price string
}

// func (u PCSReserveUpdate) Token0Price() *big.Float {
// 	if len(ev.Reserve1.Bits()) == 0 {
// 		return big.NewFloat(0)
// 	} else {
// 		return  bf().Quo(reserve0.Float(), reserve1.Float())
// 	}
// }

// func (u PCSReserveUpdate) Token1Price() *big.Float {
// 	if len(ev.Reserve0.Bits()) == 0 {
// 		return big.NewFloat(0)
// 	} else {
// 		return bf().Quo(reserve1.Float(), reserve0.Float())
// 	}
// }

type PCSEvents []PCSEvent

func (p PCSEvents) Print() {
	if len(p) == 0 {
		return
	}
	fmt.Println("PCS Events updates:")
	cnt, _ := json.MarshalIndent(p, "", "  ")
	fmt.Println(string(cnt))
}

type VolumeAggregate struct {
	Pair string
	Date int64

	VolumeUSD float64
}

type PCSEvent interface {
	GetOrdinal() uint64
	SetBase(e PCSBaseEvent)
}

type PCSBaseEvent struct {
	PairAddress   string
	Token0        string
	Token1        string
	TransactionID string
	Timestamp     uint64
}

type PCSSwap struct {
	PCSBaseEvent
	Type       string
	LogOrdinal uint64

	Sender string
	To     string
	From   string

	Amount0In  string
	Amount1In  string
	Amount0Out string
	Amount1Out string
	AmountBNB  string
	AmountUSD  string
}

func (e *PCSSwap) GetOrdinal() uint64 { return e.LogOrdinal }
func (e *PCSMint) GetOrdinal() uint64 { return e.LogOrdinal }
func (e *PCSBurn) GetOrdinal() uint64 { return e.LogOrdinal }

func (e *PCSSwap) SetBase(base PCSBaseEvent) { e.PCSBaseEvent = base }
func (e *PCSMint) SetBase(base PCSBaseEvent) { e.PCSBaseEvent = base }
func (e *PCSBurn) SetBase(base PCSBaseEvent) { e.PCSBaseEvent = base }

type PCSBurn struct {
	PCSBaseEvent
	Type       string
	LogOrdinal uint64

	To     string
	Sender string
	FeeTo  string

	Amount0   string
	Amount1   string
	AmountUSD string

	Liquidity    string
	FeeLiquidity string
}

type PCSMint struct {
	PCSBaseEvent
	Type       string
	LogOrdinal uint64

	To     string
	Sender string
	FeeTo  string

	Amount0   string
	Amount1   string
	AmountUSD string

	Liquidity    string
	FeeLiquidity string
}
