package inzure

import "reflect"

// These *FromPtr functions are fragile: only use if you know what you're doing.
//
// They're useful for pulling stuff out of Azure responses which use pointers
// to represent whether the value was present or not. You don't need to do:
//
//     if az.Foo != nil {
//         my.Foo = *az.Foo
//     }

func valFromPtr(into interface{}, ptr interface{}) {
	pV := reflect.ValueOf(ptr)
	if pV.IsNil() {
		return
	}
	iV := reflect.ValueOf(into)
	reflect.Indirect(iV).Set(reflect.Indirect(pV))
}

// same as val but calls FromAzure
func valFromPtrFromAzure(into interface{}, ptr interface{}) {
	pV := reflect.ValueOf(ptr)
	if pV.IsNil() {
		return
	}
	FromAzure := reflect.ValueOf(into).MethodByName("FromAzure")
	FromAzure.Call([]reflect.Value{pV})
}

func sliceFromPtr(into interface{}, ptr interface{}) {
	pV := reflect.ValueOf(ptr)
	if pV.IsNil() {
		return
	}
	pV = reflect.Indirect(pV)
	pL := pV.Len()
	if pL == 0 {
		return
	}
	deref := reflect.Indirect(reflect.ValueOf(into))
	for i := 0; i < pL; i++ {
		deref.Set(reflect.Append(deref, pV.Index(i)))
	}
}

// Same as the slice but calls FromAzure
/*
func sliceFromPtrFromAzure(into interface{}, ptr interface{}) {
	pV := reflect.ValueOf(ptr)
	if pV.IsNil() {
		return
	}
	pDeref := reflect.Indirect(pV)
	pL := pDeref.Len()
	if pL == 0 {
		return
	}
	deref := reflect.Indirect(reflect.ValueOf(into))
	et := deref.Elem().Type()
	for i := 0; i < pL; i++ {
		from := pDeref.Index(i)
		v := reflect.New(et)
		FromAzure := v.MethodByName("FromAzure")
		FromAzure([]reflect.Value{from})
		deref.Set(reflect.Append(deref, v))
	}
}
*/

// slicesEqual makes heavy use of reflection. It is trying to determine if two
// slices are equal in the following way:
//
// 1. Both non nil
// 2. Equal lengths
// 3. Equal elements (not sorted)
//    - Using discovered `Equals` method on elements
//    - Using `reflect.DeepEqual`
//
// This function is simply to let me be lazy.
func slicesEqual(s1 interface{}, s2 interface{}) bool {
	s1v := reflect.ValueOf(s1)
	s2v := reflect.ValueOf(s2)
	if s1v.Type() != s2v.Type() {
		return false
	}
	if s1v.Kind() != reflect.Slice || s2v.Kind() != reflect.Slice {
		return false
	}
	if s1v.IsNil() {
		if !s2v.IsNil() {
			return false
		}
	} else if s2v.IsNil() {
		return false
	}
	l := s1v.Len()
	if l != s2v.Len() {
		return false
	}
	if l == 0 {
		return true
	}
	noPtr := true
	et := s1v.Type().Elem()
	var eqFunc reflect.Value
	eqMethod, found := et.MethodByName("Equals")
	if !found {
		// Check the pointer
		et = reflect.PtrTo(et)
		eqMethod, found = et.MethodByName("Equals")
		// Set it to deep equal by default
		if !found {
			eqFunc = reflect.ValueOf(reflect.DeepEqual)
		} else {
			// Make sure the Equals func is sane
			expected := reflect.FuncOf(
				[]reflect.Type{reflect.TypeOf(et), reflect.TypeOf(et)},
				[]reflect.Type{reflect.TypeOf(true)},
				false,
			)
			eqFunc = eqMethod.Func
			if eqFunc.Type() != expected {
				eqFunc = reflect.ValueOf(reflect.DeepEqual)
			} else {
				noPtr = false
			}
		}
	} else {
		// Make sure the equals func is sane
		eqFunc = eqMethod.Func
		expected := reflect.FuncOf(
			[]reflect.Type{reflect.TypeOf(et), reflect.TypeOf(et)},
			[]reflect.Type{reflect.TypeOf(true)},
			false,
		)
		eqFunc = eqMethod.Func
		if eqFunc.Type() != expected {
			eqFunc = reflect.ValueOf(reflect.DeepEqual)
		}
	}
	// We can't go any further
	if !eqFunc.IsValid() || eqFunc.Type().Kind() != reflect.Func {
		return false
	}
	found = false
	for i := 0; i < l; i++ {
		v1 := s1v.Index(i)
		for j := 0; j < l; j++ {
			v2 := s2v.Index(j)
			var ret []reflect.Value
			if noPtr {
				ret = eqFunc.Call([]reflect.Value{v1, v2})
			} else {
				ret = eqFunc.Call([]reflect.Value{v1.Addr(), v2.Addr()})
			}
			// Dunno what happened
			if ret == nil || !ret[0].IsValid() {
				return false
			}
			b := ret[0]
			if b.Type().Kind() != reflect.Bool {
				return false
			}
			if b.Bool() {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
