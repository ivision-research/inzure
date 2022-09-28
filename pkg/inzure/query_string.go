package inzure

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type QueryString struct {
	Raw           string
	Sel           QSSelector
	ResourceGroup string
	Name          string
	Subresource   *QueryString

	finalType reflect.Type
}

func (qs *QueryString) String() string {
	if qs.Raw != "" {
		return qs.Raw
	}
	s := qs.Sel.String()
	if qs.ResourceGroup == "" {
		return s
	}
	s += "/" + qs.ResourceGroup
	if qs.Name == "" {
		return s
	}
	s += "/" + qs.Name
	if qs.Subresource == nil {
		return s
	}
	s += qs.Subresource.String()
	return s
}

// Parse takes an IQS and fills the given QueryString struct with the data
// it represents.
func (qs *QueryString) Parse(s string) error {
	l := newLexer(s)
	if yyParse(l) != 0 {
		return l.err
	}
	*qs = l.result
	qs.Raw = s
	return nil
}

func (p *QueryString) ContainsIQS(oqs *QueryString) bool {
	if oqs == nil {
		return false
	}

	if !p.Sel.Contains(&oqs.Sel) {
		return false
	}

	if p.ResourceGroup == "" {
		return true
	}

	if p.ResourceGroup != "*" && oqs.ResourceGroup != p.ResourceGroup {
		return false
	}

	if p.Name == "" {
		return true
	}

	if p.Name != "*" && oqs.Name != p.Name {
		return false
	}

	if p.Subresource == nil {
		return oqs.Subresource == nil
	}

	if oqs.Subresource == nil {
		return false
	}

	return p.Subresource.ContainsIQS(oqs.Subresource)
}

// ContainsString checks if a query string is a superset of, or equal to, a
// given query string. Without context this can be difficult, so this function
// could potentially return a false negative.
func (p *QueryString) ContainsString(s string) bool {
	if p.Raw == s {
		return true
	}
	var oqs QueryString
	err := oqs.Parse(s)

	if err != nil {
		return false
	}

	return p.ContainsIQS(&oqs)
}

// Validate ensures that the query string is actually valid.
func (qs *QueryString) Validate() error {
	if qs.Sel.Resource == "" {
		return errors.New("query string doesn't specify a resource")
	}
	ty, canFind := qs.GetReturnType()
	if !canFind {
		return errors.New("query string contains invalid selector")
	}
	if qs.Sel.Condition == nil {
		return nil
	}
	return validCondition(getBaseType(ty), qs.Sel.Condition)
}

func validCondition(ty reflect.Type, cond *QSCondition) error {
	if cmp, is := cond.Cmp.(*QSComparer); is {
		checkTy := ty
		field := &cmp.Fields
		for field != nil {
			if field.IsMethod {
				if !typeHasMethod(checkTy, field.Name, false) {
					return fmt.Errorf("type %s does not have method %s", checkTy.Name(), field.Name)
				}
				// No selectors allowed after a method for now
				if field.Next != nil {
					return errors.New("selectors not allowed after method call")
				}
				break
			}
			sf, has := checkTy.FieldByName(field.Name)
			if !has {
				return fmt.Errorf("type %s does not have field %s", checkTy.Name(), field.Name)
			}

			if field.Next == nil {
				break
			}

			field = field.Next
			if field.IsArray {
				if sf.Type.Kind() != reflect.Slice {
					return fmt.Errorf("type %s field %s is not a slice", checkTy.Name(), field.Name)
				}
			}
			checkTy = getBaseType(sf.Type)
		}
	}
	if cond.And != nil {
		if err := validCondition(ty, cond.And); err != nil {
			return err
		}
	}
	if cond.Or != nil {
		return validCondition(ty, cond.Or)
	}
	return nil
}

