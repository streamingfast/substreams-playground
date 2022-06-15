package graphnode

import (
	"encoding/json"
	"fmt"
	eth "github.com/streamingfast/eth-go"
	pbeth "github.com/streamingfast/sf-ethereum/types/pb/sf/ethereum/type/v1"
	graphnode "github.com/streamingfast/substream-pancakeswap/graph-node"
	"github.com/streamingfast/substream-pancakeswap/graph-node/subgraph"
	"math/big"
)

const (
	FactoryAddress = "0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73"
	ZeroAddress    = "0x0000000000000000000000000000000000000000"
)

var (
	FactoryAddressBytes = eth.MustNewAddress(FactoryAddress).Bytes()
	ZeroAddressBytes    = eth.MustNewAddress(ZeroAddress).Bytes()
)

// Aliases for numerical functions
var (
	S  = graphnode.S
	B  = graphnode.B
	F  = graphnode.NewFloat
	FL = graphnode.NewFloatFromLiteral
	I  = graphnode.NewInt
	IL = graphnode.NewIntFromLiteral
	bf = func() *big.Float { return new(big.Float) }
	bi = func() *big.Int { return new(big.Int) }
)

var Definition = &subgraph.Definition{
	PackageName:         "exchange",
	HighestParallelStep: 4,
	StartBlock:          6809737,
	IncludeFilter:       "",
	Entities: graphnode.NewRegistry(
		&PancakeFactory{},
		&Bundle{},
		&Token{},
		&Pair{},
		&Transaction{},
		&Mint{},
		&Burn{},
		&Swap{},
		&PancakeDayData{},
		&PairHourData{},
		&PairDayData{},
		&TokenDayData{},
	),
	DDL: ddl,
	Manifest: `specVersion: 0.0.2
description: PancakeSwap is a decentralized protocol for automated token exchange on Binance Smart Chain. (Handle Redos)
repository: https://github.com/pancakeswap
schema:
  file: ./exchange.graphql
dataSources:
  - kind: ethereum/contract
    name: Factory
    network: bsc
    source:
      address: '0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73'
      abi: Factory
      startBlock: 6809737
    mapping:
      kind: ethereum/events
      apiVersion: 0.0.8
      language: wasm/assemblyscript
      file: ../src/exchange/factory.ts
      entities:
        - Pair
        - Token
      abis:
        - name: Factory
          file: ../abis/Factory.json
        - name: BEP20
          file: ../abis/BEP20.json
        - name: BEP20NameBytes
          file: ../abis/BEP20NameBytes.json
        - name: BEP20SymbolBytes
          file: ../abis/BEP20SymbolBytes.json
      eventHandlers:
        - event: PairCreated(indexed address,indexed address,address,uint256)
          handler: handlePairCreated
templates:
  - kind: ethereum/contract
    name: Pair
    network: bsc
    source:
      abi: Pair
    mapping:
      kind: ethereum/events
      apiVersion: 0.0.4
      language: wasm/assemblyscript
      file: ../src/exchange/core.ts
      entities:
        - Pair
        - Token
      abis:
        - name: Factory
          file: ../abis/Factory.json
        - name: Pair
          file: ../abis/Pair.json
      eventHandlers:
        - event: Mint(indexed address,uint256,uint256)
          handler: handleMint
        - event: Burn(indexed address,uint256,uint256,indexed address)
          handler: handleBurn
        - event: Swap(indexed address,uint256,uint256,uint256,uint256,indexed address)
          handler: handleSwap
        - event: Transfer(indexed address,indexed address,uint256)
          handler: handleTransfer
        - event: Sync(uint112,uint112)
          handler: handleSync
`,
	GraphQLSchema: `type PancakeFactory @entity {
  id: ID!

  "Total of pairs"
  totalPairs: BigInt! @parallel(step: 1, type: SUM)

  "Total of transactions"
  totalTransactions: BigInt! @parallel(step: 4, type: SUM)

  # total volume
  totalVolumeUSD: BigDecimal! @parallel(step: 4, type: SUM)
  totalVolumeBNB: BigDecimal! @parallel(step: 4, type: SUM)

  # untracked values - less confident USD scores
  untrackedVolumeUSD: BigDecimal! @parallel(step: 4, type: SUM)

  # total liquidity
  totalLiquidityUSD: BigDecimal! @parallel(step: 4)
  totalLiquidityBNB: BigDecimal! @parallel(step: 4)
}

type Bundle @entity {
  id: ID!

  "BNB price, in USD"
  bnbPrice: BigDecimal! @parallel(step: 4)
}

type Token @entity {
  id: ID!

  "Name"
  name: String! @parallel(step: 1)
  "Symbol"
  symbol: String! @parallel(step: 1)
  "Decimals"
  decimals: BigInt! @parallel(step: 1)

  # token specific volume
  tradeVolume: BigDecimal!        @parallel(step: 4, type: SUM)
  tradeVolumeUSD: BigDecimal!     @parallel(step: 4, type: SUM) @sql(index: false)
  untrackedVolumeUSD: BigDecimal! @parallel(step: 4, type: SUM)

  # transactions across all pairs
  totalTransactions: BigInt!  @parallel(step: 4, type: SUM)

  # liquidity across all pairs
  totalLiquidity: BigDecimal!  @parallel(step: 4, type: SUM)

  # derived prices
  derivedBNB: BigDecimal @parallel(step: 2)
  derivedUSD: BigDecimal @parallel(step: 2)

  # derived fields
  tokenDayData: [TokenDayData!]! @derivedFrom(field: "token")
  pairDayDataBase: [PairDayData!]! @derivedFrom(field: "token0")
  pairDayDataQuote: [PairDayData!]! @derivedFrom(field: "token1")
  pairBase: [Pair!]! @derivedFrom(field: "token0")
  pairQuote: [Pair!]! @derivedFrom(field: "token1")
}

type Pair @entity {
  id: ID!

  name: String! @parallel(step: 1)

  # mirrored from the smart contract
  token0: Token! @parallel(step: 1)
  token1: Token! @parallel(step: 1)
  reserve0: BigDecimal!  @parallel(step: 2)
  reserve1: BigDecimal!  @parallel(step: 2)
  totalSupply: BigDecimal! @parallel(step: 4, type: SUM)

  # derived liquidity
  reserveBNB: BigDecimal!  @parallel(step: 3)
  reserveUSD: BigDecimal!  @parallel(step: 3) @sql(index: false)
  trackedReserveBNB: BigDecimal! @sql(index: false) # used for separating per pair reserves and global
  # Price in terms of the asset pair
  token0Price: BigDecimal! @parallel(step: 2)
  token1Price: BigDecimal! @parallel(step: 2)

  # lifetime volume stats
  volumeToken0: BigDecimal!  @parallel(step: 4, type: SUM)
  volumeToken1: BigDecimal! @parallel(step: 4, type: SUM)
  volumeUSD: BigDecimal! @parallel(step: 4, type: SUM) @sql(index: false)
  untrackedVolumeUSD: BigDecimal! @parallel(step: 4, type: SUM)
  totalTransactions: BigInt! @parallel(step: 4, type: SUM)

  block: BigInt! @parallel(step: 1)
  timestamp: BigInt! @parallel(step: 1)

  # derived fields
  pairHourData: [PairHourData!]! @derivedFrom(field: "pair")
  mints: [Mint!]! @derivedFrom(field: "pair")
  burns: [Burn!]! @derivedFrom(field: "pair")
  swaps: [Swap!]! @derivedFrom(field: "pair")
}

type Transaction @entity @cache(skip_db_lookup: true) {
  id: ID!

  block: BigInt! @parallel(step: 4)
  timestamp: BigInt! @parallel(step: 4)
  # This is not the reverse of Mint.transaction; it is only used to
  # track incomplete mints (similar for burns and swaps)
  mints: [Mint]!
  burns: [Burn]!
  swaps: [Swap]!
}

type Mint @entity {
  # transaction hash + "-" + index in mints Transaction array
  id: ID!
  transaction: Transaction! @parallel(step: 4)
  timestamp: BigInt!  @parallel(step: 4) # need this to pull recent txns for specific token or pair
  pair: Pair! @parallel(step: 4)
  token0: Token! @parallel(step: 4)
  token1: Token! @parallel(step: 4)

  # populated from the primary Transfer event
  to: String! @parallel(step: 4)
  liquidity: BigDecimal! @parallel(step: 4)

  # populated from the Mint event
  sender: String @parallel(step: 4)
  amount0: BigDecimal @parallel(step: 4)
  amount1: BigDecimal @parallel(step: 4)
  logIndex: BigInt @parallel(step: 4)
  # derived amount based on available prices of tokens
  amountUSD: BigDecimal @parallel(step: 4)

  # optional fee fields, if a Transfer event is fired in _mintFee
  feeTo: String @parallel(step: 4)
  feeLiquidity: BigDecimal @parallel(step: 4)
}

type Burn @entity {
  # transaction hash + "-" + index in mints Transaction array
  id: ID!
  transaction: Transaction! @parallel(step: 4)
  timestamp: BigInt! @parallel(step: 4) # need this to pull recent txns for specific token or pair
  pair: Pair! @parallel(step: 4)
  token0: Token! @parallel(step: 4)
  token1: Token! @parallel(step: 4)

  # populated from the primary Transfer event
  liquidity: BigDecimal! @parallel(step: 4)

  # populated from the Burn event
  sender: String @parallel(step: 4)
  amount0: BigDecimal @parallel(step: 4)
  amount1: BigDecimal @parallel(step: 4)
  to: String @parallel(step: 4)
  logIndex: BigInt @parallel(step: 4)
  # derived amount based on available prices of tokens
  amountUSD: BigDecimal @parallel(step: 4)

  # mark uncomplete in BNB case
  needsComplete: Boolean! @parallel(step: 4)

  # optional fee fields, if a Transfer event is fired in _mintFee
  feeTo: String @parallel(step: 4)
  feeLiquidity: BigDecimal @parallel(step: 4)
}

type Swap @entity {
  # transaction hash + "-" + index in swaps Transaction array
  id: ID!
  transaction: Transaction!  @parallel(step: 4)
  timestamp: BigInt!  @parallel(step: 4) # need this to pull recent txns for specific token or pair
  pair: Pair!  @parallel(step: 4)
  token0: Token! @parallel(step: 4)
  token1: Token! @parallel(step: 4)

  # populated from the Swap event
  sender: String! @parallel(step: 4)
  from: String! @parallel(step: 4) # the EOA that initiated the txn
  amount0In: BigDecimal! @parallel(step: 4)
  amount1In: BigDecimal! @parallel(step: 4)
  amount0Out: BigDecimal! @parallel(step: 4)
  amount1Out: BigDecimal! @parallel(step: 4)
  to: String! @parallel(step: 4)
  logIndex: BigInt @parallel(step: 4)

  # derived info
  amountUSD: BigDecimal! @parallel(step: 4)
}

type PancakeDayData @entity {
  id: ID! # timestamp rounded to current day by dividing by 86400

  date: Int!  @parallel(step: 4)

  dailyVolumeBNB: BigDecimal! @parallel(step: 4, type: SUM)
  dailyVolumeUSD: BigDecimal! @parallel(step: 4, type: SUM)
  dailyVolumeUntracked: BigDecimal! @parallel(step: 4, type: SUM)

  totalVolumeBNB: BigDecimal! @parallel(step: 4, type: SUM)
  totalLiquidityBNB: BigDecimal! @parallel(step: 4)
  totalVolumeUSD: BigDecimal!  @parallel(step: 4, type: SUM)# Accumulate at each trade, not just calculated off whatever totalVolume is. making it more accurate as it is a live conversion
  totalLiquidityUSD: BigDecimal! @parallel(step: 4)

  totalTransactions: BigInt! @parallel(step: 4)
}

type PairHourData @entity {
  id: ID!

  hourStartUnix: Int! @parallel(step: 4) # unix timestamp for start of hour
  pair: Pair! @parallel(step: 4)

  # reserves
  reserve0: BigDecimal! @parallel(step: 4)
  reserve1: BigDecimal! @parallel(step: 4)

  # total supply for LP historical returns
  totalSupply: BigDecimal! @parallel(step: 4, type: SUM)

  # derived liquidity
  reserveUSD: BigDecimal!

  # volume stats
  hourlyVolumeToken0: BigDecimal!  @parallel(step: 4, type: SUM)
  hourlyVolumeToken1: BigDecimal!  @parallel(step: 4, type: SUM)
  hourlyVolumeUSD: BigDecimal!  @parallel(step: 4, type: SUM)
  hourlyTxns: BigInt!  @parallel(step: 4, type: SUM)
}

type PairDayData @entity {
  id: ID!

  date: Int! @parallel(step: 4)
  pairAddress: Pair! @parallel(step: 4)
  token0: Token! @parallel(step: 4)
  token1: Token! @parallel(step: 4)

  # reserves
  reserve0: BigDecimal! @parallel(step: 4)
  reserve1: BigDecimal! @parallel(step: 4)

  # total supply for LP historical returns
  totalSupply: BigDecimal! @parallel(step: 4, type: SUM)

  # derived liquidity
  reserveUSD: BigDecimal! @parallel(step: 4)

  # volume stats
  dailyVolumeToken0: BigDecimal! @parallel(step: 4, type: SUM)
  dailyVolumeToken1: BigDecimal! @parallel(step: 4, type: SUM)
  dailyVolumeUSD: BigDecimal! @parallel(step: 4, type: SUM)
  dailyTxns: BigInt! @parallel(step: 4, type: SUM)
}

type TokenDayData @entity {
  id: ID!

  date: Int! @parallel(step: 4)
  token: Token! @parallel(step: 4)

  # volume stats
  dailyVolumeToken: BigDecimal! @parallel(step: 4, type: SUM)
  dailyVolumeBNB: BigDecimal! @parallel(step: 4, type: SUM)
  dailyVolumeUSD: BigDecimal! @parallel(step: 4, type: SUM)
  dailyTxns: BigInt! @parallel(step: 4, type: SUM)

  # liquidity stats
  totalLiquidityToken: BigDecimal! @parallel(step: 4)
  totalLiquidityBNB: BigDecimal! @parallel(step: 4)
  totalLiquidityUSD: BigDecimal! @parallel(step: 4)

  # price stats
  priceUSD: BigDecimal! @parallel(step: 4)
}
`,
	Abis: map[string]string{
		"BEP20": `[
  {
    "constant": true,
    "inputs": [],
    "name": "name",
    "outputs": [
      {
        "name": "",
        "type": "string"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "_spender",
        "type": "address"
      },
      {
        "name": "_value",
        "type": "uint256"
      }
    ],
    "name": "approve",
    "outputs": [
      {
        "name": "",
        "type": "bool"
      }
    ],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "totalSupply",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "_from",
        "type": "address"
      },
      {
        "name": "_to",
        "type": "address"
      },
      {
        "name": "_value",
        "type": "uint256"
      }
    ],
    "name": "transferFrom",
    "outputs": [
      {
        "name": "",
        "type": "bool"
      }
    ],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "decimals",
    "outputs": [
      {
        "name": "",
        "type": "uint8"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "_owner",
        "type": "address"
      }
    ],
    "name": "balanceOf",
    "outputs": [
      {
        "name": "balance",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "symbol",
    "outputs": [
      {
        "name": "",
        "type": "string"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "_to",
        "type": "address"
      },
      {
        "name": "_value",
        "type": "uint256"
      }
    ],
    "name": "transfer",
    "outputs": [
      {
        "name": "",
        "type": "bool"
      }
    ],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "_owner",
        "type": "address"
      },
      {
        "name": "_spender",
        "type": "address"
      }
    ],
    "name": "allowance",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "payable": true,
    "stateMutability": "payable",
    "type": "fallback"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "name": "owner",
        "type": "address"
      },
      {
        "indexed": true,
        "name": "spender",
        "type": "address"
      },
      {
        "indexed": false,
        "name": "value",
        "type": "uint256"
      }
    ],
    "name": "Approval",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "name": "from",
        "type": "address"
      },
      {
        "indexed": true,
        "name": "to",
        "type": "address"
      },
      {
        "indexed": false,
        "name": "value",
        "type": "uint256"
      }
    ],
    "name": "Transfer",
    "type": "event"
  }
]
`,
		"BEP20NameBytes": `[
  {
    "constant": true,
    "inputs": [],
    "name": "name",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  }
]
`,
		"BEP20SymbolBytes": `[
  {
    "constant": true,
    "inputs": [],
    "name": "symbol",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  }
]
`,
		"Factory": `[
  {
    "inputs": [{ "internalType": "address", "name": "_feeToSetter", "type": "address" }],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "constructor"
  },
  {
    "anonymous": false,
    "inputs": [
      { "indexed": true, "internalType": "address", "name": "token0", "type": "address" },
      { "indexed": true, "internalType": "address", "name": "token1", "type": "address" },
      { "indexed": false, "internalType": "address", "name": "pair", "type": "address" },
      { "indexed": false, "internalType": "uint256", "name": "", "type": "uint256" }
    ],
    "name": "PairCreated",
    "type": "event"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "INIT_CODE_PAIR_HASH",
    "outputs": [{ "internalType": "bytes32", "name": "", "type": "bytes32" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [{ "internalType": "uint256", "name": "", "type": "uint256" }],
    "name": "allPairs",
    "outputs": [{ "internalType": "address", "name": "", "type": "address" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "allPairsLength",
    "outputs": [{ "internalType": "uint256", "name": "", "type": "uint256" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      { "internalType": "address", "name": "tokenA", "type": "address" },
      { "internalType": "address", "name": "tokenB", "type": "address" }
    ],
    "name": "createPair",
    "outputs": [{ "internalType": "address", "name": "pair", "type": "address" }],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "feeTo",
    "outputs": [{ "internalType": "address", "name": "", "type": "address" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "feeToSetter",
    "outputs": [{ "internalType": "address", "name": "", "type": "address" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      { "internalType": "address", "name": "", "type": "address" },
      { "internalType": "address", "name": "", "type": "address" }
    ],
    "name": "getPair",
    "outputs": [{ "internalType": "address", "name": "", "type": "address" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [{ "internalType": "address", "name": "_feeTo", "type": "address" }],
    "name": "setFeeTo",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [{ "internalType": "address", "name": "_feeToSetter", "type": "address" }],
    "name": "setFeeToSetter",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  }
]
`,
		"Pair": `[
  { "inputs": [], "payable": false, "stateMutability": "nonpayable", "type": "constructor" },
  {
    "anonymous": false,
    "inputs": [
      { "indexed": true, "internalType": "address", "name": "owner", "type": "address" },
      { "indexed": true, "internalType": "address", "name": "spender", "type": "address" },
      { "indexed": false, "internalType": "uint256", "name": "value", "type": "uint256" }
    ],
    "name": "Approval",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      { "indexed": true, "internalType": "address", "name": "sender", "type": "address" },
      { "indexed": false, "internalType": "uint256", "name": "amount0", "type": "uint256" },
      { "indexed": false, "internalType": "uint256", "name": "amount1", "type": "uint256" },
      { "indexed": true, "internalType": "address", "name": "to", "type": "address" }
    ],
    "name": "Burn",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      { "indexed": true, "internalType": "address", "name": "sender", "type": "address" },
      { "indexed": false, "internalType": "uint256", "name": "amount0", "type": "uint256" },
      { "indexed": false, "internalType": "uint256", "name": "amount1", "type": "uint256" }
    ],
    "name": "Mint",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      { "indexed": true, "internalType": "address", "name": "sender", "type": "address" },
      { "indexed": false, "internalType": "uint256", "name": "amount0In", "type": "uint256" },
      { "indexed": false, "internalType": "uint256", "name": "amount1In", "type": "uint256" },
      { "indexed": false, "internalType": "uint256", "name": "amount0Out", "type": "uint256" },
      { "indexed": false, "internalType": "uint256", "name": "amount1Out", "type": "uint256" },
      { "indexed": true, "internalType": "address", "name": "to", "type": "address" }
    ],
    "name": "Swap",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      { "indexed": false, "internalType": "uint112", "name": "reserve0", "type": "uint112" },
      { "indexed": false, "internalType": "uint112", "name": "reserve1", "type": "uint112" }
    ],
    "name": "Sync",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      { "indexed": true, "internalType": "address", "name": "from", "type": "address" },
      { "indexed": true, "internalType": "address", "name": "to", "type": "address" },
      { "indexed": false, "internalType": "uint256", "name": "value", "type": "uint256" }
    ],
    "name": "Transfer",
    "type": "event"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "DOMAIN_SEPARATOR",
    "outputs": [{ "internalType": "bytes32", "name": "", "type": "bytes32" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "MINIMUM_LIQUIDITY",
    "outputs": [{ "internalType": "uint256", "name": "", "type": "uint256" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "PERMIT_TYPEHASH",
    "outputs": [{ "internalType": "bytes32", "name": "", "type": "bytes32" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      { "internalType": "address", "name": "", "type": "address" },
      { "internalType": "address", "name": "", "type": "address" }
    ],
    "name": "allowance",
    "outputs": [{ "internalType": "uint256", "name": "", "type": "uint256" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      { "internalType": "address", "name": "spender", "type": "address" },
      { "internalType": "uint256", "name": "value", "type": "uint256" }
    ],
    "name": "approve",
    "outputs": [{ "internalType": "bool", "name": "", "type": "bool" }],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [{ "internalType": "address", "name": "", "type": "address" }],
    "name": "balanceOf",
    "outputs": [{ "internalType": "uint256", "name": "", "type": "uint256" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [{ "internalType": "address", "name": "to", "type": "address" }],
    "name": "burn",
    "outputs": [
      { "internalType": "uint256", "name": "amount0", "type": "uint256" },
      { "internalType": "uint256", "name": "amount1", "type": "uint256" }
    ],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "decimals",
    "outputs": [{ "internalType": "uint8", "name": "", "type": "uint8" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "factory",
    "outputs": [{ "internalType": "address", "name": "", "type": "address" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "getReserves",
    "outputs": [
      { "internalType": "uint112", "name": "_reserve0", "type": "uint112" },
      { "internalType": "uint112", "name": "_reserve1", "type": "uint112" },
      { "internalType": "uint32", "name": "_blockTimestampLast", "type": "uint32" }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      { "internalType": "address", "name": "_token0", "type": "address" },
      { "internalType": "address", "name": "_token1", "type": "address" }
    ],
    "name": "initialize",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "kLast",
    "outputs": [{ "internalType": "uint256", "name": "", "type": "uint256" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [{ "internalType": "address", "name": "to", "type": "address" }],
    "name": "mint",
    "outputs": [{ "internalType": "uint256", "name": "liquidity", "type": "uint256" }],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "name",
    "outputs": [{ "internalType": "string", "name": "", "type": "string" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [{ "internalType": "address", "name": "", "type": "address" }],
    "name": "nonces",
    "outputs": [{ "internalType": "uint256", "name": "", "type": "uint256" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      { "internalType": "address", "name": "owner", "type": "address" },
      { "internalType": "address", "name": "spender", "type": "address" },
      { "internalType": "uint256", "name": "value", "type": "uint256" },
      { "internalType": "uint256", "name": "deadline", "type": "uint256" },
      { "internalType": "uint8", "name": "v", "type": "uint8" },
      { "internalType": "bytes32", "name": "r", "type": "bytes32" },
      { "internalType": "bytes32", "name": "s", "type": "bytes32" }
    ],
    "name": "permit",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "price0CumulativeLast",
    "outputs": [{ "internalType": "uint256", "name": "", "type": "uint256" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "price1CumulativeLast",
    "outputs": [{ "internalType": "uint256", "name": "", "type": "uint256" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [{ "internalType": "address", "name": "to", "type": "address" }],
    "name": "skim",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      { "internalType": "uint256", "name": "amount0Out", "type": "uint256" },
      { "internalType": "uint256", "name": "amount1Out", "type": "uint256" },
      { "internalType": "address", "name": "to", "type": "address" },
      { "internalType": "bytes", "name": "data", "type": "bytes" }
    ],
    "name": "swap",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "symbol",
    "outputs": [{ "internalType": "string", "name": "", "type": "string" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [],
    "name": "sync",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "token0",
    "outputs": [{ "internalType": "address", "name": "", "type": "address" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "token1",
    "outputs": [{ "internalType": "address", "name": "", "type": "address" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "totalSupply",
    "outputs": [{ "internalType": "uint256", "name": "", "type": "uint256" }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      { "internalType": "address", "name": "to", "type": "address" },
      { "internalType": "uint256", "name": "value", "type": "uint256" }
    ],
    "name": "transfer",
    "outputs": [{ "internalType": "bool", "name": "", "type": "bool" }],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      { "internalType": "address", "name": "from", "type": "address" },
      { "internalType": "address", "name": "to", "type": "address" },
      { "internalType": "uint256", "name": "value", "type": "uint256" }
    ],
    "name": "transferFrom",
    "outputs": [{ "internalType": "bool", "name": "", "type": "bool" }],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  }
]
`,
	},
	New: func(base subgraph.Base) subgraph.Subgraph {
		return &Subgraph{
			Base: base,
		}
	},
}

