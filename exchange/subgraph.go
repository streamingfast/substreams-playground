package exchange

import (
	"encoding/json"

	"github.com/streamingfast/eth-go"
	"go.uber.org/zap"
)

var tokensToPair map[string]string

type PairContext struct {
	Token0 eth.Address `json:"token_0"`
	Token1 eth.Address `json:"token_1"`
}

func (s *Subgraph) Init() error {
	tokensToPair = make(map[string]string, len(s.DynamicDataSources))

	for _, dds := range s.DynamicDataSources {
		if dds.ABI != "Pair" {
			continue
		}
		var ctx *PairContext
		if err := json.Unmarshal([]byte(dds.Context), &ctx); err != nil {
			return err
		}

		tokensToPair[generateTokensKey(ctx.Token0.Pretty(), ctx.Token1.Pretty())] = dds.GetID()
	}

	return nil
}

func (s *Subgraph) CreatePairTemplateWithTokens(addr eth.Address, token0, token1 eth.Address) error {
	tokensToPair[generateTokensKey(token0.Pretty(), token1.Pretty())] = addr.Pretty()

	ctx := &PairContext{
		Token0: token0,
		Token1: token1,
	}
	return s.CreatePairTemplate(addr, ctx)
}

func (s *Subgraph) LogStatus() {
	s.Log.Debug("loaded tracked address", zap.Int("count", len(s.DynamicDataSources)))
	s.Log.Debug("loaded tracked token pairs", zap.Int("count", len(tokensToPair)))
}

func (s *Subgraph) getPairAddressForTokens(token0, token1 string) string {
	return tokensToPair[generateTokensKey(token0, token1)]
}
