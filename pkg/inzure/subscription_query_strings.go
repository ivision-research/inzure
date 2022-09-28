package inzure

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// FromQueryString loads the item[s] identified by the query string into the
// passed interface.
//
// `into` needs to be a pointer to the expected type. For instance, if
// '/SQLServers` is given, `into` needs to be `*[]*SQLServer`. Note that you
// are given the actual pointers so modifying what you get modifies the
// Subscription as well.
//
// query strings are really just Go struct field selectors with a little more
// functionality. Everything starts on a ResourceGroup struct, so for VMs you'd
// start with `/VirtualMachines` for example. You can add conditions onto a
// query string type by putting it in brackets `[]`. For example, to get all
// virtual machines that might allow password auth, you'd use
// `/VirtualMachines[.DisablePasswordAuth != BoolTrue]`. You can also specify
// this on sub resources. To get all Containers in the subscription with public
// read access you could use `/StorageAccounts/*/*/Containers[.Access == 3]`
//
// This makes dealing with inzure data significanly easier, and the
// accompanying `inzure search` command can be used to access this interface.
func (s *Subscription) FromQueryString(qs string, into interface{}) error {
	v := reflect.ValueOf(into)
	return s.ValueFromQueryString(qs, v)
}

func (s *Subscription) ReflectFromQueryString(qs string) (reflect.Value, error) {
	var p QueryString
	if err := p.Parse(qs); err != nil {
		return reflect.ValueOf(nil), err
	}
	return s.ReflectFromParsedQueryString(&p)
}

func (s *Subscription) ReflectFromParsedQueryString(p *QueryString) (reflect.Value, error) {
	v := GetQSFillableValue(p)
	if !v.IsValid() {
		return reflect.ValueOf(nil), fmt.Errorf("bad query string %s", p.Raw)
	}
	err := s.valueFromQueryString(p, v)
	return v, err
}

// ValueFromQueryString is the same as FromQueryString
// except it accepts a reflect.Value
func (s *Subscription) ValueFromQueryString(qs string, v reflect.Value) error {
	if qs == "*" {
		return errors.New("\"*\" is not allowed for this method")
	}
	if v.Kind() != reflect.Ptr {
		return errors.New("resources can only be loaded into a pointer type")
	}
	var p QueryString
	err := p.Parse(qs)
	if err != nil {
		return err
	}
	return s.valueFromQueryString(&p, v)
}

func (s *Subscription) valueFromQueryString(qs *QueryString, v reflect.Value) error {
	if err := qs.Validate(); err != nil {
		return err
	}
	if qs.Subresource != nil {
		return s.subresourceValueFromQueryString(qs, v)
	}
	return s.baseValueFromQueryString(qs, v)
}

func (s *Subscription) subresourceValueFromQueryString(qs *QueryString, v reflect.Value) error {
	// In order to get the value we need to get the parent values. There are
	// two interesting paths:
	//
	// 1) The QS isn't specifying a single parent.
	// 2) The QS is specifying a single parent.

	// Multiple parents are name == * only. name == "" is not proper syntax

	// Both will start with a *[]*Parent. In the single case we'll need to do
	// some operations to end up with a *Parent
	pVs := GetQSFillableValueForString(qs.Sel.Resource)
	if !pVs.IsValid() {
		return fmt.Errorf(
			"failed to get a valid parent type for string %s",
			qs.String(),
		)
	}
	if qs.Name == "*" {
		if err := s.baseValueFromQueryString(qs, pVs); err != nil {
			return err
		}
		return s.subresourceValueFromMultipleParents(pVs, qs, v)
	} else {
		// get a *Parent to use instead
		pV := derefPtr(pVs)
		if pV.Kind() != reflect.Slice {
			return fmt.Errorf(
				"failed to get a single underlying parent for %s", qs.Raw,
			)
		}
		pV = reflect.New(pV.Type().Elem())
		if err := s.baseValueFromQueryString(qs, pV); err != nil {
			return err
		}
		return s.subresourceValueFromSingleParent(pV, qs, v)
	}
}

