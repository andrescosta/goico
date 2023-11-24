package reflectico

import (
	"fmt"
	"reflect"
)

func SetFieldString[T any](q T, field string, value string) {
	reflect.ValueOf(q).Elem().FieldByName(field).SetString(value)
}

func GetFieldUInt[T any](q T, field string) uint64 {
	s := reflect.ValueOf(q).Elem().FieldByName(field).String()
	var n uint64
	fmt.Sscan(s, &n)
	return n
}
