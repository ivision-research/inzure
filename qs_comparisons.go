package inzure

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type QSField struct {
	Name string

	IsArray  bool
	ArraySel QSArraySelT

	IsMethod          bool
	MethodNeedsPtr    bool
	MethodReturnIndex int
	MethodArgs        []reflect.Value

	Next *QSField
}

func qsStringValues(v []reflect.Value) string {
	if v == nil {
		return ""
	}
	s := make([]string, len(v))
	for i, e := range v {
		e = derefPtr(e)
		val := e.Interface()
		vs, is := val.(string)
		if is {
			s[i] = strconv.Quote(vs)
		} else {
			s[i] = fmt.Sprintf("%v", val)
		}
	}
	return strings.Join(s, ", ")
}

func (f *QSField) String() string {
	s := fmt.Sprintf(".%s", f.Name)
	if f.IsArray {
		s += fmt.Sprintf("[%s]", f.ArraySel)
	} else if f.IsMethod {
		s += fmt.Sprintf("(%s)", qsStringValues(f.MethodArgs))
	}
	if f.Next != nil {
		s += f.Next.String()
	}
	return s
}

func qsNilVal() reflect.Value {
	return reflect.ValueOf(nil)
}

func qsSelFromStruct(v reflect.Value, n string) (reflect.Value, error) {
	v = derefPtr(v)
	if v.Kind() != reflect.Struct {
		return qsNilVal(), fmt.Errorf("can't get field %s from nonstruct %v", n, v.Type())
	}
	f := v.FieldByName(n)
	if !f.IsValid() {
		return qsNilVal(), fmt.Errorf("type %v has no field %s", v.Type(), n)
	}
	return f, nil
}

func qsSelFromArrayIdx(v reflect.Value, n string, idx int) (reflect.Value, error) {
	v = derefPtr(v)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return qsNilVal(), fmt.Errorf("%v is not a slice", v.Type())
	}
	l := v.Len()
	if idx >= l {
		return qsNilVal(), fmt.Errorf("index %d is out bounds", idx)
	}
	return qsSelFromStruct(v.Index(idx), n)
}

