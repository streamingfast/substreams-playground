package manifest


type Manifest struct {
	Streams []Stream
}

type Stream struct {
}

func (s *Stream) SortedNodesAbove(name string) []Stream {

}
