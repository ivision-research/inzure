package inzure

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestSubIDsFromEnv(t *testing.T) {
	tf, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("failed to make temp file: %v", err)
	}

	exp := `11111111-2222-3333-4444-555555555555=BlahBlah Blah`
	contents := fmt.Sprintf(`%s
#66666666-7777-8888-9999-aaaaaaaaaaaa=Such Subscription 
#bbbbbbbb-cccc-dddd-eeee-ffffffffffff=Sub_scription
`,
		exp,
	)

	os.Setenv(EnvSubscriptionFile, tf.Name())
	_, err = tf.Write([]byte(contents))
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	defer tf.Close()
	subIDs, err := SubscriptionIDsFromEnv()
	if err != nil {
		t.Fatalf("failed to get subscription IDs: %v", err)
	}
	if len(subIDs) > 1 {
		t.Fatalf("expected only 1 sub but got %d", len(subIDs))
	}
	expected := SubIDFromString(exp)
	if subIDs[0].ID != expected.ID {
		t.Fatalf(
			"sub had bad id expected %s got %s", expected.ID, subIDs[0].ID,
		)
	}
	if subIDs[0].Alias != expected.Alias {
		t.Fatalf(
			"sub had bad alias expected %s got %s",
			expected.Alias, subIDs[0].Alias,
		)
	}
}
