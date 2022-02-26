package wasm

import (
	"fmt"

	"github.com/wasmerio/wasmer-go/wasmer"
)

func params(kinds ...wasmer.ValueKind) []*wasmer.ValueType {
	return wasmer.NewValueTypes(kinds...)
}

func returns(kinds ...wasmer.ValueKind) []*wasmer.ValueType {
	return wasmer.NewValueTypes(kinds...)
}

type abortError struct {
	message      string
	filename     string
	lineNumber   int
	columnNumber int
}

func (e *abortError) Error() string {
	return fmt.Sprintf("wasm execution aborted at %s:%d env:%d env: %s", e.filename, e.lineNumber, e.columnNumber, e.message)
}