type Subgraph struct {
	subgraph.Base
}

func (s Subgraph) Init() error {
	return nil
}

func (s Subgraph) LoadDynamicDataSources(blockNum uint64) error {
	return nil
}

func (s Subgraph) LogStatus() {
	panic("implement me")
}

// PancakeFactory
type PancakeFactory struct {
	graphnode.Base
	TotalPairs         graphnode.Int   `db:"total_pairs" csv:"total_pairs"`
	TotalTransactions  graphnode.Int   `db:"total_transactions" csv:"total_transactions"`
	TotalVolumeUSD     graphnode.Float `db:"total_volume_usd" csv:"total_volume_usd"`
	TotalVolumeBNB     graphnode.Float `db:"total_volume_bnb" csv:"total_volume_bnb"`
	UntrackedVolumeUSD graphnode.Float `db:"untracked_volume_usd" csv:"untracked_volume_usd"`
	TotalLiquidityUSD  graphnode.Float `db:"total_liquidity_usd" csv:"total_liquidity_usd"`
	TotalLiquidityBNB  graphnode.Float `db:"total_liquidity_bnb" csv:"total_liquidity_bnb"`
}

func NewPancakeFactory(id string) *PancakeFactory {
	return &PancakeFactory{
		Base:               graphnode.NewBase(id),
		TotalPairs:         IL(0),
		TotalTransactions:  IL(0),
		TotalVolumeUSD:     FL(0),
		TotalVolumeBNB:     FL(0),
		UntrackedVolumeUSD: FL(0),
		TotalLiquidityUSD:  FL(0),
		TotalLiquidityBNB:  FL(0),
	}
}
func (f *PancakeFactory) Default() {
	f.TotalPairs = IL(0)
	f.TotalTransactions = IL(0)
	f.TotalVolumeUSD = FL(0)
	f.TotalVolumeBNB = FL(0)
	f.UntrackedVolumeUSD = FL(0)
	f.TotalLiquidityUSD = FL(0)
	f.TotalLiquidityBNB = FL(0)
}
func (_ *PancakeFactory) SkipDBLookup() bool {
	return false
}
func (next *PancakeFactory) Merge(step int, cached *PancakeFactory) {
	if step == 2 {
		next.TotalPairs = graphnode.IntAdd(next.TotalPairs, cached.TotalPairs)
		if next.MutatedOnStep != 1 {
		}
	}
	if step == 5 {
		next.TotalTransactions = graphnode.IntAdd(next.TotalTransactions, cached.TotalTransactions)
		next.TotalVolumeUSD = graphnode.FloatAdd(next.TotalVolumeUSD, cached.TotalVolumeUSD)
		next.TotalVolumeBNB = graphnode.FloatAdd(next.TotalVolumeBNB, cached.TotalVolumeBNB)
		next.UntrackedVolumeUSD = graphnode.FloatAdd(next.UntrackedVolumeUSD, cached.UntrackedVolumeUSD)
		if next.MutatedOnStep != 4 {
			next.TotalLiquidityUSD = cached.TotalLiquidityUSD
			next.TotalLiquidityBNB = cached.TotalLiquidityBNB
		}
	}
}

