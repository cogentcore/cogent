// Code generated by "stringer -type=PaintTypes"; DO NOT EDIT.

package grid

import (
	"errors"
	"strconv"
)

var _ = errors.New("dummy error")

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[PaintOff-0]
	_ = x[PaintSolid-1]
	_ = x[PaintLinear-2]
	_ = x[PaintRadial-3]
	_ = x[PaintInherit-4]
	_ = x[PaintTypesN-5]
}

const _PaintTypes_name = "PaintOffPaintSolidPaintLinearPaintRadialPaintInheritPaintTypesN"

var _PaintTypes_index = [...]uint8{0, 8, 18, 29, 40, 52, 63}

func (i PaintTypes) String() string {
	if i < 0 || i >= PaintTypes(len(_PaintTypes_index)-1) {
		return "PaintTypes(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _PaintTypes_name[_PaintTypes_index[i]:_PaintTypes_index[i+1]]
}

func (i *PaintTypes) FromString(s string) error {
	for j := 0; j < len(_PaintTypes_index)-1; j++ {
		if s == _PaintTypes_name[_PaintTypes_index[j]:_PaintTypes_index[j+1]] {
			*i = PaintTypes(j)
			return nil
		}
	}
	return errors.New("String: " + s + " is not a valid option for type: PaintTypes")
}
