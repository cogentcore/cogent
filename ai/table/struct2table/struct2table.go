package struct2table

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"reflect"

	"github.com/ddkwork/golibrary/mylog"

	"cogentcore.org/cogent/ai/table"
)

type (
	Type interface {
		reflect.Kind
	}
	StructTreeMarshaler interface {
		MarshalStructTree(structPoint any) (tree table.Node[Object], ok bool)
	}
	StructTreeUnmarshaler interface {
		UnmarshalStructTree(tree *table.Node[Object]) (structPoint any, ok bool)
	}
	Interface_ interface {
		StructTreeMarshaler
		StructTreeUnmarshaler
		encoding.BinaryMarshaler
		encoding.BinaryUnmarshaler
		xml.Marshaler
		xml.Unmarshaler
		json.Marshaler
		json.Unmarshaler
	}
	Object struct {
		Field
		children []Object
	}
	Field struct {
		Key   reflect.Type
		Value reflect.Value
	}
)

//todo check Interface_

func NewObject() *Object { return &Object{} }

func (o *Object) MarshalStructTree(structPoint any) (tree table.Node[Object], ok bool) {
	typeOf := reflect.TypeOf(structPoint)
	visibleFields := reflect.VisibleFields(typeOf)
	for i, field := range visibleFields {
		println(i)
		mylog.Struct(field)
	}
	//check file type
	return
}

func (o *Object) UnmarshalStructTree(tree *Object) (structPoint any, ok bool) {
	//TODO implement me
	panic("implement me")
}

func (o *Object) MarshalBinary() (data []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (o *Object) UnmarshalBinary(data []byte) error {
	//TODO implement me
	panic("implement me")
}

func (o *Object) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	//TODO implement me
	panic("implement me")
}

func (o *Object) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	//TODO implement me
	panic("implement me")
}

func (o *Object) MarshalJSON() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (o *Object) UnmarshalJSON(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}