// Bundle
type Bundle struct {
	graphnode.Base
	BnbPrice graphnode.Float `db:"bnb_price" csv:"bnb_price"`
}

func NewBundle(id string) *Bundle {
	return &Bundle{
		Base:     graphnode.NewBase(id),
		BnbPrice: FL(0),
	}
}
func (b *Bundle) Default() {
	b.BnbPrice = FL(0)
}

func (_ *Bundle) SkipDBLookup() bool {
	return false
}
func (next *Bundle) Merge(step int, cached *Bundle) {
	if step == 5 {
		if next.MutatedOnStep != 4 {
			next.BnbPrice = cached.BnbPrice
		}
	}
}

// Token
type Token struct {
	graphnode.Base
	Name               string           `db:"name" csv:"name"`
	Symbol             string           `db:"symbol" csv:"symbol"`
	Decimals           graphnode.Int    `db:"decimals" csv:"decimals"`
	TradeVolume        graphnode.Float  `db:"trade_volume" csv:"trade_volume"`
	TradeVolumeUSD     graphnode.Float  `db:"trade_volume_usd" csv:"trade_volume_usd"`
	UntrackedVolumeUSD graphnode.Float  `db:"untracked_volume_usd" csv:"untracked_volume_usd"`
	TotalTransactions  graphnode.Int    `db:"total_transactions" csv:"total_transactions"`
	TotalLiquidity     graphnode.Float  `db:"total_liquidity" csv:"total_liquidity"`
	DerivedBNB         *graphnode.Float `db:"derived_bnb,nullable" csv:"derived_bnb"`
	DerivedUSD         *graphnode.Float `db:"derived_usd,nullable" csv:"derived_usd"`
}

func NewToken(id string) *Token {
	return &Token{
		Base:               graphnode.NewBase(id),
		Decimals:           IL(0),
		TradeVolume:        FL(0),
		TradeVolumeUSD:     FL(0),
		UntrackedVolumeUSD: FL(0),
		TotalTransactions:  IL(0),
		TotalLiquidity:     FL(0),
	}
}

func (t *Token) Default() {
	t.Decimals = IL(0)
	t.TradeVolume = FL(0)
	t.TradeVolumeUSD = FL(0)
	t.UntrackedVolumeUSD = FL(0)
	t.TotalTransactions = IL(0)
	t.TotalLiquidity = FL(0)

}

func (_ *Token) SkipDBLookup() bool {
	return false
}
func (next *Token) Merge(step int, cached *Token) {
	if step == 2 {
		if next.MutatedOnStep != 1 {
			next.Name = cached.Name
			next.Symbol = cached.Symbol
			next.Decimals = cached.Decimals
		}
	}
	if step == 3 {
		if next.MutatedOnStep != 2 {
			next.DerivedBNB = cached.DerivedBNB
			next.DerivedUSD = cached.DerivedUSD
		}
	}
	if step == 5 {
		next.TradeVolume = graphnode.FloatAdd(next.TradeVolume, cached.TradeVolume)
		next.TradeVolumeUSD = graphnode.FloatAdd(next.TradeVolumeUSD, cached.TradeVolumeUSD)
		next.UntrackedVolumeUSD = graphnode.FloatAdd(next.UntrackedVolumeUSD, cached.UntrackedVolumeUSD)
		next.TotalTransactions = graphnode.IntAdd(next.TotalTransactions, cached.TotalTransactions)
		next.TotalLiquidity = graphnode.FloatAdd(next.TotalLiquidity, cached.TotalLiquidity)
		if next.MutatedOnStep != 4 {
		}
	}
}

// Pair
type Pair struct {
	graphnode.Base
	Name               string          `db:"name" csv:"name"`
	Token0             string          `db:"token_0" csv:"token_0"`
	Token1             string          `db:"token_1" csv:"token_1"`
	Reserve0           graphnode.Float `db:"reserve_0" csv:"reserve_0"`
	Reserve1           graphnode.Float `db:"reserve_1" csv:"reserve_1"`
	TotalSupply        graphnode.Float `db:"total_supply" csv:"total_supply"`
	ReserveBNB         graphnode.Float `db:"reserve_bnb" csv:"reserve_bnb"`
	ReserveUSD         graphnode.Float `db:"reserve_usd" csv:"reserve_usd"`
	TrackedReserveBNB  graphnode.Float `db:"tracked_reserve_bnb" csv:"tracked_reserve_bnb"`
	Token0Price        graphnode.Float `db:"token_0_price" csv:"token_0_price"`
	Token1Price        graphnode.Float `db:"token_1_price" csv:"token_1_price"`
	VolumeToken0       graphnode.Float `db:"volume_token_0" csv:"volume_token_0"`
	VolumeToken1       graphnode.Float `db:"volume_token_1" csv:"volume_token_1"`
	VolumeUSD          graphnode.Float `db:"volume_usd" csv:"volume_usd"`
	UntrackedVolumeUSD graphnode.Float `db:"untracked_volume_usd" csv:"untracked_volume_usd"`
	TotalTransactions  graphnode.Int   `db:"total_transactions" csv:"total_transactions"`
	Block              graphnode.Int   `db:"block" csv:"block"`
	Timestamp          graphnode.Int   `db:"timestamp" csv:"timestamp"`
}