// GetReturnType returns the reflect.Type that should be returned by this
// query string when used with a Subscription.
func (qs *QueryString) GetReturnType() (reflect.Type, bool) {
	// We cache it so check that first
	if qs.finalType != nil {
		return qs.finalType, true
	}
	if qs.Sel.Resource == "" {
		return reflect.TypeOf(nil), false
	}
	var rg ResourceGroup
	t := reflect.TypeOf(rg)
	ft, has := t.FieldByName(qs.Sel.Resource)
	if !has {
		return reflect.TypeOf(nil), false
	}
	ty := ft.Type
	if qs.Subresource == nil {
		// No name or * name == exact type in the struct
		if qs.Name == "" || qs.Name == "*" {
			qs.finalType = ty
			return ty, true
		}
		// Otherwise we need to dereference and get the inner type
		for ty.Kind() == reflect.Slice {
			ty = ty.Elem()
		}
		qs.finalType = ty
		return ty, true
	}
	// Given a subresource there is more to do. We can't just recurse though
	// since they aren't base types.
	for ty.Kind() == reflect.Ptr || ty.Kind() == reflect.Slice {
		ty = ty.Elem()
	}
	ft, has = ty.FieldByName(qs.Subresource.Sel.Resource)
	if !has {
		return reflect.TypeOf(nil), false
	}
	// No name == exact type in the struct
	if qs.Subresource.Name == "" {
		qs.finalType = ft.Type
		return ft.Type, true
	}
	// Has a name means the type in the struct is assumed to be a slice/array
	// TODO: Should do a check for that here
	ty = ft.Type
	for ty.Kind() == reflect.Ptr || ty.Kind() == reflect.Slice {
		ty = ty.Elem()
	}
	qs.finalType = ty
	return ty, true

}

func GetQSFillableValue(qs *QueryString) reflect.Value {
	t, ok := qs.GetReturnType()
	if !ok {
		return reflect.ValueOf(nil)
	}
	return reflect.New(t)
}

// GetQSFillableValueForString returns a reflect.Value that can be filled by
// the *QueryString methods on a Subscription. You can either give this a full
// QueryString or the name of a field in a ResourceGroup.
func GetQSFillableValueForString(qs string) reflect.Value {
	if !strings.HasPrefix(qs, "/") {
		var rg ResourceGroup
		ty := reflect.TypeOf(rg)
		ft, has := ty.FieldByName(qs)
		if !has {
			return reflect.ValueOf(nil)
		}
		return reflect.New(ft.Type)
	}
	t, has := getTypeForQueryString(qs)
	if !has {
		return reflect.ValueOf(nil)
	}
	return reflect.New(t)
}

func getTypeForQueryString(s string) (reflect.Type, bool) {
	var q QueryString
	err := q.Parse(s)
	if err != nil {
		return reflect.TypeOf(nil), false
	}
	return q.GetReturnType()
}

func (qs *QueryString) BaseString() string {
	if qs.Subresource == nil {
		return qs.Raw
	}
	sp := strings.Split(qs.Raw, "/")
	if sp == nil || len(sp) == 0 {
		return qs.Raw
	}
	l := len(sp)
	if qs.Subresource.Name != "" {
		return strings.Join(sp[:l-2], "/")
	}
	return strings.Join(sp[:l-1], "/")
}

// ToQueryString accepts an interface struct and attempts to turn it in to a
// valid query string. This is not always successful and it isn't always easy
// to detect when it is unsuccessful and return an error, so YMMV.
func ToQueryString(i interface{}) (string, error) {
	var rg ResourceGroup
	v := derefPtr(reflect.ValueOf(i))
	if v.Kind() != reflect.Struct {
		return "", errors.New("cannot convert nonstruct to query string")
	}
	name := pluralize(
		nopackageStructName(v.Type()),
	)
	rt := reflect.TypeOf(rg)
	_, has := rt.FieldByName(name)
	if !has {
		return "", fmt.Errorf("resource groups don't have the field %s", name)
	}
	metaV := v.FieldByName("Meta")
	if !metaV.IsValid() {
		return "", fmt.Errorf("passed type had no Meta field: %s", v.Type())
	}
	meta, ok := metaV.Interface().(ResourceID)
	if !ok {
		return "", fmt.Errorf("Meta field wasn't a ResourceID on %s", v.Type())
	}
	return fmt.Sprintf("/%s/%s/%s", name, meta.ResourceGroupName, meta.Name), nil
}

// pluralize for now is really simple because we don't have any valid names
// that don't just add an s on the end for pluralizing.
func pluralize(s string) string {
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}
