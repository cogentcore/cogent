package struct2table

import (
	"reflect"
	"strconv"

	"github.com/ddkwork/golibrary/mylog"
	"google.golang.org/protobuf/encoding/protowire"
)

// https://521github.com/google/go-cmp/blob/c3ad8435e7bef96af35732bc0789e5a2278c6d5f/cmp/report_reflect.go#L302
var (
	anyType    = reflect.TypeOf((*interface{})(nil)).Elem()
	stringType = reflect.TypeOf((*string)(nil)).Elem()
	bytesType  = reflect.TypeOf((*[]byte)(nil)).Elem()
	byteType   = reflect.TypeOf((*byte)(nil)).Elem()
)

type Kind uint

const (
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Pointer
	Slice
	String
	Struct
	UnsafePointer
)

func (k Kind) String() string {
	if uint(k) < uint(len(kindNames)) {
		return kindNames[uint(k)]
	}
	return "kind" + strconv.Itoa(int(k))
}

var kindNames = []string{
	Invalid:       "invalid",
	Bool:          "bool",
	Int:           "int",
	Int8:          "int8",
	Int16:         "int16",
	Int32:         "int32",
	Int64:         "int64",
	Uint:          "uint",
	Uint8:         "uint8",
	Uint16:        "uint16",
	Uint32:        "uint32",
	Uint64:        "uint64",
	Uintptr:       "uintptr",
	Float32:       "float32",
	Float64:       "float64",
	Complex64:     "complex64",
	Complex128:    "complex128",
	Array:         "array",
	Chan:          "chan",
	Func:          "func",
	Interface:     "interface",
	Map:           "map",
	Pointer:       "ptr",
	Slice:         "slice",
	String:        "string",
	Struct:        "struct",
	UnsafePointer: "unsafe.Pointer",
}

var (
	reflectKind2KindMap = map[reflect.Kind]Kind{
		reflect.Invalid:       Invalid,
		reflect.Bool:          Bool,
		reflect.Int:           Int,
		reflect.Int8:          Int8,
		reflect.Int16:         Int16,
		reflect.Int32:         Int32,
		reflect.Int64:         Int64,
		reflect.Uint:          Uint,
		reflect.Uint8:         Uint8,
		reflect.Uint16:        Uint16,
		reflect.Uint32:        Uint32,
		reflect.Uint64:        Uint64,
		reflect.Uintptr:       Uintptr,
		reflect.Float32:       Float32,
		reflect.Float64:       Float64,
		reflect.Complex64:     Complex64,
		reflect.Complex128:    Complex128,
		reflect.Array:         Array,
		reflect.Chan:          Chan,
		reflect.Func:          Func,
		reflect.Interface:     Interface,
		reflect.Map:           Map,
		reflect.Pointer:       Pointer,
		reflect.Slice:         Slice,
		reflect.String:        String,
		reflect.Struct:        Struct,
		reflect.UnsafePointer: UnsafePointer,
	}

	kind2PbType = map[Kind]protowire.Type{ //todo fix it
		Invalid:       protowire.VarintType,
		Bool:          protowire.VarintType,
		Int:           protowire.VarintType,
		Int8:          protowire.VarintType,
		Int16:         protowire.VarintType,
		Int32:         protowire.VarintType,
		Int64:         protowire.VarintType,
		Uint:          protowire.VarintType,
		Uint8:         protowire.VarintType,
		Uint16:        protowire.VarintType,
		Uint32:        protowire.VarintType,
		Uint64:        protowire.VarintType,
		Uintptr:       protowire.VarintType,
		Float32:       protowire.VarintType,
		Float64:       protowire.VarintType,
		Complex64:     protowire.VarintType,
		Complex128:    protowire.VarintType,
		Array:         protowire.VarintType,
		Chan:          protowire.VarintType,
		Func:          protowire.VarintType,
		Interface:     protowire.VarintType,
		Map:           protowire.VarintType,
		Pointer:       protowire.VarintType,
		Slice:         protowire.VarintType,
		String:        protowire.VarintType,
		Struct:        protowire.VarintType,
		UnsafePointer: protowire.VarintType,
	}
)

