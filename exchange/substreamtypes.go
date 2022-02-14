package exchange

type PCSPairs []PCSPair

// sf.pancakeswap
type PCSPair struct {
	Address               string
	Token0                ERC20Token
	Token1                ERC20Token
	CreationTransactionID string

	CallStartOrdinal      uint64
	CallEndOrdinal        uint64
	TrxStartOrdinal       uint64
	TrxEndOrdinal         uint64
	LogOrdinal            uint64
}

type ERC20Tokens []ERC20Token

type ERC20Token struct {
	Address  string
	Name     string
	Symbol   string
	Decimals uint32
}

type PCSReserveUpdates []PCSReserveUpdate

type PCSReserveUpdate struct {
	PairAddress string
	Reserve0 string
	Reserve1 string

	LogOrdinal uint64
}
