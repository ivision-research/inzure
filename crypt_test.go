package inzure

import (
	"bytes"
	"testing"
)

func TestEncryptSubscription(t *testing.T) {
	pw := []byte("foobar")
	var buf bytes.Buffer
	sub := NewSubscription("foobarid")
	err := EncryptSubscriptionAsJSON(&sub, pw, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("didn't write anyhing")
	}
}

func TestDecryptSubscription(t *testing.T) {
	pw := []byte("foobar")
	var buf bytes.Buffer
	sub := NewSubscription("foobarid")
	err := EncryptSubscriptionAsJSON(&sub, pw, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("didn't write anyhing")
	}
	sub2, err := SubscriptionFromEncryptedJSON(pw, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if sub2.ID != sub.ID {
		t.Fatal("decryption failed")
	}
}

func TestHMACCatchesBitFlip(t *testing.T) {
	pw := []byte("foobar")
	var buf bytes.Buffer
	sub := NewSubscription("foobarid")
	err := EncryptSubscriptionAsJSON(&sub, pw, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("didn't write anyhing")
	}
	b := buf.Bytes()
	// Flip a bit in the first byte
	loc := 8 + 32 + 16 + 1
	if b[loc]&1 == 0 {
		b[loc] |= 1
	} else {
		b[loc] &^= 1
	}
	r := bytes.NewReader(b)
	_, err = SubscriptionFromEncryptedJSON(pw, r)
	if err == nil {
		t.Fatal("should have errored due to the bit flip")
	}
	if err.Error() != "bad mac" {
		t.Fatalf("wrong error: %v", err)
	}
}
