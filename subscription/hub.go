package subscription

import (
	"fmt"
	"sync"

	"github.com/streamingfast/sparkle-pancakeswap/state"
)

type topicSubscriptions map[string][]*Subscriber

type Hub struct {
	topicSubscriptions topicSubscriptions
	subscribersMutex   sync.Mutex // Locks `buffer` reads and writes
}

func NewHub() *Hub {
	return &Hub{
		topicSubscriptions: topicSubscriptions{},
	}
}

func (h *Hub) RegisterTopic(topic string) error {
	if _, found := h.topicSubscriptions[topic]; found {
		return fmt.Errorf("topic [%s] already registered", topic)
	}

	h.topicSubscriptions[topic] = []*Subscriber{}
	return nil
}

func (h *Hub) BroadcastDeltas(topic string, deltas []state.StateDelta) error {
	if len(deltas) == 0 {
		return nil
	}
	h.subscribersMutex.Lock()
	defer h.subscribersMutex.Unlock()
	if subscriptions, found := h.topicSubscriptions[topic]; found {
		for _, delta := range deltas {
			for _, subscription := range subscriptions {
				subscription.input <- delta
			}
		}
		return nil
	}

	return fmt.Errorf("topic [%s] not found", topic)
}

func (h *Hub) BroadcastDelta(topic string, delta state.StateDelta) error {
	h.subscribersMutex.Lock()
	defer h.subscribersMutex.Unlock()
	if subscriptions, found := h.topicSubscriptions[topic]; found {
		for _, subscription := range subscriptions {
			subscription.input <- delta
		}
		return nil
	}

	return fmt.Errorf("topic [%s] not found", topic)
}

func (h *Hub) Subscribe(subscriber *Subscriber, topic string) error {
	h.subscribersMutex.Lock()
	defer h.subscribersMutex.Unlock()

	if subscriptions, found := h.topicSubscriptions[topic]; found {
		h.topicSubscriptions[topic] = append(subscriptions, subscriber)
		return nil
	}

	return fmt.Errorf("topic [%s] not found", topic)
}

func (h *Hub) Unsubscribe(removeSub *Subscriber) {
	//h.subscribersMutex.Lock()
	//defer h.subscribersMutex.Unlock()
	//
	//var newSubscriber []*subscriber
	//for _, sub := range h.subscribers {
	//	if sub != removeSub {
	//		newSubscriber = append(newSubscriber, sub)
	//	}
	//}
	//h.subscribers = newSubscriber
}
