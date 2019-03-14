package inzure

import (
	"errors"
	"fmt"
	"reflect"
)

// SubscriptionDiff holds the diff of two subscriptions as slices of inzure
// query strings.
type SubscriptionDiff struct {
	Added    []string
	Removed  []string
	Modified []string
}

var ErrDifferentSubscriptions = errors.New("diff subscriptions are not the same")

var diffFields []string

// Diff will diff two subscriptions
func (s *Subscription) Diff(o *Subscription) (*SubscriptionDiff, error) {
	diff := &SubscriptionDiff{
		Added:    make([]string, 0),
		Removed:  make([]string, 0),
		Modified: make([]string, 0),
	}
	if diffFields == nil || len(diffFields) == 0 {
		diffFields = getDiffFields()
		if len(diffFields) == 0 {
			return nil, errors.New("failed to find diff fields")
		}
	}
	if s.ID != o.ID {
		return nil, ErrDifferentSubscriptions
	}
	for oRgName, oRg := range o.ResourceGroups {
		if sRg, has := s.ResourceGroups[oRgName]; has {
			if err := diffCompareResourceGroups(diff, sRg, oRg); err != nil {
				return nil, err
			}
		} else {
			if err := diffAddAllFields(&diff.Added, oRg); err != nil {
				return nil, err
			}
		}
	}
	for sRgName, sRg := range s.ResourceGroups {
		if _, has := o.ResourceGroups[sRgName]; !has {
			if err := diffAddAllFields(&diff.Removed, sRg); err != nil {
				return nil, err
			}
		}
	}
	return diff, nil
}

func diffCompareResourceGroups(diff *SubscriptionDiff, newRg *ResourceGroup, oldRg *ResourceGroup) error {
	nV := reflect.ValueOf(*newRg)
	oV := reflect.ValueOf(*oldRg)
	for _, f := range diffFields {
		nField := nV.FieldByName(f)
		if !nField.IsValid() {
			return errors.New("diff given an invalid Subscription")
		}
		oField := oV.FieldByName(f)
		if !oField.IsValid() {
			return errors.New("diff given an invalid Subscription")
		}
		nL := nField.Len()
		oL := oField.Len()
		if nL == 0 && oL == 0 {
			continue
		} else if nL == 0 && oL > 0 {
			// Everything was removed
			diffAddAllValues(&diff.Removed, oField, oL)
		} else if nL > 0 && oL == 0 {
			// Everything was added
			diffAddAllValues(&diff.Added, nField, nL)
		} else {
			// Could be added/modded/removed
			diffCompareSlices(diff, oField, oL, nField, nL)
		}
	}
	return nil
}

func getMeta(v reflect.Value) (ResourceID, error) {
	var rid ResourceID
	if !v.IsValid() {
		return rid, fmt.Errorf("cannot get Meta from %v: not valid", v)
	}
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	meta := v.FieldByName("Meta")
	if !meta.IsValid() {
		return rid, fmt.Errorf("cannot get Meta from %v: field doesn't exist", v)
	}
	if val, ok := meta.Interface().(ResourceID); !ok {
		return rid, fmt.Errorf("cannot get Meta from %v: field Meta isn't a ResourceID", v)
	} else {
		rid = val
	}
	return rid, nil
}

func diffCompareSlices(
	diff *SubscriptionDiff,
	oldS reflect.Value,
	oldSLen int,
	newS reflect.Value,
	newSLen int,
) error {
	// So we're going to try to do this simply before thinking about doing it
	// efficiently.
	// 1. Grab an IQS for every element in each slice.
	// 2. Create a map that uses those IQS as keys and the reflect.Value as values
	//    for each.
	// 3. Use map lookups to get resources that should be the same or to find new
	//    ones.
	oldMap := make(map[string]reflect.Value)
	newMap := make(map[string]reflect.Value)
	for i := 0; i < newSLen; i++ {
		val := newS.Index(i)
		qs, err := ToQueryString(val.Interface())
		if err != nil {
			return err
		}
		newMap[qs] = val
	}
	for i := 0; i < oldSLen; i++ {
		val := oldS.Index(i)
		qs, err := ToQueryString(val.Interface())
		if err != nil {
			return err
		}
		oldMap[qs] = val
	}
	for nQS, nVal := range newMap {
		if oVal, has := oldMap[nQS]; has {
			if !reflect.DeepEqual(nVal.Interface(), oVal.Interface()) {
				diff.Modified = append(diff.Modified, nQS)
			}
		} else {
			diff.Added = append(diff.Added, nQS)
		}
	}
	for oQS, _ := range oldMap {
		if _, has := newMap[oQS]; !has {
			diff.Removed = append(diff.Removed, oQS)
		}
	}
	return nil
}

func diffAddAllValues(to *[]string, valSlice reflect.Value, l int) error {
	for i := 0; i < l; i++ {
		v := valSlice.Index(i).Interface()
		qs, err := ToQueryString(v)
		if err != nil {
			return err
		}
		*to = append(*to, qs)
	}
	return nil
}

func diffAddAllFields(to *[]string, rg *ResourceGroup) error {
	v := reflect.ValueOf(*rg)
	for _, f := range diffFields {
		val := v.FieldByName(f)
		l := val.Len()
		if l == 0 {
			continue
		}
		for i := 0; i < l; i++ {
			qs, err := ToQueryString(val.Index(i).Interface())
			if err != nil {
				return err
			}
			*to = append(*to, qs)
		}
	}
	return nil
}

func getDiffFields() []string {
	t := reflect.TypeOf(ResourceGroup{})
	nf := t.NumField()
	ret := make([]string, 0, nf)
	for i := 0; i < nf; i++ {
		f := t.Field(i)
		if f.Type.Kind() == reflect.Slice {
			tag := f.Tag
			if tag.Get("diff") == "ignore" {
				continue
			}
			ret = append(ret, f.Name)
		}
	}
	return ret
}
