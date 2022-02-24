package pipeline

import (
	"strings"

	"github.com/streamingfast/substream-pancakeswap/subscription"
	"go.uber.org/zap"
)

// unsure where this will go

func (p *Pipeline) setupSubscriptionHub() {
	// TODO: wwwooah, SubscriptionHub has a meaning in the context of bstream,
	// this would be *another* flavor SubscriptionHub? We're talking of a generic Pub/Sub here?
	//
	// Let's discuss the purpose of this thing.
	p.subscriptionHub = subscription.NewHub()

	for storeName := range p.stores {
		if err := p.subscriptionHub.RegisterTopic(storeName); err != nil {
			zlog.Fatal("pair subscriber register topic", zap.Error(err))
		}
	}

}

func (p *Pipeline) setupPrintPairUpdates() {
	pairSubscriber := subscription.NewSubscriber()
	if err := p.subscriptionHub.Subscribe(pairSubscriber, "pairs"); err != nil {
		zlog.Fatal("subscription hub subscribe", zap.Error(err))
	}

	go func() {
		for {
			delta, err := pairSubscriber.Next()
			if err != nil {
				zlog.Fatal("pair subscriber next", zap.Error(err))
			}
			if !strings.HasPrefix(delta.Key, "pair") {
				continue
			}

			p.stores["pairs"].PrintDelta(delta)

		}
	}()
	// End subscription hub
}