func NewPair(id string) *Pair {
	return &Pair{
		Base:               graphnode.NewBase(id),
		Reserve0:           FL(0),
		Reserve1:           FL(0),
		TotalSupply:        FL(0),
		ReserveBNB:         FL(0),
		ReserveUSD:         FL(0),
		TrackedReserveBNB:  FL(0),
		Token0Price:        FL(0),
		Token1Price:        FL(0),
		VolumeToken0:       FL(0),
		VolumeToken1:       FL(0),
		VolumeUSD:          FL(0),
		UntrackedVolumeUSD: FL(0),
		TotalTransactions:  IL(0),
		Block:              IL(0),
		Timestamp:          IL(0),
	}
}

func (p *Pair) Default() {
	p.Reserve0 = FL(0)
	p.Reserve1 = FL(0)
	p.TotalSupply = FL(0)
	p.ReserveBNB = FL(0)
	p.ReserveUSD = FL(0)
	p.TrackedReserveBNB = FL(0)
	p.Token0Price = FL(0)
	p.Token1Price = FL(0)
	p.VolumeToken0 = FL(0)
	p.VolumeToken1 = FL(0)
	p.VolumeUSD = FL(0)
	p.UntrackedVolumeUSD = FL(0)
	p.TotalTransactions = IL(0)
	p.Block = IL(0)
	p.Timestamp = IL(0)
}

func (_ *Pair) SkipDBLookup() bool {
	return false
}
func (next *Pair) Merge(step int, cached *Pair) {
	if step == 2 {
		if next.MutatedOnStep != 1 {
			next.Name = cached.Name
			next.Token0 = cached.Token0
			next.Token1 = cached.Token1
			next.Block = cached.Block
			next.Timestamp = cached.Timestamp
		}
	}
	if step == 3 {
		if next.MutatedOnStep != 2 {
			next.Reserve0 = cached.Reserve0
			next.Reserve1 = cached.Reserve1
			next.Token0Price = cached.Token0Price
			next.Token1Price = cached.Token1Price
		}
	}
	if step == 4 {
		if next.MutatedOnStep != 3 {
			next.ReserveBNB = cached.ReserveBNB
			next.ReserveUSD = cached.ReserveUSD
		}
	}
	if step == 5 {
		next.TotalSupply = graphnode.FloatAdd(next.TotalSupply, cached.TotalSupply)
		next.VolumeToken0 = graphnode.FloatAdd(next.VolumeToken0, cached.VolumeToken0)
		next.VolumeToken1 = graphnode.FloatAdd(next.VolumeToken1, cached.VolumeToken1)
		next.VolumeUSD = graphnode.FloatAdd(next.VolumeUSD, cached.VolumeUSD)
		next.UntrackedVolumeUSD = graphnode.FloatAdd(next.UntrackedVolumeUSD, cached.UntrackedVolumeUSD)
		next.TotalTransactions = graphnode.IntAdd(next.TotalTransactions, cached.TotalTransactions)
		if next.MutatedOnStep != 4 {
		}
	}
}

// Transaction
type Transaction struct {
	graphnode.Base
	Block     graphnode.Int              `db:"block" csv:"block"`
	Timestamp graphnode.Int              `db:"timestamp" csv:"timestamp"`
	Mints     graphnode.LocalStringArray `db:"mints,nullable" csv:"mints"`
	Burns     graphnode.LocalStringArray `db:"burns,nullable" csv:"burns"`
	Swaps     graphnode.LocalStringArray `db:"swaps,nullable" csv:"swaps"`
}

func NewTransaction(id string) *Transaction {
	return &Transaction{
		Base:      graphnode.NewBase(id),
		Block:     IL(0),
		Timestamp: IL(0),
	}
}
func (t *Transaction) Default() {
	t.Block = IL(0)
	t.Timestamp = IL(0)
}
func (_ *Transaction) SkipDBLookup() bool {
	return true
}
func (next *Transaction) Merge(step int, cached *Transaction) {
	if step == 5 {
		if next.MutatedOnStep != 4 {
			next.Block = cached.Block
			next.Timestamp = cached.Timestamp
		}
	}
}

// Mint
type Mint struct {
	graphnode.Base
	Transaction  string           `db:"transaction" csv:"transaction"`
	Timestamp    graphnode.Int    `db:"timestamp" csv:"timestamp"`
	Pair         string           `db:"pair" csv:"pair"`
	Token0       string           `db:"token_0" csv:"token_0"`
	Token1       string           `db:"token_1" csv:"token_1"`
	To           string           `db:"to" csv:"to"`
	Liquidity    graphnode.Float  `db:"liquidity" csv:"liquidity"`
	Sender       *string          `db:"sender,nullable" csv:"sender"`
	Amount0      *graphnode.Float `db:"amount_0,nullable" csv:"amount_0"`
	Amount1      *graphnode.Float `db:"amount_1,nullable" csv:"amount_1"`
	LogIndex     *graphnode.Int   `db:"log_index,nullable" csv:"log_index"`
	AmountUSD    *graphnode.Float `db:"amount_usd,nullable" csv:"amount_usd"`
	FeeTo        *string          `db:"fee_to,nullable" csv:"fee_to"`
	FeeLiquidity *graphnode.Float `db:"fee_liquidity,nullable" csv:"fee_liquidity"`
}

func NewMint(id string) *Mint {
	return &Mint{
		Base:      graphnode.NewBase(id),
		Timestamp: IL(0),
		Liquidity: FL(0),
	}
}

func (m *Mint) Default() {
	m.Timestamp = IL(0)
	m.Liquidity = FL(0)
}

func (_ *Mint) SkipDBLookup() bool {
	return false
}
func (next *Mint) Merge(step int, cached *Mint) {
	if step == 5 {
		if next.MutatedOnStep != 4 {
			next.Transaction = cached.Transaction
			next.Timestamp = cached.Timestamp
			next.Pair = cached.Pair
			next.Token0 = cached.Token0
			next.Token1 = cached.Token1
			next.To = cached.To
			next.Liquidity = cached.Liquidity
			next.Sender = cached.Sender
			next.Amount0 = cached.Amount0
			next.Amount1 = cached.Amount1
			next.LogIndex = cached.LogIndex
			next.AmountUSD = cached.AmountUSD
			next.FeeTo = cached.FeeTo
			next.FeeLiquidity = cached.FeeLiquidity
		}
	}
}

// Burn
type Burn struct {
	graphnode.Base
	Transaction   string           `db:"transaction" csv:"transaction"`
	Timestamp     graphnode.Int    `db:"timestamp" csv:"timestamp"`
	Pair          string           `db:"pair" csv:"pair"`
	Token0        string           `db:"token_0" csv:"token_0"`
	Token1        string           `db:"token_1" csv:"token_1"`
	Liquidity     graphnode.Float  `db:"liquidity" csv:"liquidity"`
	Sender        *string          `db:"sender,nullable" csv:"sender"`
	Amount0       *graphnode.Float `db:"amount_0,nullable" csv:"amount_0"`
	Amount1       *graphnode.Float `db:"amount_1,nullable" csv:"amount_1"`
	To            *string          `db:"to,nullable" csv:"to"`
	LogIndex      *graphnode.Int   `db:"log_index,nullable" csv:"log_index"`
	AmountUSD     *graphnode.Float `db:"amount_usd,nullable" csv:"amount_usd"`
	NeedsComplete graphnode.Bool   `db:"needs_complete" csv:"needs_complete"`
	FeeTo         *string          `db:"fee_to,nullable" csv:"fee_to"`
	FeeLiquidity  *graphnode.Float `db:"fee_liquidity,nullable" csv:"fee_liquidity"`
}

func NewBurn(id string) *Burn {
	return &Burn{
		Base:      graphnode.NewBase(id),
		Timestamp: IL(0),
		Liquidity: FL(0),
	}
}

func (b *Burn) Default() {
	b.Timestamp = IL(0)
	b.Liquidity = FL(0)
}

func (_ *Burn) SkipDBLookup() bool {
	return false
}
func (next *Burn) Merge(step int, cached *Burn) {
	if step == 5 {
		if next.MutatedOnStep != 4 {
			next.Transaction = cached.Transaction
			next.Timestamp = cached.Timestamp
			next.Pair = cached.Pair
			next.Token0 = cached.Token0
			next.Token1 = cached.Token1
			next.Liquidity = cached.Liquidity
			next.Sender = cached.Sender
			next.Amount0 = cached.Amount0
			next.Amount1 = cached.Amount1
			next.To = cached.To
			next.LogIndex = cached.LogIndex
			next.AmountUSD = cached.AmountUSD
			next.NeedsComplete = cached.NeedsComplete
			next.FeeTo = cached.FeeTo
			next.FeeLiquidity = cached.FeeLiquidity
		}
	}
}

// Swap
type Swap struct {
	graphnode.Base
	Transaction string          `db:"transaction" csv:"transaction"`
	Timestamp   graphnode.Int   `db:"timestamp" csv:"timestamp"`
	Pair        string          `db:"pair" csv:"pair"`
	Token0      string          `db:"token_0" csv:"token_0"`
	Token1      string          `db:"token_1" csv:"token_1"`
	Sender      string          `db:"sender" csv:"sender"`
	From        string          `db:"from" csv:"from"`
	Amount0In   graphnode.Float `db:"amount_0_in" csv:"amount_0_in"`
	Amount1In   graphnode.Float `db:"amount_1_in" csv:"amount_1_in"`
	Amount0Out  graphnode.Float `db:"amount_0_out" csv:"amount_0_out"`
	Amount1Out  graphnode.Float `db:"amount_1_out" csv:"amount_1_out"`
	To          string          `db:"to" csv:"to"`
	LogIndex    *graphnode.Int  `db:"log_index,nullable" csv:"log_index"`
	AmountUSD   graphnode.Float `db:"amount_usd" csv:"amount_usd"`
}

func NewSwap(id string) *Swap {
	return &Swap{
		Base:       graphnode.NewBase(id),
		Timestamp:  IL(0),
		Amount0In:  FL(0),
		Amount1In:  FL(0),
		Amount0Out: FL(0),
		Amount1Out: FL(0),
		AmountUSD:  FL(0),
	}
}
func (s *Swap) Default() {
	s.Timestamp = IL(0)
	s.Amount0In = FL(0)
	s.Amount1In = FL(0)
	s.Amount0Out = FL(0)
	s.Amount1Out = FL(0)
	s.AmountUSD = FL(0)
}

func (_ *Swap) SkipDBLookup() bool {
	return false
}
func (next *Swap) Merge(step int, cached *Swap) {
	if step == 5 {
		if next.MutatedOnStep != 4 {
			next.Transaction = cached.Transaction
			next.Timestamp = cached.Timestamp
			next.Pair = cached.Pair
			next.Token0 = cached.Token0
			next.Token1 = cached.Token1
			next.Sender = cached.Sender
			next.From = cached.From
			next.Amount0In = cached.Amount0In
			next.Amount1In = cached.Amount1In
			next.Amount0Out = cached.Amount0Out
			next.Amount1Out = cached.Amount1Out
			next.To = cached.To
			next.LogIndex = cached.LogIndex
			next.AmountUSD = cached.AmountUSD
		}
	}
}

