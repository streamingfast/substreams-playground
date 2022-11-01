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
substreams run -e mainnet.sol.streamingfast.io:443 substreams.yaml store_mint_native_volumes -s 31330000 -t +50 -i
```

This will request the server to
1) backprocess 31,313,760 (substreams initialBlock) to 31,330,000 (requested start point) in parallel
2) because of the [-i] flag, it will send you the full state of the store at that particular block
3) then it starts streaming the next 50 blocks.
