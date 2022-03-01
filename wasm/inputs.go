package wasm

import "github.com/streamingfast/substream-pancakeswap/state"

type InputType int

const (
	InputStream InputType = iota
	InputStore
	OutputStore
)

type Input struct {
	Type InputType
	Name string

	// Transient data between calls
	StreamData []byte

	// InputType == InputStore || OutputStore
	Store   *state.Builder

	// If InputType == OutputStore
	MergeStrategy string // MAX_INT, MIN_INT, LAST_KEY, FIRST_KEY, SUM_INT, SUM_FLOAT, DISABLE_PARALLELISM
}
