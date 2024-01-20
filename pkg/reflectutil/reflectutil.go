package reflectutil

import (
	"reflect"
	"runtime"
	"strings"
)

func FuncName(fn interface{}) string {
	ptr := reflect.ValueOf(fn).Pointer()
	n := runtime.FuncForPC(ptr).Name()
	dot := strings.LastIndex(n, ".")
	if dot > -1 {
		n = n[dot+1:]
	}
	return n
}
