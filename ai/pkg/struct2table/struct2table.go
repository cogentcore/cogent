package struct2table

import (
	"reflect"

	"cogentcore.org/cogent/ai/pkg/tree"
)

type (
	Type interface {
		reflect.Kind
	}
	Object struct {
		Field
		children []Object
		*tree.Node[*Object]
	}
	Field struct {
		Key   reflect.Type
		Value reflect.Value
	}
)

func NewObject() tree.Provider[*Object] { return &Object{} }