func qsSelFromArray(v reflect.Value, n string) ([]reflect.Value, error) {
	v = derefPtr(v)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("%v is not a slice (Kind: %v)", v.Type(), v.Kind())
	}
	l := v.Len()
	var err error
	ret := make([]reflect.Value, l)
	for i := 0; i < l; i++ {
		e := v.Index(i)
		ret[i], err = qsSelFromStruct(e, n)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func derefTypePtr(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func (qsc *QSComparer) getCmpFunc(t reflect.Type) (qsCmpFunc, error) {
	t = derefTypePtr(t)
	for f := &qsc.Fields; f != nil; f = f.Next {
		if f.IsMethod {
			if f.Next != nil {
				//if i < end {
				return nil, errors.New("methods can only come at the end")
			}
			m, has := t.MethodByName(f.Name)
			if !has {
				// Try the pointer
				t = reflect.PtrTo(t)
				m, has = t.MethodByName(f.Name)
				if !has {
					return nil, fmt.Errorf(
						"type %v has no method %s", t, f.Name,
					)
				}
				f.MethodNeedsPtr = true
			}
			qsc.fun = m.Func
			nInParams := qsc.fun.Type().NumIn()
			if nInParams-1 != len(f.MethodArgs) {
				return nil, fmt.Errorf(
					"didn't pass enough method arguments for %s: %d != %d",
					f.Name, nInParams, len(f.MethodArgs),
				)
			}
			// Note we always just grab the first parameter
			nOutParams := qsc.fun.Type().NumOut()
			if nOutParams == 0 {
				return nil, fmt.Errorf(
					"method %s doesn't return anything", f.Name,
				)
			} else {
				t = qsc.fun.Type().Out(0)
			}
		} else {
			if t.Kind() != reflect.Struct {
				return nil, fmt.Errorf(
					"can't get field %s on nonstruct %s", f.Name, t,
				)
			}
			tmp, has := t.FieldByName(f.Name)
			if !has {
				return nil, fmt.Errorf("type %v has no field %s", t, f.Name)
			}
			t = derefTypePtr(tmp.Type)
			if f.IsArray {
				if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
					return nil, fmt.Errorf(
						"field %s wasn't actually a slice (%v)",
						f.Name, t,
					)
				}
				if err := qsCheckArraySel(f.ArraySel); err != nil {
					return nil, err
				}
				switch f.ArraySel {
				case QSArraySelLen:
					if f.Next != nil {
						return nil, fmt.Errorf(
							"%s selector can only be used last", QSArraySelLen,
						)
					}
					t = reflect.TypeOf(int64(0))
				default:
					t = t.Elem()
				}
			}
		}
	}
	to := reflect.ValueOf(qsc.To)
	if !t.ConvertibleTo(to.Type()) {
		return nil, fmt.Errorf("incompatible types: %v and %v", t, to.Type())
	}
	iface := to.Interface()
	var cf qsCmpFunc
	var err error
	switch c := iface.(type) {
	case UnknownBool:
		cf, err = qsc.cmpForUB(c)
	case string:
		cf, err = qsc.cmpForString(c)
	case int:
		cf, err = qsc.cmpForInt64(int64(c))
	case int8:
		cf, err = qsc.cmpForInt64(int64(c))
	case int16:
		cf, err = qsc.cmpForInt64(int64(c))
	case int32:
		cf, err = qsc.cmpForInt64(int64(c))
	case int64:
		cf, err = qsc.cmpForInt64(int64(c))
	case uint:
		cf, err = qsc.cmpForUint64(uint64(c))
	case uint8:
		cf, err = qsc.cmpForUint64(uint64(c))
	case uint16:
		cf, err = qsc.cmpForUint64(uint64(c))
	case uint32:
		cf, err = qsc.cmpForUint64(uint64(c))
	case uint64:
		cf, err = qsc.cmpForUint64(uint64(c))
	case bool:
		cf, err = qsc.cmpForBool(c)
	default:
		return nil, fmt.Errorf("can't handle type %T", c)
	}
	return cf, err
}

type IQSComparer interface {
	Compare(reflect.Value) (bool, error)
}

func (qsc *QSComparer) Compare(base reflect.Value) (bool, error) {
	base = derefPtr(base)
	// getCmpFunc also does some validation
	cmpFunc, err := qsc.getCmpFunc(base.Type())
	if err != nil {
		return false, err
	}
	return qsc.compareRecurse(base, &qsc.Fields, cmpFunc)
}

// compmareRecurse is just the recursive function that will walk the Fields
// until the end and figure out how to make the comparison. We need to do that
// this way because of .Array[...] potential.
func (qsc *QSComparer) compareRecurse(
	base reflect.Value, field *QSField, cmpFunc qsCmpFunc,
) (bool, error) {
	rv := derefPtr(base)
	for f := field; f != nil; f = f.Next {
		// Methods will be fairly simple here, we just ensure that it is at
		// the end and do the comparison with our built up cmpFunc.
		if f.IsMethod {
			if f.Next != nil {
				return false, errors.New("methods can only be called at the end")
			}
			return qsc.compareMethod(rv, f, cmpFunc)
		} else {
			// If it isn't a method we grab the field and check if it is an
			// array or not.
			var err error
			rv, err = qsSelFromStruct(rv, f.Name)
			if err != nil {
				return false, err
			}
			// Get out if it isn't an array, we're done
			if !f.IsArray {
				continue
			}
			// Otherwise check the selector, if it is just a single index this
			// can be treated like a normal field. Otherwise we have to create
			// our recursive function.
			sel := f.ArraySel
			f = f.Next
			if sel < 0 {
				// Given f.next is nil we need to just run the comparison
				if f == nil {
					return doSpecialArrayComparison(rv, sel, cmpFunc)
				}
				// Otherwise we need to grab every potential value from mthe
				// array and run a second set of recursion using it. This
				// will be handled in the appropriate functions in the switch
				// case.
				if err := qsCheckArray(rv); err != nil {
					return false, err
				}
				switch sel {
				case QSArraySelAll:
					return qsc.compareRecurseArrAll(rv, f, cmpFunc)
				case QSArraySelAny:
					return qsc.compareRecurseArrAny(rv, f, cmpFunc)
				case QSArraySelLen:
					return false, fmt.Errorf("%s selector can only be used last", QSArraySelLen)
				default:
					return false, fmt.Errorf("%d is a bad array selector", sel)
				}
			} else {
				// When f.next is nil we have something like:
				// .A.B.C[idx] and want to compare that. So we're done.
				if f == nil {
					return cmpFunc(rv)
				}
				// Otherwise treat it normally
				var err error
				rv, err = qsSelFromArrayIdx(rv, f.Name, int(sel))
				if err != nil {
					return false, err
				}
			}
		}
	}
	return cmpFunc(rv)
}

func (qsc *QSComparer) compareMethod(
	base reflect.Value, m *QSField, cmpFunc qsCmpFunc,
) (bool, error) {
	if m.MethodNeedsPtr && base.Kind() != reflect.Ptr {
		if !base.CanAddr() {
			val := reflect.New(base.Type())
			reflect.Indirect(val).Set(base)
			base = val
		} else {
			base = base.Addr()
		}
	}
	args := make([]reflect.Value, len(m.MethodArgs)+1)
	args[0] = base
	copy(args[1:], m.MethodArgs)
	retCount := qsc.fun.Type().NumOut()
	checkErr := retCount > 1
	if err := qsVerifyMethodArgs(qsc.fun, m.MethodReturnIndex, args); err != nil {
		return false, err
	}
	ret := qsc.fun.Call(args)
	if !checkErr {
		return cmpFunc(ret[m.MethodReturnIndex])
	}
	for _, r := range ret {
		if r.CanInterface() {
			v, is := r.Interface().(error)
			if is {
				if v != nil {
					return false, v
				}
			}
		} // TODO else error maybe?
	}
	return cmpFunc(ret[m.MethodReturnIndex])
}

func qsVerifyMethodArgs(fun reflect.Value, idx int, args []reflect.Value) error {
	t := fun.Type()
	if idx >= t.NumOut() {
		return fmt.Errorf(
			"can't access return index %d (max: %d)",
			idx, t.NumOut()-1,
		)
	}
	inp := t.NumIn()
	if len(args) != inp {
		return fmt.Errorf(
			"method %s takes %d arguments but %d given",
			t.Name(), inp, len(args),
		)
	}
	for i := 0; i < inp; i++ {
		it := t.In(i)
		if it != args[i].Type() {
			if args[i].Type().ConvertibleTo(it) {
				args[i] = args[i].Convert(it)
			} else {
				return fmt.Errorf(
					"method type mismatch on argument %d: %s != %s",
					i, args[i].Type(), it,
				)
			}
		}
	}
	return nil
}

func qsCheckArray(v reflect.Value) error {
	k := v.Kind()
	if k == reflect.Array || k == reflect.Slice {
		return nil
	}
	return fmt.Errorf("%v is not an array or slice", v.Type())
}

func qsCheckArraySel(sel QSArraySelT) error {
	if sel < 0 && sel < QSArraySelUk {
		return fmt.Errorf("%d is a bad array selector", sel)
	}
	return nil
}

func doSpecialArrayComparison(
	v reflect.Value, sel QSArraySelT, cmpFunc qsCmpFunc,
) (bool, error) {
	if err := qsCheckArray(v); err != nil {
		return false, err
	}
	l := v.Len()
	switch sel {
	case QSArraySelLen:
		return cmpFunc(reflect.ValueOf(l))
	case QSArraySelAny:
		if l == 0 {
			return false, nil
		}
		for i := 0; i < l; i++ {
			passed, err := cmpFunc(v.Index(i))
			if err != nil {
				return false, err
			}
			if passed {
				return true, nil
			}
		}
		return false, nil
	case QSArraySelAll:
		if l == 0 {
			return false, nil
		}
		for i := 0; i < l; i++ {
			passed, err := cmpFunc(v.Index(i))
			if err != nil {
				return false, err
			}
			if !passed {
				return false, nil
			}
		}
		return true, nil
	default:
		panic("unreachable")
	}
}

func (qsc *QSComparer) compareRecurseArrAny(
	v reflect.Value, f *QSField, cmpFunc qsCmpFunc,
) (bool, error) {
	l := v.Len()
	if l == 0 {
		return false, nil
	}
	for i := 0; i < l; i++ {
		e := v.Index(i)
		passed, err := qsc.compareRecurse(e, f, cmpFunc)
		if err != nil {
			return false, err
		}
		if passed {
			return true, nil
		}
	}
	return false, nil
}

func (qsc *QSComparer) compareRecurseArrAll(
	v reflect.Value, f *QSField, cmpFunc qsCmpFunc,
) (bool, error) {
	l := v.Len()
	if l == 0 {
		return false, nil
	}
	for i := 0; i < l; i++ {
		e := v.Index(i)
		passed, err := qsc.compareRecurse(e, f, cmpFunc)
		if err != nil {
			return false, err
		}
		if !passed {
			return false, nil
		}
	}
	return true, nil
}

type QSComparer struct {
	Fields QSField
	Op     QSOpT
	To     interface{}

	fun reflect.Value
}

func (qsc *QSComparer) String() string {
	s := fmt.Sprintf("%s %s ", qsc.Fields.String(), qsc.Op)
	if reflect.TypeOf(qsc.To).Kind() == reflect.String {
		s += fmt.Sprintf("\"%s\"", qsc.To)
	} else {
		s += fmt.Sprintf("%v", qsc.To)
	}
	return s
}

type qsCmpFunc func(reflect.Value) (bool, error)

func (qsc *QSComparer) cmpForUB(to UnknownBool) (qsCmpFunc, error) {
	var cf func(left UnknownBool) bool
	switch qsc.Op {
	case QSOpEq:
		cf = func(l UnknownBool) bool { return l == to }
	case QSOpNe:
		cf = func(l UnknownBool) bool { return l != to }
	case QSOpGt:
		cf = func(l UnknownBool) bool { return l > to }
	case QSOpGte:
		cf = func(l UnknownBool) bool { return l >= to }
	case QSOpLt:
		cf = func(l UnknownBool) bool { return l < to }
	case QSOpLte:
		cf = func(l UnknownBool) bool { return l <= to }
	default:
		return nil, fmt.Errorf("UnknownBool types don't support %s comparison", qsc.Op)
	}

	return func(v reflect.Value) (bool, error) {
		v = derefPtr(v)
		if !v.CanInterface() {
			return false, fmt.Errorf("can't interface %v", v.Type())
		}
		l, ok := v.Interface().(UnknownBool)
		if !ok {
			return false, fmt.Errorf("type %v is not an UnknownBool", v.Type())
		}
		return cf(l), nil
	}, nil
}

func (qsc *QSComparer) cmpForString(to string) (qsCmpFunc, error) {
	var cf func(left string) bool
	switch qsc.Op {
	case QSOpEq:
		cf = func(l string) bool { return l == to }
	case QSOpNe:
		cf = func(l string) bool { return l != to }
	case QSOpLike, QSOpNotLike:
		re, err := regexp.Compile(to)
		if err != nil {
			return nil, err
		}
		negate := qsc.Op == QSOpNotLike
		cf = func(l string) bool {
			v := re.MatchString(l)
			if negate {
				return !v
			}
			return v
		}
	default:
		return nil, fmt.Errorf("string types don't support %s comparison", qsc.Op)
	}

	return func(v reflect.Value) (bool, error) {
		v = derefPtr(v)
		if !v.CanInterface() {
			return false, fmt.Errorf("can't interface %v", v.Type())
		}
		l, ok := v.Interface().(string)
		if !ok {
			return false, fmt.Errorf("type %v is not a string", v.Type())
		}
		return cf(l), nil
	}, nil
}

func (qsc *QSComparer) cmpForBool(to bool) (qsCmpFunc, error) {
	var cf func(left bool) bool
	switch qsc.Op {
	case QSOpEq:
		cf = func(l bool) bool { return l == to }
	case QSOpNe:
		cf = func(l bool) bool { return l != to }
	default:
		return nil, fmt.Errorf("bool types don't support %s comparison", qsc.Op)
	}

	return func(v reflect.Value) (bool, error) {
		v = derefPtr(v)
		if !v.CanInterface() {
			return false, fmt.Errorf("can't interface %v", v.Type())
		}
		l, ok := v.Interface().(bool)
		if !ok {
			return false, fmt.Errorf("type %v is not a bool", v.Type())
		}
		return cf(l), nil
	}, nil
}

func (qsc *QSComparer) cmpForInt64(to int64) (qsCmpFunc, error) {
	var cf func(left int64) bool
	switch qsc.Op {
	case QSOpEq:
		cf = func(l int64) bool { return l == to }
	case QSOpNe:
		cf = func(l int64) bool { return l != to }
	case QSOpGt:
		cf = func(l int64) bool { return l > to }
	case QSOpGte:
		cf = func(l int64) bool { return l >= to }
	case QSOpLt:
		cf = func(l int64) bool { return l < to }
	case QSOpLte:
		cf = func(l int64) bool { return l <= to }
	default:
		return nil, fmt.Errorf("int64 types don't support %s comparison", qsc.Op)
	}

	return func(v reflect.Value) (bool, error) {
		v = derefPtr(v)
		if !v.CanInterface() {
			return false, fmt.Errorf("can't interface %v", v.Type())
		}
		switch val := v.Interface().(type) {
		case int:
			return cf(int64(val)), nil
		case int8:
			return cf(int64(val)), nil
		case int16:
			return cf(int64(val)), nil
		case int32:
			return cf(int64(val)), nil
		case int64:
			return cf(int64(val)), nil
		default:
			t := reflect.TypeOf(to)
			if v.Type().ConvertibleTo(t) {
				v = v.Convert(t)
				if cv, ok := v.Interface().(int64); ok {
					return cf(cv), nil
				}
			}
			return false, fmt.Errorf("type %v is not convertible to an int64", v.Type())
		}
	}, nil
}

func (qsc *QSComparer) cmpForUint64(to uint64) (qsCmpFunc, error) {
	var cf func(left uint64) bool
	switch qsc.Op {
	case QSOpEq:
		cf = func(l uint64) bool { return l == to }
	case QSOpNe:
		cf = func(l uint64) bool { return l != to }
	case QSOpGt:
		cf = func(l uint64) bool { return l > to }
	case QSOpGte:
		cf = func(l uint64) bool { return l >= to }
	case QSOpLt:
		cf = func(l uint64) bool { return l < to }
	case QSOpLte:
		cf = func(l uint64) bool { return l <= to }
	default:
		return nil, fmt.Errorf("uint64 types don't support %s comparison", qsc.Op)
	}

	return func(v reflect.Value) (bool, error) {
		v = derefPtr(v)
		if !v.CanInterface() {
			return false, fmt.Errorf("can't interface %v", v.Type())
		}
		switch val := v.Interface().(type) {
		case uint:
			return cf(uint64(val)), nil
		case uint8:
			return cf(uint64(val)), nil
		case uint16:
			return cf(uint64(val)), nil
		case uint32:
			return cf(uint64(val)), nil
		case uint64:
			return cf(uint64(val)), nil
		default:
			t := reflect.TypeOf(to)
			if v.Type().ConvertibleTo(t) {
				v = v.Convert(t)
				if cv, ok := v.Interface().(uint64); ok {
					return cf(cv), nil
				}
			}
			return false, fmt.Errorf("type %v is not convertible to an uint64", v.Type())
		}
	}, nil
}
