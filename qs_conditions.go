package inzure

import "reflect"

type QSCondition struct {
	Raw string
	Cmp IQSComparer
	And *QSCondition
	Or  *QSCondition
}

func (qsc *QSCondition) Compare(v reflect.Value) (bool, error) {
	ok, err := qsc.Cmp.Compare(v)
	if err != nil {
		return false, err
	}
	if !ok {
		if qsc.Or != nil {
			return qsc.Or.Compare(v)
		}
		return false, nil
	} else if qsc.And != nil {
		return qsc.And.Compare(v)
	}
	return true, nil
}

func (qsc *QSCondition) PushAnd(ic *QSCondition) {
	if qsc.And == nil {
		qsc.And = ic
	} else {
		last := qsc.And
		for ; last.And != nil; last = last.And {
			// empty
		}
		last.And = ic
	}
	if qsc.Or != nil {
		qsc.Or.PushAnd(ic)
	}
}

func (qsc *QSCondition) PushOr(ic *QSCondition) {
	if qsc.Or == nil {
		qsc.Or = ic
	} else {
		last := qsc.Or
		for ; last.Or != nil; last = last.Or {
			// empty
		}
		last.Or = ic
	}
	if qsc.And != nil {
		qsc.And.PushOr(ic)
	}
}

func (qsc *QSCondition) FilterValue(v reflect.Value) (reflect.Value, error) {
	if qsc.Cmp == nil {
		return v, nil
	}
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		return qsc.filterSliceOfValues(v)
	}

	return qsc.filterSingleValue(v)
}

func (qsc *QSCondition) filterSingleValue(v reflect.Value) (reflect.Value, error) {
	ok, err := qsc.Cmp.Compare(v)
	if err != nil {
		return qsNilVal(), err
	}
	if !ok {
		if qsc.Or != nil {
			return qsc.Or.FilterValue(v)
		}
		return qsNilVal(), nil
	} else if qsc.And != nil {
		return qsc.And.FilterValue(v)
	}
	return v, nil
}

func (qsc *QSCondition) filterSliceOfValues(v reflect.Value) (reflect.Value, error) {
	l := v.Len()
	if l == 0 {
		return v, nil
	}
	// Make a new slice to hold results that pass the filter and then
	// append them as we go on
	newV := reflect.MakeSlice(v.Type(), 0, l)
	for i := 0; i < l; i++ {
		val, err := qsc.filterSingleValue(v.Index(i))
		if err != nil {
			return qsNilVal(), err
		} else if val.IsValid() {
			newV = reflect.Append(newV, val)
		}
	}
	return newV, nil
}

func (qsc *QSCondition) Equals(o *QSCondition) bool {
	if o == nil {
		return false
	}
	// in practice this isn't perfectly accurate.. you could have
	//
	// .Foo == "Bar" && .Bar == "Baz"
	//
	// 				and
	//
	// .Bar == "Baz" && .Foo == "Bar"
	//
	// which are equivalent but unequal in this comparison. Oh well.
	return qsc.Raw == o.Raw
}

func (qsc *QSCondition) String() string { return qsc.Raw }