func (s *Subscription) subresourceValueFromMultipleParents(
	parents reflect.Value, qs *QueryString, v reflect.Value,
) error {
	parents = derefPtr(parents)
	sLen := parents.Len()
	if sLen == 0 {
		return nil
	}
	rV := reflect.Indirect(v)
	// In the end, we need to fill up with the subresource type, so time to
	// start doing that.
	ty, _ := qs.GetReturnType()
	// Loop through all of the parents and use the single parent methods to
	// to fill up a slice. Then we'll just keep appending that slice to
	// a final value.
	tmp := reflect.New(ty)
	for i := 0; i < sLen; i++ {
		reflect.Indirect(tmp).Set(reflect.Indirect(tmp).Slice(0, 0))
		parent := parents.Index(i)
		err := s.subresourceValueFromSingleParent(parent, qs, tmp)
		if err != nil {
			return err
		}
		// TODO Why do I have to deref here? What was happening in that
		// method..? I think this is just leftover from something else and
		// isn't actually necessary.
		tmp = derefPtr(tmp)
		tLen := tmp.Len()
		if tLen == 0 {
			continue
		}
		rV.Set(reflect.AppendSlice(rV, tmp))
	}
	return nil
}

func (s *Subscription) subresourceValueFromSingleParent(
	parent reflect.Value, qs *QueryString, v reflect.Value,
) error {
	values, err := filterSubresources(parent, qs)
	if err != nil {
		return err
	}
	// I'm not sure if this case is possible? Let's leave the check anyway
	// since it is simple to deal with.
	if values.Kind() != reflect.Slice {
		return setSingleQSValue(v, values, qs.Raw)
	} else if qs.ResourceGroup == "*" ||
		qs.Name == "*" ||
		qs.Subresource.Name == "" {

		ind := reflect.Indirect(v)
		if err := checkSliceSubresource(
			qs.Sel.Resource, qs.Subresource.Sel.Resource, ind,
		); err != nil {
			return err
		}
		l := values.Len()
		for i := 0; i < l; i++ {
			e := values.Index(i)
			ind.Set(reflect.Append(ind, e))
		}
		return nil
	}
	// Otherwise find our item with the right name
	l := values.Len()
	for i := 0; i < l; i++ {
		chk := derefPtr(values.Index(i))
		name, err := getResourceName(chk)
		if err != nil {
			return err
		}
		if strings.ToLower(qs.Subresource.Name) == name {
			return setSingleQSValue(v, values.Index(i), qs.Raw)
		}
	}
	return fmt.Errorf("string %s does not match any resources in %s", qs.String(), s.ID)

}

func getResourceName(v reflect.Value) (string, error) {
	var name string
	meta := v.FieldByName("Meta")
	if !meta.IsValid() {
		meta = v.FieldByName("Name")
		if !meta.IsValid() {
			return "", fmt.Errorf("type %s doesn't have a Meta or Name field", v)
		}
		name = meta.String()
	} else {
		name = meta.FieldByName("Name").String()
	}
	return strings.ToLower(name), nil
}

func (s *Subscription) baseValueFromQueryString(qs *QueryString, v reflect.Value) error {
	if qs.ResourceGroup == "" {
		return s.fillWithAllOfType(qs, qs.Sel.Resource, v)
	}
	var rg *ResourceGroup
	// * means all resource groups, so we don't need to grab a specific one here
	if qs.ResourceGroup != "*" {
		var ok bool
		rg, ok = s.ResourceGroups[qs.ResourceGroup]
		if !ok {
			return fmt.Errorf(
				"resource group %s doesn't exist in subscription %s",
				qs.ResourceGroup, s.ID,
			)
		}
	}
	if qs.Name == "" || qs.Name == "*" {
		if rg == nil {
			return s.fillWithAllOfType(qs, qs.Sel.Resource, v)
		} else {
			return s.fillWithAllOfTypeInGroup(qs, qs.Sel.Resource, rg, v)
		}
	}
	var values []reflect.Value
	var err error
	if rg != nil {
		values, err = getAllOfType(qs, qs.Sel.Resource, rg)
		if err != nil {
			return err
		}
	} else {
		// TODO: Magic
		values = make([]reflect.Value, 0, 10)
		for _, rg := range s.ResourceGroups {
			nv, err := getAllOfType(qs, qs.Sel.Resource, rg)
			if err != nil {
				return err
			}
			values = append(values, nv...)
		}
	}
	// Otherwise we just continue on and get our single value
	value, err := getSingle(
		s.ID, qs.BaseString(), qs.Name, values,
	)
	if err != nil {
		return err
	}
	return setSingleQSValue(v, value, qs.BaseString())

}

