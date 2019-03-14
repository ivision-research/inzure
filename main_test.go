package inzure

import (
	"fmt"
	"os"
	"testing"
)

var testSub *Subscription

func TestMain(m *testing.M) {
	var err error
	testSub, err = SubscriptionFromFile("mock.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get mock subscription: %v", err)
		os.Exit(-1)
	}
	os.Exit(m.Run())
}
