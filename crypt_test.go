package inzure

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	msg := "Hello this is an encrypted message that should be multiple blocks"
	key := [32]byte{
		0xa9, 0x4b, 0x18, 0xf4, 0xad, 0xed, 0xea, 0x89,
		0xc0, 0xa5, 0x4b, 0xf3, 0x03, 0x5c, 0xff, 0x16,
		0xf8, 0xef, 0x2d, 0xf9, 0xce, 0x7b, 0x88, 0x6d,
		0x4e, 0x4e, 0x62, 0xd1, 0xdf, 0x98, 0xa1, 0x6d,
	}

	var buf bytes.Buffer
	err := encrypt(key[:], []byte(msg), &buf)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := decrypt(key[:], &buf)
	if err != nil {
		t.Fatal(err)
	}
	if msg != string(dec) {
		t.Fatalf("%s != %s", string(dec), msg)
	}
}

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
