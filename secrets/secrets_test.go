package secrets

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestEncryption(t *testing.T) {
	alice, _ := GenerateKey()
	bob, _ := GenerateKey()
	alicePub, _ := PublicKeyFromSeed(alice)
	bobPub, _ := PublicKeyFromSeed(bob)
	message := "This is a test message"
	encrypted, err := Encrypt(alice, bobPub, message)
	if err != nil {
		t.Fatalf("encryption error: %v\n", err)
	}
	encoded := Encode(alicePub, encrypted)
	fmt.Println(encoded)
	// ===========
	split := strings.Split(encoded[2:len(encoded)-2], ":")
	encodedPubkey, _ := base64.StdEncoding.DecodeString(split[0])
	encodedData, _ := base64.StdEncoding.DecodeString(split[1])
	decrypted, err := Decrypt(bob, encodedPubkey, encodedData)
	if err != nil {
		t.Fatalf("decryption error: %v\n", err)
	}
	if decrypted != message {
		t.Fatal("data mismatch")
	}
}
