// Code generated by 'yaegi extract cogentcore.org/core/tensor/table'. DO NOT EDIT.

package imports

import (
	"cogentcore.org/core/tensor/table"
	"reflect"
)

func init() {
	Symbols["cogentcore.org/core/tensor/table/table"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"AddAggName":             reflect.ValueOf(table.AddAggName),
		"Ascending":              reflect.ValueOf(table.Ascending),
		"ColumnNameOnly":         reflect.ValueOf(table.ColumnNameOnly),
		"Comma":                  reflect.ValueOf(table.Comma),
		"ConfigFromDataValues":   reflect.ValueOf(table.ConfigFromDataValues),
		"ConfigFromHeaders":      reflect.ValueOf(table.ConfigFromHeaders),
		"ConfigFromTableHeaders": reflect.ValueOf(table.ConfigFromTableHeaders),
		"Contains":               reflect.ValueOf(table.Contains),
		"DelimsN":                reflect.ValueOf(table.DelimsN),
		"DelimsValues":           reflect.ValueOf(table.DelimsValues),
		"Descending":             reflect.ValueOf(table.Descending),
		"Detect":                 reflect.ValueOf(table.Detect),
		"DetectTableHeaders":     reflect.ValueOf(table.DetectTableHeaders),
		"Equals":                 reflect.ValueOf(table.Equals),
		"Headers":                reflect.ValueOf(table.Headers),
		"IgnoreCase":             reflect.ValueOf(table.IgnoreCase),
		"InferDataType":          reflect.ValueOf(table.InferDataType),
		"NewIndexView":           reflect.ValueOf(table.NewIndexView),
		"NewSliceTable":          reflect.ValueOf(table.NewSliceTable),
		"NewTable":               reflect.ValueOf(table.NewTable),
		"NoHeaders":              reflect.ValueOf(table.NoHeaders),
		"ShapeFromString":        reflect.ValueOf(table.ShapeFromString),
		"Space":                  reflect.ValueOf(table.Space),
		"Tab":                    reflect.ValueOf(table.Tab),
		"TableColumnType":        reflect.ValueOf(table.TableColumnType),
		"TableHeaderChar":        reflect.ValueOf(table.TableHeaderChar),
		"TableHeaderToType":      reflect.ValueOf(&table.TableHeaderToType).Elem(),
		"UpdateSliceTable":       reflect.ValueOf(table.UpdateSliceTable),
		"UseCase":                reflect.ValueOf(table.UseCase),

		// type definitions
		"Delims":         reflect.ValueOf((*table.Delims)(nil)),
		"Filterer":       reflect.ValueOf((*table.Filterer)(nil)),
		"IndexView":      reflect.ValueOf((*table.IndexView)(nil)),
		"LessFunc":       reflect.ValueOf((*table.LessFunc)(nil)),
		"SplitAgg":       reflect.ValueOf((*table.SplitAgg)(nil)),
		"Splits":         reflect.ValueOf((*table.Splits)(nil)),
		"SplitsLessFunc": reflect.ValueOf((*table.SplitsLessFunc)(nil)),
		"Table":          reflect.ValueOf((*table.Table)(nil)),
	}
}
