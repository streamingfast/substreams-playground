PancakeSwap Substreams
======================

Install [`substreams` and its dependencies here](https://github.com/streamingfast/substreams), compile these modules with:

```
cd ../eth-token/build.sh
cd ../pancakeswap
./build.sh
```

and try with:

```
substreams run -e bsc-dev.streamingfast.io:443 substreams.yaml pairs,block_to_pairs,db_out,volumes,totals -s 6810706 -t 6810711
```

## Visual data flow

This is a flow that is executed for each block.  The graph is produced automatically from the `.yaml` manifest.

```mermaid

graph TD;
  block_to_pairs[map: block_to_pairs]
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> block_to_pairs
  pairs[store: pairs]
  block_to_pairs --> pairs
  pcs_tokens[store: pcs_tokens]
  block_to_pairs --> pcs_tokens
  ethtokens:tokens --> pcs_tokens
  block_to_reserves[map: block_to_reserves]
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> block_to_reserves
  pairs --> block_to_reserves
  pcs_tokens --> block_to_reserves
  reserves[store: reserves]
  sf.substreams.v1.Clock[source: sf.substreams.v1.Clock] --> reserves
  block_to_reserves --> reserves
  pairs --> reserves
  prices[store: prices]
  sf.substreams.v1.Clock[source: sf.substreams.v1.Clock] --> prices
  block_to_reserves --> prices
  pairs --> prices
  reserves --> prices
  mint_burn_swaps_extractor[map: mint_burn_swaps_extractor]
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> mint_burn_swaps_extractor
  pairs --> mint_burn_swaps_extractor
  prices --> mint_burn_swaps_extractor
  pcs_tokens --> mint_burn_swaps_extractor
  totals[store: totals]
  sf.substreams.v1.Clock[source: sf.substreams.v1.Clock] --> totals
  block_to_pairs --> totals
  mint_burn_swaps_extractor --> totals
  volumes[store: volumes]
  sf.substreams.v1.Clock[source: sf.substreams.v1.Clock] --> volumes
  mint_burn_swaps_extractor --> volumes
  db_out[map: db_out]
  sf.substreams.v1.Clock[source: sf.substreams.v1.Clock] --> db_out
  pcs_tokens -- deltas --> db_out
  pairs -- deltas --> db_out
  totals -- deltas --> db_out
  volumes -- deltas --> db_out
  reserves -- deltas --> db_out
  mint_burn_swaps_extractor --> db_out
  pcs_tokens --> db_out
  ethtokens:block_to_tokens[map: ethtokens:block_to_tokens]
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> ethtokens:block_to_tokens
  ethtokens:tokens[store: ethtokens:tokens]
  ethtokens:block_to_tokens --> ethtokens:tokens
  ```
