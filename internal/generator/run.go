package generator

import (
	"fmt"
)

func Run(_ string) error {
	// Stub: in v0 this will parse schema/*.schema.go via AST and emit code.
	// We keep a placeholder here to keep CLI functional.
	fmt.Println("generator: (stub) parsing schema and writing generated code...")
	return nil
}
