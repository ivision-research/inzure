package inzure

import "testing"

func TestGeneralQSContainsAllChild(t *testing.T) {
	var gen QueryString
	err := gen.Parse("/StorageAccount")
	if err != nil {
		t.Fatalf("Failed to parse generic query string: %v", err)
	}
	children := []string{
		"/StorageAccount/rg",
		"/StorageAccount/rg/name",
		"/StorageAccount/*/name",
		"/StorageAccount/*/*",
		"/StorageAccount[.IsClassic == false]/rg/name",
	}

	for _, c := range children {
		var cqs QueryString
		err = cqs.Parse(c)
		if err != nil {
			t.Fatalf("Failed to parse child qs %s: %v", c, err)
		}
		if !gen.ContainsIQS(&cqs) {
			t.Fatalf("Generic didn't contain %s but should have", c)
		}
	}
}
