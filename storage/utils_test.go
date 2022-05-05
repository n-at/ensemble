package storage

import (
	"encoding/hex"
	"regexp"
	"testing"
)

func TestNewId(t *testing.T) {
	id := NewId()
	if len(id) == 0 {
		t.Fatal("empty id")
	}

	//2cc06884-c95d-44af-a951-876da22d997a

	ok, err := regexp.MatchString("^[\\da-f]{8}-[\\da-f]{4}-[\\da-f]{4}-[\\da-f]{4}-[\\da-f]{12}$", id)
	if err != nil {
		t.Fatalf("match error: %s", err)
	}
	if !ok {
		t.Fatalf("not a UUID: %s", id)
	}
}

func TestSha256(t *testing.T) {
	testString := "helloworld"
	testHash := "936a185caaa266bb9cbe981e9e05cb78cd732b0b3280eb944412bb6f8f8f07af"

	hash := Sha256(testString)
	if len(hash) != 32 {
		t.Fatalf("not a 32-byte hash")
	}
	if hex.EncodeToString(hash) != testHash {
		t.Fatalf("wrong hash")
	}
}

func TestEncryptPassword(t *testing.T) {
	testString := "helloworld"

	hash, err := EncryptPassword(testString)
	if err != nil {
		t.Fatalf("encrypt error: %s", err)
	}
	if len(hash) == 0 {
		t.Fatalf("empty hash")
	}
	if !CheckPassword(testString, hash) {
		t.Fatalf("check password failed")
	}
}

func TestCheckPassword(t *testing.T) {
	testString := "helloworld"
	testHash := "$2a$15$l3J9nrdGujVWHUzgbRZ7LuYjPdf/rvmn3gQMbiOn6sRog.Lca8twy"

	if !CheckPassword(testString, testHash) {
		t.Fatalf("check password failed")
	}
}

func TestEncryptString(t *testing.T) {
	testString := "helloworld"
	testKey := "testing"

	emptyKeyResult, err := EncryptString("", testString)
	if err != nil {
		t.Fatalf("encrypt with empty key error: %s", err)
	}
	if testString != emptyKeyResult {
		t.Fatalf("encrypt with empty key should return input string '%s', returned: %s", testString, emptyKeyResult)
	}

	encryptResult, err := EncryptString(testKey, testString)
	if err != nil {
		t.Fatalf("encrypt error: %s", err)
	}
	if len(encryptResult) == 0 {
		t.Fatalf("encrypt empty result: %s", err)
	}

	decryptResult, err := DecryptString(testKey, encryptResult)
	if err != nil {
		t.Fatalf("decrypt encrypted error: %s", err)
	}
	if decryptResult != testString {
		t.Fatalf("encrypt/decrypt result mismatch, expected '%s', returned: %s", testString, decryptResult)
	}
}

func TestDecryptString(t *testing.T) {
	testString := "helloworld"

	emptyKeyResult, err := DecryptString("", testString)
	if err != nil {
		t.Fatalf("decrypt with empty key error: %s", err)
	}
	if emptyKeyResult != testString {
		t.Fatalf("decrypt with empty key should return input string '%s', returned: %s", testString, emptyKeyResult)
	}
}
