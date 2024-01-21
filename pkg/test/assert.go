package test

import (
	"errors"
	"reflect"
	"testing"
)

func Nil(t *testing.T, o interface{}, msg ...string) {
	t.Helper()
	if o != nil {
		if len(msg) > 0 {
			t.Errorf(msg[0])
		} else {
			t.Errorf("expected <nil> got %#v", o)
		}
		t.FailNow()
	}
}

func NotNil(t *testing.T, o interface{}) {
	t.Helper()
	if o == nil {
		t.Errorf("not expected <nil>")
		t.FailNow()
	}
}

func ErrorIs(t *testing.T, err error, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("expected %v got %v", target, err)
		t.FailNow()
	}
}

func ErrorNotIs(t *testing.T, err error, target error) {
	t.Helper()
	if errors.Is(err, target) {
		t.Errorf("Expected error: %v,got %v", target, err)
		t.FailNow()
	}
}

func NotEmpty(t *testing.T, o interface{}) {
	t.Helper()
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
	t.Helper()
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
	t.Helper()
	if i != l {
		t.Errorf("expected %d got %d", i, l)
		t.FailNow()
	}
}

func NotIsLen(t *testing.T, o interface{}, l int) {
	t.Helper()
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
