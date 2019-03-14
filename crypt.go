package inzure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// PBKDF2Rounds the number of rounds to run for PBKDF2. This can also be
	// overriden with the INZURE_PBKDF2_ROUNDS as long as that value is higher
	// than this default.
	PBKDF2Rounds int = 10000
	saltSize     int = 8
	keyLength    int = 32
	// RoundsEnvironmentalVariableName is the name of the environmental variable
	// that can be set to override the default PBKDF2Rounds. If this value is not
	// greater than the default, the default is used.
	RoundsEnvironmentalVariableName = "INZURE_PBKDF2_ROUNDS"
	// KeyEnvironmentalVariableName is the environmental variable to check
	// for a user's chosen encryption/decryption key
	KeyEnvironmentalVariableName = "INZURE_ENCRYPT_PASSWORD"
	// EncryptedFileExtension is the extension that should be put at the end of
	// a file that is encrypted by this package. If you see a file with this ext
	// as the input it is reasonable to assume it is encrypted and the
	// environmental defined by KeyEnvironmentalVariableName is set.
	EncryptedFileExtension = ".enc"
)

// EncryptSubscriptionAsJSON writes the given Subscription as an encrypted JSON
// to the given writer. This uses inzure's encryption format defined as
// follows:
//
// 1. PBKDF2 is used on the given password with inzure.PBKDF2Rounds number of
// rounds and an 8 byte salt from crypto/rand. Note that this salt is merely
// writen to the writer as the first 8 bytes and is not a secret.
//
// 2. AES256 is used in CBC mode to encrypt the output marshaled JSON with the
// IV as the first block.
//
// 3. An HMAC with SHA256 is taken of the entire cipher text (including the IV)
// and written to the writer after the salt.
//
// Note that this method is intended to make people actually use encryption for
// this data and is not intended to be the most secure possible way to encrypt
// this data. If you have a better tool it is recommended that you use it.
//
// If pw is nil this function checks the KeyEnvironmentalVariableName
// environmental variable.
func EncryptSubscriptionAsJSON(sub *Subscription, pw []byte, w io.Writer) error {
	key, salt, err := genKey(pw, nil)
	if err != nil {
		return err
	}
	n, err := w.Write(salt)
	if err != nil {
		return err
	}
	if n != saltSize {
		return errors.New("invalid write count for salt")
	}
	b, err := json.Marshal(sub)
	if err != nil {
		return err
	}
	return encrypt(key, b, w)
}

// SubscriptionFromEncryptedJSON is the counterpart decryption function.
//
// If pw is nil this function checks the KeyEnvironmentalVariableName
// environmental variable.
func SubscriptionFromEncryptedJSON(pw []byte, r io.Reader) (*Subscription, error) {
	if pw == nil || len(pw) == 0 {
		pw = pwFromEnv()
		if pw == nil {
			return nil, fmt.Errorf("no password given and %s not set", KeyEnvironmentalVariableName)
		}
	}
	salt := make([]byte, saltSize)
	n, err := io.ReadFull(r, salt)
	if err != nil {
		return nil, err
	}
	if n != saltSize {
		return nil, fmt.Errorf("failed to read %d bytes from reader for salt", saltSize)
	}
	key, _, err := genKey(pw, salt)
	if err != nil {
		return nil, err
	}
	sub := new(Subscription)
	js, err := decrypt(key, r)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(js, sub)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling decrypted text: %v", err)
	}
	return sub, nil
}

func decrypt(key []byte, r io.Reader) ([]byte, error) {
	h := hmac.New(sha256.New, key)
	mac := make([]byte, h.Size())
	n, err := io.ReadFull(r, mac)
	if err != nil {
		return nil, err
	}
	if n != h.Size() {
		return nil, fmt.Errorf("failed to read %d bytes from reader for mac", h.Size())
	}
	iv := make([]byte, aes.BlockSize)
	n, err = io.ReadFull(r, iv)
	if err != nil {
		return nil, err
	}
	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("failed to read %d bytes from reader for IV", aes.BlockSize)
	}
	js := make([]byte, 0, 2048)
	tmp := make([]byte, 256)
	for {
		n, err = r.Read(tmp)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		js = append(js, tmp[:n]...)
	}
	_, _ = h.Write(iv)
	_, _ = h.Write(js)
	calculatedMac := h.Sum(nil)
	if !hmac.Equal(mac, calculatedMac) {
		return nil, errors.New("bad mac")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCDecrypter(block, iv)
	cbc.CryptBlocks(js, js)
	return unpad(js), nil
}

func encrypt(key []byte, plaintext []byte, w io.Writer) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	padded := pad(plaintext)
	ct := make([]byte, aes.BlockSize+len(padded))
	iv := ct[:aes.BlockSize]
	n, err := rand.Read(iv)
	if err != nil {
		return err
	}
	if n != len(iv) {
		return fmt.Errorf("needed %d bytes for IV but got %d", aes.BlockSize, n)
	}
	cbc := cipher.NewCBCEncrypter(block, iv)
	cbc.CryptBlocks(ct[aes.BlockSize:], padded)
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(ct)
	mac := h.Sum(nil)
	wrote, err := w.Write(mac)
	if err != nil {
		return err
	}
	if wrote != h.Size() {
		return errors.New("invalid write cound for mac")
	}
	wrote, err = w.Write(ct)
	if err != nil {
		return err
	}
	if wrote != len(ct) {
		return errors.New("invalid write count for cipher text")
	}
	return nil
}

func genKey(pw []byte, salt []byte) ([]byte, []byte, error) {
	if salt == nil || len(salt) == 0 {
		salt = make([]byte, saltSize)
		n, err := rand.Read(salt)
		if err != nil {
			return nil, nil, err
		}
		if n != saltSize {
			return nil, nil, fmt.Errorf("needed %d bytes for salt but got %d", saltSize, n)
		}
	}
	rounds := PBKDF2Rounds
	userRounds := os.Getenv(RoundsEnvironmentalVariableName)
	if userRounds != "" {
		v, err := strconv.ParseUint(userRounds, 10, 32)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"user chosen value for %s is not an unsigned integer: %s",
				RoundsEnvironmentalVariableName, userRounds,
			)
		}
		if int(v) > rounds {
			rounds = int(v)
		}
	}
	key := pbkdf2.Key(pw, salt, rounds, keyLength, sha256.New)
	return key, salt, nil
}

func pwFromEnv() []byte {
	pw := os.Getenv(KeyEnvironmentalVariableName)
	if pw == "" {
		return nil
	}
	return []byte(pw)
}

// PKCS7 padding

func pad(b []byte) []byte {
	pad := byte(aes.BlockSize - len(b)%aes.BlockSize)
	if pad == 0 {
		pad = aes.BlockSize
	}
	for i := byte(0); i < pad; i++ {
		b = append(b, pad)
	}
	return b
}

func unpad(b []byte) []byte {
	pad := int(b[len(b)-1])
	if pad > len(b) {
		return []byte{}
	}
	return b[:len(b)-pad]
}
