package exchange

type PCSPairsPriceStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *PCSPairsPriceStateBuilder) BuildState(reserveUpdates PCSReserveUpdates, builder *StateBuilder) error {

	for _, update := range reserveUpdates {

	}
	return nil
}
