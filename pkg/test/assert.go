package test

import (
	"errors"
	"reflect"
	"testing"
)

func Nil(t *testing.T, o interface{}, msg ...string) {
	if o != nil {
		if len(msg) > 0 {
			t.Error(msg[0])
		} else {
			t.Errorf("expected <nil> got %#v", o)
		}
		t.FailNow()
	}
}

func NotNil(t *testing.T, o interface{}) {
	if o == nil {
		t.Error("not expected <nil>")
		t.FailNow()
	}
}

func ErrorNotIs(t *testing.T, err error, errNotIs error) {
	if !errors.Is(err, errNotIs) {
		t.Errorf("Error not expected:%s", err)
		t.FailNow()
	}
}

func NotEmpty(t *testing.T, o interface{}) {
	obj := reflect.ValueOf(o)
	//exhaustive:ignore only for Slice, Map and Chan
	switch obj.Kind() {
	case reflect.Slice, reflect.Map, reflect.Chan:
		if obj.Len() == 0 {
			t.Errorf("expected data got empty slice")
			t.FailNow()
		}
	default:
		t.Errorf("expected slice/chan/map")
		t.FailNow()
	}
}

func Empty(t *testing.T, o interface{}) {
	obj := reflect.ValueOf(o)
	//exhaustive:ignore only for Slice, Map and Chan
	switch obj.Kind() {
	case reflect.Slice, reflect.Map, reflect.Chan:
		if obj.Len() != 0 {
			t.Errorf("expected an empty slice, got: %d", obj.Len())
			t.FailNow()
		}
	default:
		t.Errorf("expected slice/chan/map")
		t.FailNow()
	}
}

func Equals(t *testing.T, i int, l int) {
	if i != l {
		t.Errorf("expected %d got %d", i, l)
		t.FailNow()
	}
}

func NotIsLen(t *testing.T, o interface{}, l int) {
	NotNil(t, o)
	obj := reflect.ValueOf(o)
	//exhaustive:ignore only for Slice, Map and Chan
	switch obj.Kind() {
	case reflect.Slice, reflect.Map, reflect.Chan:
		if obj.Len() != l {
			t.Errorf("expected %d data got %d", l, obj.Len())
			t.FailNow()
		}
	case reflect.Ptr:
		if obj.IsNil() {
			t.Errorf("expected %d data got <nil>", l)
			t.FailNow()
		}
		d := obj.Elem().Interface()
		NotIsLen(t, d, l)
	default:
		t.Errorf("expected slice/chan/map")
		t.FailNow()
	}
}
