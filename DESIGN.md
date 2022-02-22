
Example `substreams.yaml` declaration:

---
---
































---------



How could PancakeSwap be mapped to a Transforms based, event-driven stream, with caching between layers

## Transforms

The Transforms can be specified in the `transforms` field of a `firehose.Request`


### SuccessfulTransactions  (native Firehose transform)

Context-Free (parallel preproc)

* Purges unsuccessful transactions from a Block.
* Declares it doesn't want its output cached, because this transform doesn't provide enough value to warrant a full recache.


### AccountsFilter (native)

Context-Free (parallel preproc)

* Specifies the Factory contract address, and event signatures (PairCreated()), and filters out the incoming Block
* Outputs: lightweight blocks, containing only the transactions that include those events


### EventsFilter  (native)

Context-Free

CONFIG: JSON abi, to do decoding
INPUT: Block
OUTPUT: `repeated ethereum.DecodedLogEvent`, sorted by `trx_idx, call_idx`

Used to decode the `PairCreated()` as a parent of `PairExtractor`
Used to decode the `Transfer()` events, parent of `TransferSummer`



### PairExtractor

Context-Free (parallel preproc)

Calls Eth to get the token decimals, etc..

CONFIG: JSON-ABI matching the event we're after
INPUT: `repeated ethereum.DecodedLogEvent` from a previous EventsFilter{`PairCreated()`} with the corresponding JSON ABI, outputs those matching PairCreated()
OUTPUT:
  1. pancakeswap.v1.PairsCreated
  2. database.EntitiesUpdates{
       database.Entity{"type": "PancakeFactory", "id": "0", "$add": {"total_pairs": 1}}
//    UPDATE pacnafatory WHERE Id = 0 SET total_pairs = total_pairs + 1;
//      on undo, UNDO
     }

Goes through history, filters Events and Accounts (the PancakeSwap Factory)
and catches the Pairs
Extracts the token, via a eth_call if necessary.
* PairCache, gets the outputted Pairs from PairExtractor, declared to use PairExtractor output.
  * Declares it needs linear consummation of previous Transform
* PairExtractor reboot, load from previous pairs?


firehose.Request{
  transforms: [PairExtractor, EmptyOutputSkipper]
  transforms: [PairExtractor]
}


### PairCache

Context-Aware (bstream.Handler?)
DEFINITION SHOULD CONTAIN A START BLOCK FOR THIS EXTRACTION TO BE COMPLETE
* Would request that its SOURCE had been processed starting at least at block ^^
CACHES_DATA: true
INPUT: `pancakeswap.v1.PairsCreated`     -> `repeated pancakeswap.v1.PairCreated`
OUTPUT:
* `pancakeswap.v1.PairsCreated`  -> `repeated pancakeswap.v1.PairCreated`
* TRANSITIVE: `pancakeswap.v1.PairCreatedMap`  -> `map<pancakeswap.v1.PairCreated`
INIT: loads all previous 100 blocks' worth of data from its SOURCE, for all concerned history.

FLUSHABLE ONLY EACH 100 BLOCKS. Can be resurrected from the flushed content, to avoid reprocessing history.
* Snapshotting functionality? Concept of state is accumulated here?!
  * YES YES YES
* Operator driven cache, or some algorithmic size vs frequency vs CPU time tradeoff?

* Add it to its local storage. Storage that is checkpointed at the current block.
  * Priming its cache requires linear processing
* This one needs to be aware of forks, and fix its cache before handing down some references
  * Could pass the PairCache as a ref to a `map`, guaranteed to be non-threaded downstream

* Is LinearBuilder, so needs to have processed all of the previous history before it can be used
  at the current block.


### ReservesExtractor

CONTEXT-FREE
INPUT: Block
OUTPUT: `pancakeswap.v1.Reserves`      -> `repeated pancakeswap.v1.Reserve`

The `pancakeswap.v1.Reserve` object needs to include an absolute positioning within the Block, so we know how to stitch with the other transactions.

This would just extract the reserves from a given signature, but do that in a context-free way. It wouldn't know if it's part of PancakeSwap yet.

Ideally context-free, but now we need the PairCache, so it's kind of forced to be linear?

* Could we have an OPTIONAL context-free handler that is passed the PairCache when in parallel, and a context-aware handler when we're more live?!


### PairFilteredReserveExtractor


CONTEXT-AWARE:
INPUT: PairCreatedMap + PairCreated events
INPUT: du ReservesExtractor
OUTPUT: `repeated pancakeswap.v1.Reserve`

We know this was filtered according to the PancakeSwap pairs only.


### ETHPriceTracker

CONTEXT-FREE:
INPUT: `repeated pancakeswap.v1.Reserve`
OUTPUT: `pancakeswap.v1.ETHPrice`

Only for a FEW tokens will we output an ETH Price, because we want to ignore whatever TOKENA/TOKENB prices.

This module will filter certain reserve updates, and issue a new price for ETH in USD when some match

### ETHPriceCache

CONTEXT-AWARE
INPUT: `pancakeswap.v1.ETHPrice`
OUTPUT: `pancakeswap.v1.WeightedETHPrice`, one per block, with or without input?
QUERYABLE: `GetWeightedETHPrice(position)` ?


### PancakeToDatabase

