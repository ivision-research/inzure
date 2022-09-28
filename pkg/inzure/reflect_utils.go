package inzure

import (
	"bufio"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

func getBaseType(ty reflect.Type) reflect.Type {
	for ty.Kind() == reflect.Ptr || ty.Kind() == reflect.Slice {
		ty = ty.Elem()
	}
	return ty
}

func typeHasMethod(ty reflect.Type, method string, nonPtr bool) bool {
	if ty.Kind() != reflect.Struct {
		ty = getBaseType(ty)
		if ty.Kind() != reflect.Struct {
			return false
		}
	}
	_, has := ty.MethodByName(method)
	if !has && nonPtr {
		return false
	}
	_, has = reflect.PtrTo(ty).MethodByName(method)
	return has
}

// derefPtr will keep calling Elem() until we no longer have a reflect.Ptr
// Kind()
func derefPtr(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

// getField attempts to get a field from a struct Value
func getField(s string, v reflect.Value) (reflect.Value, error) {
	v = derefPtr(v)
	if v.Kind() != reflect.Struct {
		return reflect.ValueOf(nil), fmt.Errorf("can't get field %s in nonstruct: %s", s, v.Kind())
	}
	if strings.Contains(s, ".") {
		return getNestedField(s, v)
	}
	return v.FieldByName(s), nil
}

// getNestedField will attempt to find a field formatted like Foo.Bar.Baz in
// the given Value. getNestedField should only be called from getField!
func getNestedField(s string, v reflect.Value) (ret reflect.Value, err error) {
	scan := bufio.NewScanner(strings.NewReader(s))
	scan.Split(dotScanFunc)
	ret = v
	for scan.Scan() {
		ret, err = getField(scan.Text(), ret)
		if err != nil {
			return
		}
	}
	return
}

func nopackageStructName(t reflect.Type) string {
	name := t.Name()
	idx := strings.LastIndex(name, ".")
	if idx == -1 {
		return name
	}
	return name[idx+1:]
}

func dotScanFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for width, i := 0, 0; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if r == '.' {
			return i + width, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data[:], nil
	}
	return 0, nil, nil
}
