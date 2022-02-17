package subscription

import (
	"github.com/streamingfast/sparkle-pancakeswap/state"
)

type Subscriber struct {
	input             chan state.StateDelta
	passedGracePeriod bool // allows blocks to go in channel even if len > h.sourceChannelSize
	Shutdown          func(error)
}

func NewSubscriber() *Subscriber {
	return &Subscriber{
		input:             make(chan state.StateDelta, 100),
		passedGracePeriod: false,
		Shutdown:          nil,
	}
}

func (s *Subscriber) Next() (*state.StateDelta, error) {
	select {
	case next := <-s.input:
		return &next, nil
	}

	//return nil, io.EOF
}
