Solana SPL Token Transfers Substreams
=====================================

Install [`substreams` and its dependencies here](../README.md), compile these modules with:

## Building
```
cargo build --release
```

## Protobuf Generation
```
substreams protogen substreams.yaml --exclude-paths="sf/solana,sf/substreams,google"
```

## Running the Substrams
```
substreams run -k -e mainnet.sol.streamingfast.io:443 substreams.yaml store_mint_native_volumes -s 30010000 -t 30010010
```

This will make the substreams backprocess 30,000,000 to 30,010,000 in parallel, then start streaming the next 10 blocks.
