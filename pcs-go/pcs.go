package pcs

func (p *Pair) GetOrdinal() uint64 {
	return p.LogOrdinal
}

func (p *Event) GetOrdinal() uint64 {
	return p.LogOrdinal
}