// PancakeDayData
type PancakeDayData struct {
	graphnode.Base
	Date                 int64           `db:"date" csv:"date"`
	DailyVolumeBNB       graphnode.Float `db:"daily_volume_bnb" csv:"daily_volume_bnb"`
	DailyVolumeUSD       graphnode.Float `db:"daily_volume_usd" csv:"daily_volume_usd"`
	DailyVolumeUntracked graphnode.Float `db:"daily_volume_untracked" csv:"daily_volume_untracked"`
	TotalVolumeBNB       graphnode.Float `db:"total_volume_bnb" csv:"total_volume_bnb"`
	TotalLiquidityBNB    graphnode.Float `db:"total_liquidity_bnb" csv:"total_liquidity_bnb"`
	TotalVolumeUSD       graphnode.Float `db:"total_volume_usd" csv:"total_volume_usd"`
	TotalLiquidityUSD    graphnode.Float `db:"total_liquidity_usd" csv:"total_liquidity_usd"`
	TotalTransactions    graphnode.Int   `db:"total_transactions" csv:"total_transactions"`
}

func NewPancakeDayData(id string) *PancakeDayData {
	return &PancakeDayData{
		Base:                 graphnode.NewBase(id),
		DailyVolumeBNB:       FL(0),
		DailyVolumeUSD:       FL(0),
		DailyVolumeUntracked: FL(0),
		TotalVolumeBNB:       FL(0),
		TotalLiquidityBNB:    FL(0),
		TotalVolumeUSD:       FL(0),
		TotalLiquidityUSD:    FL(0),
		TotalTransactions:    IL(0),
	}
}

func (d *PancakeDayData) Default() {
	d.DailyVolumeBNB = FL(0)
	d.DailyVolumeUSD = FL(0)
	d.DailyVolumeUntracked = FL(0)
	d.TotalVolumeBNB = FL(0)
	d.TotalLiquidityBNB = FL(0)
	d.TotalVolumeUSD = FL(0)
	d.TotalLiquidityUSD = FL(0)
	d.TotalTransactions = IL(0)

}

func (_ *PancakeDayData) SkipDBLookup() bool {
	return false
}
func (next *PancakeDayData) Merge(step int, cached *PancakeDayData) {
	if step == 5 {
		next.DailyVolumeBNB = graphnode.FloatAdd(next.DailyVolumeBNB, cached.DailyVolumeBNB)
		next.DailyVolumeUSD = graphnode.FloatAdd(next.DailyVolumeUSD, cached.DailyVolumeUSD)
		next.DailyVolumeUntracked = graphnode.FloatAdd(next.DailyVolumeUntracked, cached.DailyVolumeUntracked)
		next.TotalVolumeBNB = graphnode.FloatAdd(next.TotalVolumeBNB, cached.TotalVolumeBNB)
		next.TotalVolumeUSD = graphnode.FloatAdd(next.TotalVolumeUSD, cached.TotalVolumeUSD)
		if next.MutatedOnStep != 4 {
			next.Date = cached.Date
			next.TotalLiquidityBNB = cached.TotalLiquidityBNB
			next.TotalLiquidityUSD = cached.TotalLiquidityUSD
			next.TotalTransactions = cached.TotalTransactions
		}
	}
}

// PairHourData
type PairHourData struct {
	graphnode.Base
	HourStartUnix      int64           `db:"hour_start_unix" csv:"hour_start_unix"`
	Pair               string          `db:"pair" csv:"pair"`
	Reserve0           graphnode.Float `db:"reserve_0" csv:"reserve_0"`
	Reserve1           graphnode.Float `db:"reserve_1" csv:"reserve_1"`
	TotalSupply        graphnode.Float `db:"total_supply" csv:"total_supply"`
	ReserveUSD         graphnode.Float `db:"reserve_usd" csv:"reserve_usd"`
	HourlyVolumeToken0 graphnode.Float `db:"hourly_volume_token_0" csv:"hourly_volume_token_0"`
	HourlyVolumeToken1 graphnode.Float `db:"hourly_volume_token_1" csv:"hourly_volume_token_1"`
	HourlyVolumeUSD    graphnode.Float `db:"hourly_volume_usd" csv:"hourly_volume_usd"`
	HourlyTxns         graphnode.Int   `db:"hourly_txns" csv:"hourly_txns"`
}

func NewPairHourData(id string) *PairHourData {
	return &PairHourData{
		Base:               graphnode.NewBase(id),
		Reserve0:           FL(0),
		Reserve1:           FL(0),
		TotalSupply:        FL(0),
		ReserveUSD:         FL(0),
		HourlyVolumeToken0: FL(0),
		HourlyVolumeToken1: FL(0),
		HourlyVolumeUSD:    FL(0),
		HourlyTxns:         IL(0),
	}
}

func (d *PairHourData) Default() {
	d.Reserve0 = FL(0)
	d.Reserve1 = FL(0)
	d.TotalSupply = FL(0)
	d.ReserveUSD = FL(0)
	d.HourlyVolumeToken0 = FL(0)
	d.HourlyVolumeToken1 = FL(0)
	d.HourlyVolumeUSD = FL(0)
	d.HourlyTxns = IL(0)

}

func (_ *PairHourData) SkipDBLookup() bool {
	return false
}
func (next *PairHourData) Merge(step int, cached *PairHourData) {
	if step == 5 {
		next.TotalSupply = graphnode.FloatAdd(next.TotalSupply, cached.TotalSupply)
		next.HourlyVolumeToken0 = graphnode.FloatAdd(next.HourlyVolumeToken0, cached.HourlyVolumeToken0)
		next.HourlyVolumeToken1 = graphnode.FloatAdd(next.HourlyVolumeToken1, cached.HourlyVolumeToken1)
		next.HourlyVolumeUSD = graphnode.FloatAdd(next.HourlyVolumeUSD, cached.HourlyVolumeUSD)
		next.HourlyTxns = graphnode.IntAdd(next.HourlyTxns, cached.HourlyTxns)
		if next.MutatedOnStep != 4 {
			next.HourStartUnix = cached.HourStartUnix
			next.Pair = cached.Pair
			next.Reserve0 = cached.Reserve0
			next.Reserve1 = cached.Reserve1
		}
	}
}

// PairDayData
type PairDayData struct {
	graphnode.Base
	Date              int64           `db:"date" csv:"date"`
	PairAddress       string          `db:"pair_address" csv:"pair_address"`
	Token0            string          `db:"token_0" csv:"token_0"`
	Token1            string          `db:"token_1" csv:"token_1"`
	Reserve0          graphnode.Float `db:"reserve_0" csv:"reserve_0"`
	Reserve1          graphnode.Float `db:"reserve_1" csv:"reserve_1"`
	TotalSupply       graphnode.Float `db:"total_supply" csv:"total_supply"`
	ReserveUSD        graphnode.Float `db:"reserve_usd" csv:"reserve_usd"`
	DailyVolumeToken0 graphnode.Float `db:"daily_volume_token_0" csv:"daily_volume_token_0"`
	DailyVolumeToken1 graphnode.Float `db:"daily_volume_token_1" csv:"daily_volume_token_1"`
	DailyVolumeUSD    graphnode.Float `db:"daily_volume_usd" csv:"daily_volume_usd"`
	DailyTxns         graphnode.Int   `db:"daily_txns" csv:"daily_txns"`
}

func NewPairDayData(id string) *PairDayData {
	return &PairDayData{
		Base:              graphnode.NewBase(id),
		Reserve0:          FL(0),
		Reserve1:          FL(0),
		TotalSupply:       FL(0),
		ReserveUSD:        FL(0),
		DailyVolumeToken0: FL(0),
		DailyVolumeToken1: FL(0),
		DailyVolumeUSD:    FL(0),
		DailyTxns:         IL(0),
	}
}

func (d *PairDayData) Default() {
	d.Reserve0 = FL(0)
	d.Reserve1 = FL(0)
	d.TotalSupply = FL(0)
	d.ReserveUSD = FL(0)
	d.DailyVolumeToken0 = FL(0)
	d.DailyVolumeToken1 = FL(0)
	d.DailyVolumeUSD = FL(0)
	d.DailyTxns = IL(0)

}

func (_ *PairDayData) SkipDBLookup() bool {
	return false
}
func (next *PairDayData) Merge(step int, cached *PairDayData) {
	if step == 5 {
		next.TotalSupply = graphnode.FloatAdd(next.TotalSupply, cached.TotalSupply)
		next.DailyVolumeToken0 = graphnode.FloatAdd(next.DailyVolumeToken0, cached.DailyVolumeToken0)
		next.DailyVolumeToken1 = graphnode.FloatAdd(next.DailyVolumeToken1, cached.DailyVolumeToken1)
		next.DailyVolumeUSD = graphnode.FloatAdd(next.DailyVolumeUSD, cached.DailyVolumeUSD)
		next.DailyTxns = graphnode.IntAdd(next.DailyTxns, cached.DailyTxns)
		if next.MutatedOnStep != 4 {
			next.Date = cached.Date
			next.PairAddress = cached.PairAddress
			next.Token0 = cached.Token0
			next.Token1 = cached.Token1
			next.Reserve0 = cached.Reserve0
			next.Reserve1 = cached.Reserve1
			next.ReserveUSD = cached.ReserveUSD
		}
	}
}

// TokenDayData
type TokenDayData struct {
	graphnode.Base
	Date                int64           `db:"date" csv:"date"`
	Token               string          `db:"token" csv:"token"`
	DailyVolumeToken    graphnode.Float `db:"daily_volume_token" csv:"daily_volume_token"`
	DailyVolumeBNB      graphnode.Float `db:"daily_volume_bnb" csv:"daily_volume_bnb"`
	DailyVolumeUSD      graphnode.Float `db:"daily_volume_usd" csv:"daily_volume_usd"`
	DailyTxns           graphnode.Int   `db:"daily_txns" csv:"daily_txns"`
	TotalLiquidityToken graphnode.Float `db:"total_liquidity_token" csv:"total_liquidity_token"`
	TotalLiquidityBNB   graphnode.Float `db:"total_liquidity_bnb" csv:"total_liquidity_bnb"`
	TotalLiquidityUSD   graphnode.Float `db:"total_liquidity_usd" csv:"total_liquidity_usd"`
	PriceUSD            graphnode.Float `db:"price_usd" csv:"price_usd"`
}

func NewTokenDayData(id string) *TokenDayData {
	return &TokenDayData{
		Base:                graphnode.NewBase(id),
		DailyVolumeToken:    FL(0),
		DailyVolumeBNB:      FL(0),
		DailyVolumeUSD:      FL(0),
		DailyTxns:           IL(0),
		TotalLiquidityToken: FL(0),
		TotalLiquidityBNB:   FL(0),
		TotalLiquidityUSD:   FL(0),
		PriceUSD:            FL(0),
	}
}

func (d *TokenDayData) Default() {
	d.DailyVolumeToken = FL(0)
	d.DailyVolumeBNB = FL(0)
	d.DailyVolumeUSD = FL(0)
	d.DailyTxns = IL(0)
	d.TotalLiquidityToken = FL(0)
	d.TotalLiquidityBNB = FL(0)
	d.TotalLiquidityUSD = FL(0)
	d.PriceUSD = FL(0)

}

