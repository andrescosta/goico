package reflectico

import (
	"reflect"
)

func SetFieldString[T any](q T, field string, value string) {
	reflect.ValueOf(q).Elem().FieldByName(field).SetString(value)
}
