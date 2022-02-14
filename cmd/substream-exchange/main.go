package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/firehose"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/sparkle-pancakeswap/exchange"
	"github.com/streamingfast/sparkle/indexer"
	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
)

func main() {
	// Start piping blocks from a Firehose instance

	// these, taken from `index.go` in `sparkle/cli` stuff

	rpcEndpoint := "http://localhost:8546" //  kc port-forward sub-pancake4-exchange-lucid-koschei-59686b7cc6-k49jk 8546:10.0.1.19:8546

	//blocksStore, err := dstore.NewDBinStore("gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1")
	blocksStore, err := dstore.NewDBinStore("./localblocks")
	if err != nil {
		log.Fatalln("error setting up blocks store:", err)
	}

	pipe := setupPipeline(rpcEndpoint)

	const startBlock = 6810700 // 6809737
	hose := firehose.New([]dstore.Store{blocksStore}, startBlock, pipe,
		firehose.WithForkableSteps(bstream.StepIrreversible),
	)

	if err := hose.Run(context.Background()); err != nil {
		log.Fatalln("running the firehose:", err)
	}
}

func setupPipeline(rpcEndpoint string) bstream.Handler {

	httpClient := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true, // don't reuse connections
		},
		Timeout: 3 * time.Second,
	}
	rpcClient := rpc.NewClient(rpcEndpoint, rpc.WithHttpClient(httpClient))
	subgraphDef := exchange.Definition

	rpcCacheStore, err := dstore.NewStore("./rpc-cache", "", "", false)
	if err != nil {
		log.Fatalln("setting up store for rpc-cache:", err)
	}

	rpcCache := indexer.NewCache(rpcCacheStore, rpcCacheStore, 0, 999)

	intr := exchange.NewSubstreamIntrinsics(rpcClient, rpcCache, true)
	_ = subgraphDef

	pairsStore := exchange.NewStateBuilder("pairs")
	pairExtractor := &exchange.PairExtractor{SubstreamIntrinsics: intr, Contract: eth.Address(exchange.FactoryAddressBytes)}
	pairsStoreProc := &exchange.PCSPairsStore{SubstreamIntrinsics: intr, PairsStore: pairsStore}
	reservesExtractor := &exchange.ReservesExtractor{SubstreamIntrinsics: intr}

	return bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {

		// TODO: eventually, handle the `undo` signals.

		blk := block.ToNative().(*pbcodec.Block)
		intr.SetCurrentBlock(blk)

		fmt.Println("block", blk.Num(), blk.ID())

		pairs, err := pairExtractor.Map(blk)
		if err != nil {
			return fmt.Errorf("extracting pairs: %w", err)
		}

		if len(pairs) != 0 {
			fmt.Println("pairs updates:")
			cnt, _ := json.MarshalIndent(pairs, "", "  ")
			fmt.Println(string(cnt))
		}
		// TODO: flush `pairs` output to disk somewhere

		if err := pairsStoreProc.Process(blk, pairs); err != nil {
			return fmt.Errorf("processing pair cache: %w", err)
		}

		if len(pairsStore.Deltas) != 0 {
			fmt.Println("state deltas:")
			cnt, _ := json.MarshalIndent(pairsStore.Deltas, "", "  ")
			fmt.Println(string(cnt))
			// TODO: flush the StateDeltas produced in the "Process" step above, apply for downstream
		}

		pairsStore.Flush()

		updates, err := reservesExtractor.Map(blk, pairsStore)
		if err != nil {
			return fmt.Errorf("processing reserves extractor: %w", err)
		}

		if len(updates) != 0 {
			fmt.Println("reserves updates:")
			cnt, _ := json.MarshalIndent(updates, "", "  ")
			fmt.Println(string(cnt))
		}

		// TODO: flush those `updates` somewhere, as the output of the reserves extractor

		return nil
	})
}
