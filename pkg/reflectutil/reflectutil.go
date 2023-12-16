package reflectutil

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
func GetFieldString[T any](q T, field string) string {
	u := reflect.ValueOf(q).Elem().FieldByName(field).String()
	return u
}

func CanConvert[S any](i interface{}) bool {
	var a S
	t1 := reflect.TypeOf(i)
	t2 := reflect.TypeOf(a)
	return t1.ConvertibleTo(t2)
}

func NewInstance[T any](t T) T {
	s := reflect.ValueOf(t)
	if s.Kind() == reflect.Ptr {
		s = reflect.Indirect(s)
	}
	n := reflect.New(s.Type()).Interface().(T)
	return n
}