func (_ *TokenDayData) SkipDBLookup() bool {
	return false
}
func (next *TokenDayData) Merge(step int, cached *TokenDayData) {
	if step == 5 {
		next.DailyVolumeToken = graphnode.FloatAdd(next.DailyVolumeToken, cached.DailyVolumeToken)
		next.DailyVolumeBNB = graphnode.FloatAdd(next.DailyVolumeBNB, cached.DailyVolumeBNB)
		next.DailyVolumeUSD = graphnode.FloatAdd(next.DailyVolumeUSD, cached.DailyVolumeUSD)
		next.DailyTxns = graphnode.IntAdd(next.DailyTxns, cached.DailyTxns)
		if next.MutatedOnStep != 4 {
			next.Date = cached.Date
			next.Token = cached.Token
			next.TotalLiquidityToken = cached.TotalLiquidityToken
			next.TotalLiquidityBNB = cached.TotalLiquidityBNB
			next.TotalLiquidityUSD = cached.TotalLiquidityUSD
			next.PriceUSD = cached.PriceUSD
		}
	}
}

func codecLogToEthLog(l *pbeth.Log, idx uint32) *eth.Log {
	return &eth.Log{
		Address:    l.Address,
		Topics:     l.Topics,
		Data:       l.Data,
		Index:      l.Index,
		BlockIndex: idx,
	}
}

type DDL struct {
	createTables map[string]string
	indexes      map[string][]*index
	schemaSetup  string
}

var ddl *DDL

type index struct {
	createStatement string
	dropStatement   string
}

var createTables = map[string]string{}
var indexes = map[string][]*index{}

