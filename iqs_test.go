package inzure

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func testSimpleBaseQS(t *testing.T, qs string) {
	val, err := testSub.ReflectFromQueryString(qs)
	if err != nil {
		t.Fatal(err)

		for val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		l := val.Len()
		fieldName := strings.Replace(qs, "/", "", 1)
		for i := 0; i < l; i++ {
			search := val.Index(i).Interface()
			found := false
			for _, rg := range testSub.ResourceGroups {
				rgAll := reflect.ValueOf(*rg).FieldByName(fieldName)
				rgL := rgAll.Len()
				for j := 0; j < rgL; j++ {
					test := rgAll.Index(j).Interface()
					if reflect.DeepEqual(search, test) {
						found = true
						break
					}
				}
			}
			if !found {
				t.Fatalf("QS returned %v but it couldn't be found", search)
			}
		}
	}
}

func TestSimpleBaseQS(t *testing.T) {
	ty := reflect.TypeOf(ResourceGroup{})
	nf := ty.NumField()
	fields := make([]string, 0, nf)
	for i := 0; i < nf; i++ {
		f := ty.Field(i)
		if f.Type.Kind() == reflect.Slice {
			fields = append(fields, f.Name)
		}
	}

	for _, f := range fields {
		qs := fmt.Sprintf("/%s", f)
		testSimpleBaseQS(t, qs)
	}
}

func TestNameLikeQS(t *testing.T) {
	into := make([]*SQLServer, 0)
	s := "/SQLServers[.Meta.Name ~ \"^inzure-.*\"]"
	err := testSub.FromQueryString(s, &into)
	if err != nil {
		t.Fatal(err)
	}
	for _, serv := range into {
		if !strings.HasPrefix(serv.Meta.Name, "inzure-") {
			t.Fatalf("%s should not have been returned by qs %s", serv.Meta.Name, s)
		}
	}
}

func testGetQS(t *testing.T, qs string) (reflect.Value, error) {
	/*
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("pancied when trying to get qs %s", qs)
			}
		}()
	*/
	return testSub.ReflectFromQueryString(qs)
}

/*
	qs := fmt.Sprintf("/WebApps[Language.Language == %d]", LanguagePython)
	v, err := testGetQS(t, qs)
	if err != nil {
		t.Fatalf("%s failed but shouldn't have: %v", qs, err)
	}
	fmt.Println(v)
*/

func iqsTestSlice(v reflect.Value, f func(interface{})) {
	v = derefPtr(v)
	s := v.Len()
	for i := 0; i < s; i++ {
		e := v.Index(i)
		f(e.Interface())
	}
}

func TestSubresourcesQS(t *testing.T) {
	qs := "/StorageAccounts/*/*/Containers[.Name !~ \"^inzure\"]"
	v, err := testGetQS(t, qs)
	if err != nil {
		t.Fatalf("%s failed but shouldn't have: %v", qs, err)
	}
	chkFunc := func(v interface{}) {
		switch c := v.(type) {
		case Container:
			if strings.HasPrefix(c.Name, "inzure") {
				t.Fatalf("Container %s shouldn't have been selected", c.Name)
			}
		default:
			t.Fatalf("got the wrong type %T", c)
		}
	}
	iqsTestSlice(v, chkFunc)
}

func TestBadQSFail(t *testing.T) {
	badQSs := []string{
		"",
		"SQLSevers",                              // no preceding /
		"/VirtualMachines[.Meta.Name == \"bob\"", // no closing ]
		"/RedisServers[.SuchField == BoolTrue]",  // field doesn't exist
		"/PostgresServer",                        // typo, no trailing s
		"/WeBApPs",                               // Case sensitivity
		"/WebApps[.Meta.ResourceGroupName == 'foo']", // ' isn't valid
		"/WebApps[.HTTPSOnly = BoolFalse]",           // = instead of ==
		"/WebApps[.HTTPSOnly == bOOlFaLse]",          // case sensitivity
		"/WebApps[HTTPSOnly == BoolFalse]",           // bad field selector (no .)
		"/WebApps[.NonExistentMethod() == 1]",
	}
	for _, qs := range badQSs {
		_, err := testSub.ReflectFromQueryString(qs)
		if err == nil {
			t.Fatalf("qs %s should have failed but didn't", qs)
		}
	}
}

func TestQSMethodCall(t *testing.T) {
	qs := "/NetworkSecurityGroups[.AllowsIPToPortString(\"12.34.56.78\", \"22\")[0] == BoolTrue]"
	into := make([]*NetworkSecurityGroup, 0, 5)
	err := testSub.FromQueryString(qs, &into)
	if err != nil {
		t.Fatalf("Failed to execute query string %s: %v", qs, err)
	}
	if len(into) == 0 {
		t.Fatalf("Should have had a returned value for %s", qs)
	} else if len(into) > 1 {
		t.Fatalf("Should have had only one returned value for %s", qs)
	}
	if into[0].Meta.Name != "inzure-nsg-3" {
		t.Fatalf("Got the wrong NSG: %s", into[0].Meta.Name)
	}

	qs = "/CosmosDBs[.Firewall.AllowsIPString(\"0.0.0.0\")]"
}

func TestQSValidOdd(t *testing.T) {
	okQSs := []string{
		"/VirtualMachines/this.is_ok.rg",  // Allow dots and underscores in rg
		"/VirtualMachines/rg/allows_this", // Allow underscores in name
	}
	for _, qs := range okQSs {
		var p QueryString
		err := p.Parse(qs)
		if err != nil {
			t.Fatalf("qs %s failed but should have passed:\n%v", qs, err)
		}
	}
}