func getSingle(sub, qs, name string, all []reflect.Value) (reflect.Value, error) {
	var val reflect.Value
	for _, e := range all {
		val = e
		for val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		eName, err := getResourceName(val)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		if eName == strings.ToLower(name) {
			return e, nil
		}
	}
	return reflect.ValueOf(nil), fmt.Errorf("string %s does not match any resources in %s", qs, sub)
}

func checkSlice(ts string, v reflect.Value) error {
	if v.Kind() != reflect.Slice {
		//return fmt.Errorf("type to check wasn't a slice: %s", v.Kind())
		panic(fmt.Errorf("type to check wasn't a slice: %s", v.Kind()))
	}
	return nil
}

func checkSliceSubresource(pts string, ts string, v reflect.Value) error {
	// TODO This was panicing as a debug helper
	if v.Kind() != reflect.Slice {
		//return fmt.Errorf("type to check wasn't a slice: %s", v.Kind())
		panic(fmt.Errorf("type to check wasn't a slice: %s", v.Kind()))
	}
	// TODO I need to actually write the check here...
	return nil

}

func (s *Subscription) fillWithAllOfTypeInGroup(
	p *QueryString, ts string, group *ResourceGroup,
	v reflect.Value,
) error {
	ind := reflect.Indirect(v)
	if err := checkSlice(ts, ind); err != nil {
		return err
	}
	values, err := getAllOfType(p, ts, group)
	if err != nil {
		return err
	}
	for _, e := range values {
		ind.Set(reflect.Append(ind, e))
	}
	return nil
}

func setSingleQSValue(toSet reflect.Value, from reflect.Value, qs string) error {
	ind := reflect.Indirect(toSet)
	if ind.Kind() != reflect.Ptr {
		for from.Kind() == reflect.Ptr {
			from = from.Elem()
		}
	}
	t := from.Type()
	if ind.Type() != t {
		return fmt.Errorf(
			"bad type for loading %s: %s (expected %s)",
			qs, ind.Type(), t,
		)
	}
	ind.Set(from)
	return nil
}

func filterSubresources(pV reflect.Value, qs *QueryString) (reflect.Value, error) {
	pV = derefPtr(pV)
	val := pV.FieldByName(qs.Subresource.Sel.Resource)
	if qs.Subresource.Sel.Condition != nil {
		return qs.Subresource.Sel.Condition.FilterValue(val)
	}
	return val, nil
}

func (s *Subscription) fillWithAllOfType(
	qs *QueryString, ts string, v reflect.Value,
) error {
	ind := reflect.Indirect(v)
	if err := checkSlice(ts, ind); err != nil {
		return err
	}
	for _, rg := range s.ResourceGroups {
		values, err := getAllOfType(qs, ts, rg)
		if err != nil {
			return err
		}
		for _, e := range values {
			if e.IsValid() {
				ind.Set(reflect.Append(ind, e))
			}
		}
	}
	return nil
}

func typeSliceToValues(qs *QueryString, val reflect.Value) ([]reflect.Value, error) {
	size := val.Len()
	into := make([]reflect.Value, 0, size)
	for i := 0; i < size; i++ {
		e := val.Index(i)
		if qs.Sel.Condition != nil {
			e, err := qs.Sel.Condition.FilterValue(e)
			if err != nil {
				return nil, err
			}
			if e.IsValid() {
				into = append(into, e)
			}
		} else {
			into = append(into, e)
		}
	}
	return into, nil
}

// getAllOfType returns all of the given resources of a given type in the
// resource group.
func getAllOfType(p *QueryString, ts string, rg *ResourceGroup) ([]reflect.Value, error) {
	v := reflect.ValueOf(*rg)
	return typeSliceToValues(p, v.FieldByName(ts))
}