func init() {
	ddl = &DDL{
		createTables: map[string]string{},
		indexes:      map[string][]*index{},
	}

	Definition.DDL = ddl

	ddl.createTables["pancake_factory"] = `
create table if not exists %%SCHEMA%%.pancake_factory
(
	id text not null,

	"total_pairs" numeric not null,

	"total_transactions" numeric not null,

	"total_volume_usd" numeric not null,

	"total_volume_bnb" numeric not null,

	"untracked_volume_usd" numeric not null,

	"total_liquidity_usd" numeric not null,

	"total_liquidity_bnb" numeric not null,

	vid bigserial not null constraint pancake_factory_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.pancake_factory owner to graph;
alter sequence %%SCHEMA%%.pancake_factory_vid_seq owned by %%SCHEMA%%.pancake_factory.vid;
alter table only %%SCHEMA%%.pancake_factory alter column vid SET DEFAULT nextval('%%SCHEMA%%.pancake_factory_vid_seq'::regclass);
`

	ddl.createTables["bundle"] = `
create table if not exists %%SCHEMA%%.bundle
(
	id text not null,

	"bnb_price" numeric not null,

	vid bigserial not null constraint bundle_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.bundle owner to graph;
alter sequence %%SCHEMA%%.bundle_vid_seq owned by %%SCHEMA%%.bundle.vid;
alter table only %%SCHEMA%%.bundle alter column vid SET DEFAULT nextval('%%SCHEMA%%.bundle_vid_seq'::regclass);
`

	ddl.createTables["token"] = `
create table if not exists %%SCHEMA%%.token
(
	id text not null,

	"name" text not null,

	"symbol" text not null,

	"decimals" numeric not null,

	"trade_volume" numeric not null,

	"trade_volume_usd" numeric not null,

	"untracked_volume_usd" numeric not null,

	"total_transactions" numeric not null,

	"total_liquidity" numeric not null,

	"derived_bnb" numeric,

	"derived_usd" numeric,

	vid bigserial not null constraint token_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.token owner to graph;
alter sequence %%SCHEMA%%.token_vid_seq owned by %%SCHEMA%%.token.vid;
alter table only %%SCHEMA%%.token alter column vid SET DEFAULT nextval('%%SCHEMA%%.token_vid_seq'::regclass);
`

	ddl.createTables["pair"] = `
create table if not exists %%SCHEMA%%.pair
(
	id text not null,

	"name" text not null,

	"token_0" text not null,

	"token_1" text not null,

	"reserve_0" numeric not null,

	"reserve_1" numeric not null,

	"total_supply" numeric not null,

	"reserve_bnb" numeric not null,

	"reserve_usd" numeric not null,

	"tracked_reserve_bnb" numeric not null,

	"token_0_price" numeric not null,

	"token_1_price" numeric not null,

	"volume_token_0" numeric not null,

	"volume_token_1" numeric not null,

	"volume_usd" numeric not null,

	"untracked_volume_usd" numeric not null,

	"total_transactions" numeric not null,

	"block" numeric not null,

	"timestamp" numeric not null,

	vid bigserial not null constraint pair_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.pair owner to graph;
alter sequence %%SCHEMA%%.pair_vid_seq owned by %%SCHEMA%%.pair.vid;
alter table only %%SCHEMA%%.pair alter column vid SET DEFAULT nextval('%%SCHEMA%%.pair_vid_seq'::regclass);
`

	ddl.createTables["transaction"] = `
create table if not exists %%SCHEMA%%.transaction
(
	id text not null,

	"block" numeric not null,

	"timestamp" numeric not null,

	"mints" text[],

	"burns" text[],

	"swaps" text[],

	vid bigserial not null constraint transaction_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.transaction owner to graph;
alter sequence %%SCHEMA%%.transaction_vid_seq owned by %%SCHEMA%%.transaction.vid;
alter table only %%SCHEMA%%.transaction alter column vid SET DEFAULT nextval('%%SCHEMA%%.transaction_vid_seq'::regclass);
`

	ddl.createTables["mint"] = `
create table if not exists %%SCHEMA%%.mint
(
	id text not null,

	"transaction" text not null,

	"timestamp" numeric not null,

	"pair" text not null,

	"token_0" text not null,

	"token_1" text not null,

	"to" text not null,

	"liquidity" numeric not null,

	"sender" text,

	"amount_0" numeric,

	"amount_1" numeric,

	"log_index" numeric,

	"amount_usd" numeric,

	"fee_to" text,

	"fee_liquidity" numeric,

	vid bigserial not null constraint mint_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.mint owner to graph;
alter sequence %%SCHEMA%%.mint_vid_seq owned by %%SCHEMA%%.mint.vid;
alter table only %%SCHEMA%%.mint alter column vid SET DEFAULT nextval('%%SCHEMA%%.mint_vid_seq'::regclass);
`

	ddl.createTables["burn"] = `
create table if not exists %%SCHEMA%%.burn
(
	id text not null,

	"transaction" text not null,

	"timestamp" numeric not null,

	"pair" text not null,

	"token_0" text not null,

	"token_1" text not null,

	"liquidity" numeric not null,

	"sender" text,

	"amount_0" numeric,

	"amount_1" numeric,

	"to" text,

	"log_index" numeric,

	"amount_usd" numeric,

	"needs_complete" boolean not null,

	"fee_to" text,

	"fee_liquidity" numeric,

	vid bigserial not null constraint burn_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.burn owner to graph;
alter sequence %%SCHEMA%%.burn_vid_seq owned by %%SCHEMA%%.burn.vid;
alter table only %%SCHEMA%%.burn alter column vid SET DEFAULT nextval('%%SCHEMA%%.burn_vid_seq'::regclass);
`

	ddl.createTables["swap"] = `
create table if not exists %%SCHEMA%%.swap
(
	id text not null,

	"transaction" text not null,

	"timestamp" numeric not null,

	"pair" text not null,

	"token_0" text not null,

	"token_1" text not null,

	"sender" text not null,

	"from" text not null,

	"amount_0_in" numeric not null,

	"amount_1_in" numeric not null,

	"amount_0_out" numeric not null,

	"amount_1_out" numeric not null,

	"to" text not null,

	"log_index" numeric,

	"amount_usd" numeric not null,

	vid bigserial not null constraint swap_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.swap owner to graph;
alter sequence %%SCHEMA%%.swap_vid_seq owned by %%SCHEMA%%.swap.vid;
alter table only %%SCHEMA%%.swap alter column vid SET DEFAULT nextval('%%SCHEMA%%.swap_vid_seq'::regclass);
`

	ddl.createTables["pancake_day_data"] = `
create table if not exists %%SCHEMA%%.pancake_day_data
(
	id text not null,

	"date" numeric not null,

	"daily_volume_bnb" numeric not null,

	"daily_volume_usd" numeric not null,

	"daily_volume_untracked" numeric not null,

	"total_volume_bnb" numeric not null,

	"total_liquidity_bnb" numeric not null,

	"total_volume_usd" numeric not null,

	"total_liquidity_usd" numeric not null,

	"total_transactions" numeric not null,

	vid bigserial not null constraint pancake_day_data_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.pancake_day_data owner to graph;
alter sequence %%SCHEMA%%.pancake_day_data_vid_seq owned by %%SCHEMA%%.pancake_day_data.vid;
alter table only %%SCHEMA%%.pancake_day_data alter column vid SET DEFAULT nextval('%%SCHEMA%%.pancake_day_data_vid_seq'::regclass);
`

	ddl.createTables["pair_hour_data"] = `
create table if not exists %%SCHEMA%%.pair_hour_data
(
	id text not null,

	"hour_start_unix" numeric not null,

	"pair" text not null,

	"reserve_0" numeric not null,

	"reserve_1" numeric not null,

	"total_supply" numeric not null,

	"reserve_usd" numeric not null,

	"hourly_volume_token_0" numeric not null,

	"hourly_volume_token_1" numeric not null,

	"hourly_volume_usd" numeric not null,

	"hourly_txns" numeric not null,

	vid bigserial not null constraint pair_hour_data_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.pair_hour_data owner to graph;
alter sequence %%SCHEMA%%.pair_hour_data_vid_seq owned by %%SCHEMA%%.pair_hour_data.vid;
alter table only %%SCHEMA%%.pair_hour_data alter column vid SET DEFAULT nextval('%%SCHEMA%%.pair_hour_data_vid_seq'::regclass);
`

	ddl.createTables["pair_day_data"] = `
create table if not exists %%SCHEMA%%.pair_day_data
(
	id text not null,

	"date" numeric not null,

	"pair_address" text not null,

	"token_0" text not null,

	"token_1" text not null,

	"reserve_0" numeric not null,

	"reserve_1" numeric not null,

	"total_supply" numeric not null,

	"reserve_usd" numeric not null,

	"daily_volume_token_0" numeric not null,

	"daily_volume_token_1" numeric not null,

	"daily_volume_usd" numeric not null,

	"daily_txns" numeric not null,

	vid bigserial not null constraint pair_day_data_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.pair_day_data owner to graph;
alter sequence %%SCHEMA%%.pair_day_data_vid_seq owned by %%SCHEMA%%.pair_day_data.vid;
alter table only %%SCHEMA%%.pair_day_data alter column vid SET DEFAULT nextval('%%SCHEMA%%.pair_day_data_vid_seq'::regclass);
`

	ddl.createTables["token_day_data"] = `
create table if not exists %%SCHEMA%%.token_day_data
(
	id text not null,

	"date" numeric not null,

	"token" text not null,

	"daily_volume_token" numeric not null,

	"daily_volume_bnb" numeric not null,

	"daily_volume_usd" numeric not null,

	"daily_txns" numeric not null,

	"total_liquidity_token" numeric not null,

	"total_liquidity_bnb" numeric not null,

	"total_liquidity_usd" numeric not null,

	"price_usd" numeric not null,

	vid bigserial not null constraint token_day_data_pkey primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.token_day_data owner to graph;
alter sequence %%SCHEMA%%.token_day_data_vid_seq owned by %%SCHEMA%%.token_day_data.vid;
alter table only %%SCHEMA%%.token_day_data alter column vid SET DEFAULT nextval('%%SCHEMA%%.token_day_data_vid_seq'::regclass);
`

	ddl.indexes["pancake_factory"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_block_range_closed on %%SCHEMA%%.pancake_factory (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_id on %%SCHEMA%%.pancake_factory (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_updated_block_number on %%SCHEMA%%.pancake_factory (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_id_block_range_fake_excl on %%SCHEMA%%.pancake_factory using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_total_pairs on %%SCHEMA%%.pancake_factory using btree ("total_pairs");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_total_pairs;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_total_transactions on %%SCHEMA%%.pancake_factory using btree ("total_transactions");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_total_transactions;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_total_volume_usd on %%SCHEMA%%.pancake_factory using btree ("total_volume_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_total_volume_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_total_volume_bnb on %%SCHEMA%%.pancake_factory using btree ("total_volume_bnb");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_total_volume_bnb;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_untracked_volume_usd on %%SCHEMA%%.pancake_factory using btree ("untracked_volume_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_untracked_volume_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_total_liquidity_usd on %%SCHEMA%%.pancake_factory using btree ("total_liquidity_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_total_liquidity_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_factory_total_liquidity_bnb on %%SCHEMA%%.pancake_factory using btree ("total_liquidity_bnb");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_factory_total_liquidity_bnb;`,
		})

		return indexes
	}()

	ddl.indexes["bundle"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists bundle_block_range_closed on %%SCHEMA%%.bundle (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.bundle_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists bundle_id on %%SCHEMA%%.bundle (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.bundle_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists bundle_updated_block_number on %%SCHEMA%%.bundle (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.bundle_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists bundle_id_block_range_fake_excl on %%SCHEMA%%.bundle using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.bundle_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists bundle_bnb_price on %%SCHEMA%%.bundle using btree ("bnb_price");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.bundle_bnb_price;`,
		})

		return indexes
	}()

	ddl.indexes["token"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_block_range_closed on %%SCHEMA%%.token (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_id on %%SCHEMA%%.token (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_updated_block_number on %%SCHEMA%%.token (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_id_block_range_fake_excl on %%SCHEMA%%.token using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_name on %%SCHEMA%%.token ("left"("name", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_name;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_symbol on %%SCHEMA%%.token ("left"("symbol", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_symbol;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_decimals on %%SCHEMA%%.token using btree ("decimals");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_decimals;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_trade_volume on %%SCHEMA%%.token using btree ("trade_volume");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_trade_volume;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_untracked_volume_usd on %%SCHEMA%%.token using btree ("untracked_volume_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_untracked_volume_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_total_transactions on %%SCHEMA%%.token using btree ("total_transactions");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_total_transactions;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_total_liquidity on %%SCHEMA%%.token using btree ("total_liquidity");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_total_liquidity;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_derived_bnb on %%SCHEMA%%.token using btree ("derived_bnb");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_derived_bnb;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_derived_usd on %%SCHEMA%%.token using btree ("derived_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_derived_usd;`,
		})

		return indexes
	}()

	ddl.indexes["pair"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_block_range_closed on %%SCHEMA%%.pair (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_id on %%SCHEMA%%.pair (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_updated_block_number on %%SCHEMA%%.pair (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_id_block_range_fake_excl on %%SCHEMA%%.pair using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_name on %%SCHEMA%%.pair ("left"("name", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_name;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_token_0 on %%SCHEMA%%.pair using gist ("token_0", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_token_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_token_1 on %%SCHEMA%%.pair using gist ("token_1", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_token_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_reserve_0 on %%SCHEMA%%.pair using btree ("reserve_0");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_reserve_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_reserve_1 on %%SCHEMA%%.pair using btree ("reserve_1");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_reserve_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_total_supply on %%SCHEMA%%.pair using btree ("total_supply");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_total_supply;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_reserve_bnb on %%SCHEMA%%.pair using btree ("reserve_bnb");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_reserve_bnb;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_token_0_price on %%SCHEMA%%.pair using btree ("token_0_price");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_token_0_price;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_token_1_price on %%SCHEMA%%.pair using btree ("token_1_price");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_token_1_price;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_volume_token_0 on %%SCHEMA%%.pair using btree ("volume_token_0");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_volume_token_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_volume_token_1 on %%SCHEMA%%.pair using btree ("volume_token_1");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_volume_token_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_untracked_volume_usd on %%SCHEMA%%.pair using btree ("untracked_volume_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_untracked_volume_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_total_transactions on %%SCHEMA%%.pair using btree ("total_transactions");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_total_transactions;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_block on %%SCHEMA%%.pair using btree ("block");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_block;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_timestamp on %%SCHEMA%%.pair using btree ("timestamp");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_timestamp;`,
		})

		return indexes
	}()

	ddl.indexes["transaction"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists transaction_block_range_closed on %%SCHEMA%%.transaction (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.transaction_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists transaction_id on %%SCHEMA%%.transaction (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.transaction_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists transaction_updated_block_number on %%SCHEMA%%.transaction (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.transaction_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists transaction_id_block_range_fake_excl on %%SCHEMA%%.transaction using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.transaction_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists transaction_block on %%SCHEMA%%.transaction using btree ("block");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.transaction_block;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists transaction_timestamp on %%SCHEMA%%.transaction using btree ("timestamp");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.transaction_timestamp;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists transaction_mints on %%SCHEMA%%.transaction using gin (mints);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.transaction_mints;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists transaction_burns on %%SCHEMA%%.transaction using gin (burns);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.transaction_burns;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists transaction_swaps on %%SCHEMA%%.transaction using gin (swaps);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.transaction_swaps;`,
		})

		return indexes
	}()

	ddl.indexes["mint"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_block_range_closed on %%SCHEMA%%.mint (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_id on %%SCHEMA%%.mint (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_updated_block_number on %%SCHEMA%%.mint (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_id_block_range_fake_excl on %%SCHEMA%%.mint using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_transaction on %%SCHEMA%%.mint using gist ("transaction", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_transaction;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_timestamp on %%SCHEMA%%.mint using btree ("timestamp");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_timestamp;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_pair on %%SCHEMA%%.mint using gist ("pair", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_pair;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_token_0 on %%SCHEMA%%.mint using gist ("token_0", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_token_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_token_1 on %%SCHEMA%%.mint using gist ("token_1", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_token_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_to on %%SCHEMA%%.mint ("left"("to", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_to;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_liquidity on %%SCHEMA%%.mint using btree ("liquidity");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_liquidity;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_sender on %%SCHEMA%%.mint ("left"("sender", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_sender;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_amount_0 on %%SCHEMA%%.mint using btree ("amount_0");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_amount_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_amount_1 on %%SCHEMA%%.mint using btree ("amount_1");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_amount_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_log_index on %%SCHEMA%%.mint using btree ("log_index");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_log_index;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_amount_usd on %%SCHEMA%%.mint using btree ("amount_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_amount_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_fee_to on %%SCHEMA%%.mint ("left"("fee_to", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_fee_to;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists mint_fee_liquidity on %%SCHEMA%%.mint using btree ("fee_liquidity");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.mint_fee_liquidity;`,
		})

		return indexes
	}()

	ddl.indexes["burn"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_block_range_closed on %%SCHEMA%%.burn (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_id on %%SCHEMA%%.burn (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_updated_block_number on %%SCHEMA%%.burn (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_id_block_range_fake_excl on %%SCHEMA%%.burn using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_transaction on %%SCHEMA%%.burn using gist ("transaction", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_transaction;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_timestamp on %%SCHEMA%%.burn using btree ("timestamp");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_timestamp;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_pair on %%SCHEMA%%.burn using gist ("pair", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_pair;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_token_0 on %%SCHEMA%%.burn using gist ("token_0", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_token_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_token_1 on %%SCHEMA%%.burn using gist ("token_1", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_token_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_liquidity on %%SCHEMA%%.burn using btree ("liquidity");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_liquidity;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_sender on %%SCHEMA%%.burn ("left"("sender", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_sender;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_amount_0 on %%SCHEMA%%.burn using btree ("amount_0");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_amount_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_amount_1 on %%SCHEMA%%.burn using btree ("amount_1");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_amount_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_to on %%SCHEMA%%.burn ("left"("to", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_to;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_log_index on %%SCHEMA%%.burn using btree ("log_index");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_log_index;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_amount_usd on %%SCHEMA%%.burn using btree ("amount_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_amount_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_needs_complete on %%SCHEMA%%.burn using btree ("needs_complete");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_needs_complete;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_fee_to on %%SCHEMA%%.burn ("left"("fee_to", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_fee_to;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists burn_fee_liquidity on %%SCHEMA%%.burn using btree ("fee_liquidity");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.burn_fee_liquidity;`,
		})

		return indexes
	}()

	ddl.indexes["swap"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_block_range_closed on %%SCHEMA%%.swap (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_id on %%SCHEMA%%.swap (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_updated_block_number on %%SCHEMA%%.swap (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_id_block_range_fake_excl on %%SCHEMA%%.swap using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_transaction on %%SCHEMA%%.swap using gist ("transaction", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_transaction;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_timestamp on %%SCHEMA%%.swap using btree ("timestamp");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_timestamp;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_pair on %%SCHEMA%%.swap using gist ("pair", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_pair;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_token_0 on %%SCHEMA%%.swap using gist ("token_0", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_token_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_token_1 on %%SCHEMA%%.swap using gist ("token_1", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_token_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_sender on %%SCHEMA%%.swap ("left"("sender", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_sender;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_from on %%SCHEMA%%.swap ("left"("from", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_from;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_amount_0_in on %%SCHEMA%%.swap using btree ("amount_0_in");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_amount_0_in;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_amount_1_in on %%SCHEMA%%.swap using btree ("amount_1_in");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_amount_1_in;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_amount_0_out on %%SCHEMA%%.swap using btree ("amount_0_out");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_amount_0_out;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_amount_1_out on %%SCHEMA%%.swap using btree ("amount_1_out");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_amount_1_out;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_to on %%SCHEMA%%.swap ("left"("to", 256));`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_to;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_log_index on %%SCHEMA%%.swap using btree ("log_index");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_log_index;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists swap_amount_usd on %%SCHEMA%%.swap using btree ("amount_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.swap_amount_usd;`,
		})

		return indexes
	}()

	ddl.indexes["pancake_day_data"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_block_range_closed on %%SCHEMA%%.pancake_day_data (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_id on %%SCHEMA%%.pancake_day_data (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_updated_block_number on %%SCHEMA%%.pancake_day_data (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_id_block_range_fake_excl on %%SCHEMA%%.pancake_day_data using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_date on %%SCHEMA%%.pancake_day_data using btree ("date");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_date;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_daily_volume_bnb on %%SCHEMA%%.pancake_day_data using btree ("daily_volume_bnb");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_daily_volume_bnb;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_daily_volume_usd on %%SCHEMA%%.pancake_day_data using btree ("daily_volume_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_daily_volume_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_daily_volume_untracked on %%SCHEMA%%.pancake_day_data using btree ("daily_volume_untracked");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_daily_volume_untracked;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_total_volume_bnb on %%SCHEMA%%.pancake_day_data using btree ("total_volume_bnb");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_total_volume_bnb;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_total_liquidity_bnb on %%SCHEMA%%.pancake_day_data using btree ("total_liquidity_bnb");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_total_liquidity_bnb;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_total_volume_usd on %%SCHEMA%%.pancake_day_data using btree ("total_volume_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_total_volume_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_total_liquidity_usd on %%SCHEMA%%.pancake_day_data using btree ("total_liquidity_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_total_liquidity_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pancake_day_data_total_transactions on %%SCHEMA%%.pancake_day_data using btree ("total_transactions");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pancake_day_data_total_transactions;`,
		})

		return indexes
	}()

	ddl.indexes["pair_hour_data"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_block_range_closed on %%SCHEMA%%.pair_hour_data (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_id on %%SCHEMA%%.pair_hour_data (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_updated_block_number on %%SCHEMA%%.pair_hour_data (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_id_block_range_fake_excl on %%SCHEMA%%.pair_hour_data using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_hour_start_unix on %%SCHEMA%%.pair_hour_data using btree ("hour_start_unix");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_hour_start_unix;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_pair on %%SCHEMA%%.pair_hour_data using gist ("pair", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_pair;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_reserve_0 on %%SCHEMA%%.pair_hour_data using btree ("reserve_0");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_reserve_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_reserve_1 on %%SCHEMA%%.pair_hour_data using btree ("reserve_1");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_reserve_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_total_supply on %%SCHEMA%%.pair_hour_data using btree ("total_supply");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_total_supply;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_reserve_usd on %%SCHEMA%%.pair_hour_data using btree ("reserve_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_reserve_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_hourly_volume_token_0 on %%SCHEMA%%.pair_hour_data using btree ("hourly_volume_token_0");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_hourly_volume_token_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_hourly_volume_token_1 on %%SCHEMA%%.pair_hour_data using btree ("hourly_volume_token_1");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_hourly_volume_token_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_hourly_volume_usd on %%SCHEMA%%.pair_hour_data using btree ("hourly_volume_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_hourly_volume_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_hour_data_hourly_txns on %%SCHEMA%%.pair_hour_data using btree ("hourly_txns");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_hour_data_hourly_txns;`,
		})

		return indexes
	}()

	ddl.indexes["pair_day_data"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_block_range_closed on %%SCHEMA%%.pair_day_data (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_id on %%SCHEMA%%.pair_day_data (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_updated_block_number on %%SCHEMA%%.pair_day_data (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_id_block_range_fake_excl on %%SCHEMA%%.pair_day_data using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_date on %%SCHEMA%%.pair_day_data using btree ("date");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_date;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_pair_address on %%SCHEMA%%.pair_day_data using gist ("pair_address", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_pair_address;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_token_0 on %%SCHEMA%%.pair_day_data using gist ("token_0", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_token_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_token_1 on %%SCHEMA%%.pair_day_data using gist ("token_1", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_token_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_reserve_0 on %%SCHEMA%%.pair_day_data using btree ("reserve_0");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_reserve_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_reserve_1 on %%SCHEMA%%.pair_day_data using btree ("reserve_1");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_reserve_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_total_supply on %%SCHEMA%%.pair_day_data using btree ("total_supply");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_total_supply;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_reserve_usd on %%SCHEMA%%.pair_day_data using btree ("reserve_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_reserve_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_daily_volume_token_0 on %%SCHEMA%%.pair_day_data using btree ("daily_volume_token_0");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_daily_volume_token_0;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_daily_volume_token_1 on %%SCHEMA%%.pair_day_data using btree ("daily_volume_token_1");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_daily_volume_token_1;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_daily_volume_usd on %%SCHEMA%%.pair_day_data using btree ("daily_volume_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_daily_volume_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists pair_day_data_daily_txns on %%SCHEMA%%.pair_day_data using btree ("daily_txns");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.pair_day_data_daily_txns;`,
		})

		return indexes
	}()

	ddl.indexes["token_day_data"] = func() []*index {
		var indexes []*index
		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_block_range_closed on %%SCHEMA%%.token_day_data (COALESCE(upper(block_range), 2147483647)) where (COALESCE(upper(block_range), 2147483647) < 2147483647);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_block_range_closed;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_id on %%SCHEMA%%.token_day_data (id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_id;`,
		})
		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_updated_block_number on %%SCHEMA%%.token_day_data (_updated_block_number);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_updated_block_number;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_id_block_range_fake_excl on %%SCHEMA%%.token_day_data using gist (block_range, id);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_id_block_range_fake_excl;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_date on %%SCHEMA%%.token_day_data using btree ("date");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_date;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_token on %%SCHEMA%%.token_day_data using gist ("token", block_range);`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_token;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_daily_volume_token on %%SCHEMA%%.token_day_data using btree ("daily_volume_token");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_daily_volume_token;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_daily_volume_bnb on %%SCHEMA%%.token_day_data using btree ("daily_volume_bnb");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_daily_volume_bnb;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_daily_volume_usd on %%SCHEMA%%.token_day_data using btree ("daily_volume_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_daily_volume_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_daily_txns on %%SCHEMA%%.token_day_data using btree ("daily_txns");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_daily_txns;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_total_liquidity_token on %%SCHEMA%%.token_day_data using btree ("total_liquidity_token");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_total_liquidity_token;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_total_liquidity_bnb on %%SCHEMA%%.token_day_data using btree ("total_liquidity_bnb");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_total_liquidity_bnb;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_total_liquidity_usd on %%SCHEMA%%.token_day_data using btree ("total_liquidity_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_total_liquidity_usd;`,
		})

		indexes = append(indexes, &index{
			createStatement: `create index if not exists token_day_data_price_usd on %%SCHEMA%%.token_day_data using btree ("price_usd");`,
			dropStatement:   `drop index if exists %%SCHEMA%%.token_day_data_price_usd;`,
		})

		return indexes
	}()
	ddl.schemaSetup = `
CREATE SCHEMA if not exists %%SCHEMA%%;
DO
$do$
    BEGIN
        IF NOT EXISTS (
                SELECT FROM pg_catalog.pg_roles  -- SELECT list can be empty for this
                WHERE  rolname = 'graph') THEN
            CREATE ROLE graph;
        END IF;
    END
$do$;

set statement_timeout = 0;
set idle_in_transaction_session_timeout = 0;
set client_encoding = 'UTF8';
set standard_conforming_strings = on;
select pg_catalog.set_config('search_path', '', false);
set check_function_bodies = false;
set xmloption = content;
set client_min_messages = warning;
set row_security = off;

create extension if not exists btree_gist with schema %%SCHEMA%%;


create table if not exists %%SCHEMA%%.cursor
(
	id integer not null
		constraint cursor_pkey
			primary key,
	cursor text
);
alter table %%SCHEMA%%.cursor owner to graph;

create table %%SCHEMA%%.poi2$
(
    digest      bytea     not null,
    id          text      not null,
    vid         bigserial not null
        constraint poi2$_pkey
            primary key,
    block_range int4range not null,
	_updated_block_number  numeric not null,
    constraint poi2$_id_block_range_excl
        exclude using gist (id with =, block_range with &&)
);

alter table %%SCHEMA%%.poi2$
    owner to graph;

create index brin_poi2$
    on %%SCHEMA%%.poi2$ using brin (lower(block_range), COALESCE(upper(block_range), 2147483647), vid);

CREATE INDEX poi2$_updated_block_number
    ON %%SCHEMA%%.poi2$ USING btree
	(_updated_block_number ASC NULLS LAST)
	TABLESPACE pg_default;

create index poi2$_block_range_closed
    on %%SCHEMA%%.poi2$ (COALESCE(upper(block_range), 2147483647))
    where (COALESCE(upper(block_range), 2147483647) < 2147483647);

create index attr_12_0_poi2$_digest
    on %%SCHEMA%%.poi2$ (digest);

create index attr_12_1_poi2$_id
    on %%SCHEMA%%.poi2$ ("left"(id, 256));

create table if not exists %%SCHEMA%%.dynamic_data_source_xxx
(
	id text not null,
	context text not null,
	abi text not null,
	vid bigserial not null
		constraint dynamic_data_source_xxx_pkey
			primary key,
	block_range int4range not null,
	_updated_block_number numeric not null
);

alter table %%SCHEMA%%.dynamic_data_source_xxx owner to graph;

create index if not exists dynamic_data_source_xxx_block_range_closed
	on %%SCHEMA%%.dynamic_data_source_xxx (COALESCE(upper(block_range), 2147483647))
	where (COALESCE(upper(block_range), 2147483647) < 2147483647);

create index if not exists dynamic_data_source_xxx_id
	on %%SCHEMA%%.dynamic_data_source_xxx (id);

create index if not exists dynamic_data_source_xxx_abi
	on %%SCHEMA%%.dynamic_data_source_xxx (abi);

`

}

func (d *DDL) InitiateSchema(handleStatement func(statement string) error) error {
	err := handleStatement(d.schemaSetup)
	if err != nil {
		return fmt.Errorf("handle statement: %w", err)
	}
	return nil
}

func (d *DDL) CreateTables(handleStatement func(table string, statement string) error) error {
	for table, statement := range d.createTables {
		err := handleStatement(table, statement)
		if err != nil {
			return fmt.Errorf("handle statement: %w", err)
		}
	}
	return nil
}

func (d *DDL) CreateIndexes(handleStatement func(table string, statement string) error) error {
	for table, idxs := range d.indexes {
		for _, idx := range idxs {
			err := handleStatement(table, idx.createStatement)
			if err != nil {
				return fmt.Errorf("handle statement: %w", err)
			}
		}
	}
	return nil
}

func (d *DDL) DropIndexes(handleStatement func(table string, statement string) error) error {
	for table, idxs := range d.indexes {
		for _, idx := range idxs {
			err := handleStatement(table, idx.dropStatement)
			if err != nil {
				return fmt.Errorf("handle statement: %w", err)
			}
		}
	}
	return nil
}

type TypedEntity struct {
	Type   string
	Entity graphnode.Entity
}

func (t *TypedEntity) UnmarshalJSON(data []byte) error {
	s := &struct {
		Type   string          `json:"type" yaml:"type"`
		Entity json.RawMessage `json:"entity" yaml:"entity"`
	}{}

	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	var ent graphnode.Entity
	switch s.Type {
	case "pancake_factory":
		tempEnt := &PancakeFactory{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "bundle":
		tempEnt := &Bundle{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "token":
		tempEnt := &Token{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "pair":
		tempEnt := &Pair{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "transaction":
		tempEnt := &Transaction{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "mint":
		tempEnt := &Mint{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "burn":
		tempEnt := &Burn{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "swap":
		tempEnt := &Swap{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "pancake_day_data":
		tempEnt := &PancakeDayData{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "pair_hour_data":
		tempEnt := &PairHourData{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "pair_day_data":
		tempEnt := &PairDayData{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	case "token_day_data":
		tempEnt := &TokenDayData{}
		err := json.Unmarshal(s.Entity, &tempEnt)
		if err != nil {
			return err
		}
		ent = tempEnt
	}

	t.Entity = ent
	t.Type = s.Type

	return nil
}