func reflectKind2Kind(k reflect.Kind) Kind { return reflectKind2KindMap[k] }
func Kind2PbType(k Kind) protowire.Type    { return kind2PbType[k] }

func (k Kind) IsValid() bool {
	switch k {
	case Invalid,
		Uintptr,
		Chan,
		UnsafePointer,
		Pointer,
		Interface,
		Func,
		Complex64,
		Complex128:
		mylog.Error(k.String() + " not support")
		return false
	}
	return true
}

//func (k Kind) IsPacked() bool {
//}
//func (k Kind) IsList() bool {
//}
//func (k Kind) IsMap() bool {
//}

func Marshe(obj any) {
	fields := reflect.VisibleFields(reflect.TypeOf(obj))
	for _, field := range fields {
		k := reflectKind2Kind(field.Type.Kind())
		if !k.IsValid() {
			continue
		}
		switch k {
		case Invalid:
		case Bool:
		case Int:
		case Int8:
		case Int16:
		case Int32:
		case Int64:
		case Uint:
		case Uint8:
		case Uint16:
		case Uint32:
		case Uint64:
		case Uintptr:
		case Float32:
		case Float64:
		case Complex64:
		case Complex128:
		case Array: //todo 新函数递归
		case Chan:
		case Func:
		case Interface:
		case Map: //todo 新函数递归
		case Pointer:
		case Slice: //todo 新函数递归
		case String:
		case Struct: //todo 新函数递归
		case UnsafePointer:
		}
	}
}

//func marshal()            {} //序列化结构体
//func marshalField()       {} //遍历结构体字段
//func marshalList()        {} //Array Slice
//func marshalMap()         {} //Map
//func MarshalAppend()      {}
//func MarshalState()       {}
//func marshalMessage()     {} //Struct ?
//func marshalMessageSlow() {}
//

//func Marshal(m Message) ([]byte, error)
//func MarshalAppend(b []byte, m Message) ([]byte, error)
//func MarshalState(in protoiface.MarshalInput) (protoiface.MarshalOutput, error)
//func marshal(b []byte, m protoreflect.Message) (out protoiface.MarshalOutput, err error)
//func marshalMessage(b []byte, m protoreflect.Message) ([]byte, error)
//func marshalMessageSlow(b []byte, m protoreflect.Message) ([]byte, error)
//func marshalField(b []byte, fd protoreflect.FieldDescriptor, value protoreflect.Value) ([]byte, error)
//func marshalList(b []byte, fd protoreflect.FieldDescriptor, list protoreflect.List) ([]byte, error)
//func marshalMap(b []byte, fd protoreflect.FieldDescriptor, mapv protoreflect.Map) ([]byte, error)
//func marshalSingular(b []byte, fd protoreflect.FieldDescriptor, v protoreflect.Value) ([]byte, error)
//func sizeMessageSet(m protoreflect.Message) (size int)
//func marshalMessageSet(b []byte, m protoreflect.Message) ([]byte, error)
//func marshalMessageSetField(b []byte, fd protoreflect.FieldDescriptor, value protoreflect.Value) ([]byte, error)
//func Size(m Message) int
//func size(m protoreflect.Message) (size int)
//func sizeMessageSlow(m protoreflect.Message) (size int)
//func sizeField(fd protoreflect.FieldDescriptor, value protoreflect.Value) (size int)
//func sizeList(num protowire.Number, fd protoreflect.FieldDescriptor, list protoreflect.List) (size int)
//func sizeMap(num protowire.Number, fd protoreflect.FieldDescriptor, mapv protoreflect.Map) (size int)
//func sizeSingular(num protowire.Number, kind protoreflect.Kind, v protoreflect.Value) int
