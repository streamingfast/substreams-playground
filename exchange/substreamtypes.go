package exchange

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
	Decimals uint32
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

// type PairPrice struct {}
// type TokenPrice struct {}

// type Volume24hStateBuidler struct {}

// type VolumeStateBuilderPerPair struct {}
// type VolumeStateBuilderPerToken struct {}

type Swaps []PCSSwap

func (p Swaps) Print() {
	if len(p) == 0 {
		return
	}
	fmt.Println("Swaps updates:")
	cnt, _ := json.MarshalIndent(p, "", "  ")
	fmt.Println(string(cnt))
}

type PCSSwap struct {
	PairAddress string
	// Token0      string
	// Token1      string

	Transaction string

	Amount0In  string
	Amount1In  string
	Amount0Out string
	Amount1Out string

	AmountUSD string

	Sender string
	To     string
	From   string

	LogOrdinal uint64
}

func (s PCSSwap) GetOrdinal() uint64 { return s.LogOrdinal }

type VolumeAggregate struct {
	Pair string
	Date int64

	VolumeUSD float64
}