INPUT: `pancakeswap.v1.Reserve`, `pancakeswap.v1.PairCreated`, `pancakeswap.v1.ETHPrice`
OUTPUT: `database.RowUpdates`

Transforms all the data bits into writable rows. Can collate and merge changes at the block level, do to ONE operation instead of 25.
  Example: created 25 pairs, just run an UPDATE on the `total_pairs` ONCE, with +25.


### DatabaseEntityFilter

Filters the input rows, keeping only certain ones, tweaking some others, etc..

INPUT: `database.RowUpdates`
OUTPUT: `database.RowUpdates`
SIDE-EFFECT: none

Allows for stripping some columns, stripping some tables

### DatabaseEntityWriter

CONTEXT-AWARE SINK
INPUT: `database.RowUpdates`
OUTPUT: nil
SIDE-EFFECT: write to Postgres


### ConsensusProgress

```proto
message ConsensusProgressConfig {
  bool send_on_new = 1;
  bool send_on_undo = 2;
  bool send_on_irreversible = 3;
  bool send_on_stalled = 4; // or orphaned, meaning we're sure it's not part of the chain anymore, determinism not guaranteed here
  string irreversibility_condition = 1;
}
```

INPUT: anything, accompanied by a Block, and with a ForkDB.Object, indicating the Step
OUTPUT: whatever was received, or a `transforms.Skip`



### IrreversibleMarker

CONTEXT-AWARE, feeds from the ForkDB's output.

INPUT: `bstream.types.v1.Irreversible`
OUTPUT: `bstream.types.v1.Irreversible`

```
```

Offers APIs to be queried by the Forkable, to feed the irreversible blocks.

```
IsBlockIrreversible(blockID string) bool
```


### EmptyOutputSkipper (native transform)

Configured with the hash of the whole Transforms chain

SIDE-EFFECT: each 100 blocks irreversible, flush a roaring bitmap to side storage.
  * Forkable::MarkBlockIrreversible()
    *
  * API pour querier le Hash du Transform Pipeline courant.
  * Ou de quoi pour écrire un fichier associé aux 100 blocks courants
    * Cet API là devrait pas fonctionner si on a envoyé autre chose que des blocks irreversible dans la chunk de 100 blocks.
INIT: load a series of blocks?
INPUT:
* readOnlyBlock
* `any` from previous OUTPUT
*
OUTPUT: `bstream.


RoaringBitmap, une par 100 blocks
* ABSOLUTELY REQUIRES input blocks to be IRREVERSIBLE

Offers API, used by `bstream` on INITIALIZATION (boot of the transform chain)
* Fetches all the 100 blocks concerned by the block RANGE


## Alternative flows

Each Transform would have a SOURCE
* and that source can be a previous transform by name, or by hash, so we can either instantiate the live source, or fetch the corresponding files
* a source of Blocks, so FileSource + LiveSource, etc.


## Proto message types:


```proto
message ethereum.DecodedLogEvent {
  uint64 trx_idx;
  uint64 call_idx;
  string transaction_id;
  string from_addr;
  string event_name;
  string params;
}

message pancakeswap.v1.ETHPrice {
  uint64 trx_idx;
  uint64 call_idx;
  sf.types.BigInt volume;
  sf.types.BigFloat price;
  string pair_addr;
}

message pancakeswap.v1.Reserve {
  uint64 trx_idx;
  uint64 call_idx;
  string pair_id;
  BigInt reserve0;
  BigInt reserve1;
}

message pancakeswap.v1.PairsCreated {
  repeated pairs pancakeswap.v1.PairCreated;
}

message pancakeswap.v1.PairCreated {
  string pair_id;
  Token token0;
  Token token1;

  message Token {
	string addr;
	uint64 decials;
	string symbol;
	string name;
  }
}

message database.RowUpdates {
  repeated Field;

  message Field {
    string column;
	string value;  // CSV representation for the backing db
  }
}
```


## Organization

We need some chains of CONTEXT-FREE and CONTEXT-AWARE processes
Each time we hit a context-aware module, we need to have it process linearly, and then in lockstep with the rest.
* Hopefully, the linear process it sources can be done extremely quickly, because its source will be context-free.
* Since everything probably starts with some filtering that can be context-free, the full chain can be reduced a lot

This sort of embodies the different stages that we have in Pancake.


## Benefits

We can more easily reduce the auto-generated tables and indexes, and have some subgraphs build on the work done by other subgraphs, and decide they want to write only this or that column, this or that collection, etc..


-------------


OnesSource

TwosSource

FoursSource

CombiningSource
INPUT: `OnesSource`, `TwosSource`, `FoursSource`
OUTPUT: `MathOperation`

Computes: if a math operation between numbers in a block, you output Skip, otherwise, output the math operation.

If you "SkipEmptyMessage" at the end of it, you cannot after the fact, prevent loading segments of OnesSource



---------

Would we model CONTEXT-FREE + a SINGLE Context-Aware stream, so the RoaringBitmap optimization can apply to all.


pairs, transactions, balances, swaps



------

## State DB model


### WritableRowExtractor

CONTEXT-FREE:
OUTPUT: repeated WritableRows

-----

### StateDBRowSharder

CONTEXT-FREE:
Params: {
  index: 1,
  shards: 10,
}
OUTPUT: repeated WritableRows

### StateDBWriter

CONTEXT-AWARE:
