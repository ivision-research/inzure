package inzure

import (
	"encoding/json"
	"os"
	"strings"
)

// SubscriptionFromFile will load a subscription from a file. This handles both
// encrypted and unencrypted files.
func SubscriptionFromFile(fname string) (sub *Subscription, err error) {
	return SubscriptionFromFilePassword(fname, nil)
}

// SubscriptionFromFilePassword will load a Subscription from the given
// encrypted JSON (must have the .enc extension)
func SubscriptionFromFilePassword(fname string, pw []byte) (sub *Subscription, err error) {
	var f *os.File
	f, err = os.Open(fname)
	if err != nil {
		return
	}
	if strings.HasSuffix(fname, EncryptedFileExtension) {
		sub, err = SubscriptionFromEncryptedJSON(pw, f)
	} else {
		sub = new(Subscription)
		err = json.NewDecoder(f).Decode(sub)
	}
	return
}
