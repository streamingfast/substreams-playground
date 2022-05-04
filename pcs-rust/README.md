PancakeSwap Substreams
======================

Install [`substreams` and its dependencies here](../README.md), compile these modules with:

```
cd ../eth-token/build.sh
cd ../pcs-rust
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
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> block_to_pairs
  block_to_pairs[map: block_to_pairs] --> pairs
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> block_to_reserves
  pairs[store: pairs] --> block_to_reserves
  build_pcs_token_state[store: build_pcs_token_state] --> block_to_reserves
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> reserves
  block_to_reserves[map: block_to_reserves] --> reserves
  pairs[store: pairs] --> reserves
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> prices
  block_to_reserves[map: block_to_reserves] --> prices
  pairs[store: pairs] --> prices
  reserves[store: reserves] --> prices
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> mint_burn_swaps_extractor
  pairs[store: pairs] --> mint_burn_swaps_extractor
  prices[store: prices] --> mint_burn_swaps_extractor
  build_pcs_token_state[store: build_pcs_token_state] --> mint_burn_swaps_extractor
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> totals
  block_to_pairs[map: block_to_pairs] --> totals
  mint_burn_swaps_extractor[map: mint_burn_swaps_extractor] --> totals
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> volumes
  mint_burn_swaps_extractor[map: mint_burn_swaps_extractor] --> volumes
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> block_to_tokens
  block_to_tokens[map: block_to_tokens] --> tokens
  block_to_pairs[map: block_to_pairs] --> build_pcs_token_state
  tokens[store: tokens] --> build_pcs_token_state
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> db_out
  build_pcs_token_state[store: build_pcs_token_state] -- "deltas" --> db_out
  pairs[store: pairs] -- "deltas" --> db_out
  totals[store: totals] -- "deltas" --> db_out
  volumes[store: volumes] -- "deltas" --> db_out
  reserves[store: reserves] -- "deltas" --> db_out
  mint_burn_swaps_extractor[map: mint_burn_swaps_extractor] --> db_out
  build_pcs_token_state[store: build_pcs_token_state] --> db_out
  ```
