package reflectico

import (
	"reflect"
)

func SetFieldString[T any](q T, field string, value string) {
	reflect.ValueOf(q).Elem().FieldByName(field).SetString(value)
}

func SetFieldUInt[T any](q T, field string, value *uint64) {
	reflect.ValueOf(q).Elem().FieldByName(field).Set(reflect.ValueOf(value))
}

func GetFieldUInt[T any](q T, field string) uint64 {
	u := reflect.ValueOf(q).Elem().FieldByName(field).Uint()
	return u
}
