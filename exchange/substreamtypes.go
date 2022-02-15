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

	CallStartOrdinal uint64
	CallEndOrdinal   uint64
	TrxStartOrdinal  uint64
	TrxEndOrdinal    uint64
	LogOrdinal       uint64
}

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

	LogOrdinal  uint64
	Token0Price string
	Token1Price string
}
