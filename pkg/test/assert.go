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
			t.Errorf("expected <nil> got %v", o)
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
			t.Errorf("expected data got empty reference")
			t.FailNow()
		}
	case reflect.String:
		if obj.String() == "" {
			t.Errorf("expected data got empty string")
			t.FailNow()
		}
	default:
		t.Errorf("expected slice/chan/map")
		t.FailNow()
	}
}

func Len(t *testing.T, o interface{}, len1 int) {
	t.Helper()
	obj := reflect.ValueOf(o)
	//exhaustive:ignore only for Slice, Map and Chan
	switch obj.Kind() {
	case reflect.Slice, reflect.Map, reflect.Chan:
		if obj.Len() != len1 {
			t.Errorf("expected len equal to %d got %d", len1, obj.Len())
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

func Equals(t *testing.T, i any, l any) {
	t.Helper()
	if !reflect.DeepEqual(i, l) {
		t.Errorf("expected the same values, got %v - %v", i, l)
		t.FailNow()
	}
}

func NotEquals(t *testing.T, i any, l any) {
	t.Helper()
	if reflect.DeepEqual(i, l) {
		t.Errorf("expected different values, got %v - %v", i, l)
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
