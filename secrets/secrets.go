package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

type Seed []byte
type PublicKey []byte

func PublicKeyFromSeed(seed Seed) (PublicKey, error) {
	return curve25519.X25519(seed, curve25519.Basepoint)
}

func SharedSecret(seed Seed, point PublicKey) ([]byte, error) {
	return curve25519.X25519(seed, point)
}

func GenerateKey() (Seed, error) {
	out := make([]byte, 32)
	_, err := rand.Read(out)
	return out, err
}

func Encrypt(seed Seed, point PublicKey, message string) ([]byte, error) {
	key, err := SharedSecret(seed, point)
	if err != nil {
		return nil, err
	}
	cipher, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(message)+cipher.Overhead())
	nonce := make([]byte, cipher.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}
	out = cipher.Seal(out, nonce, []byte(message), []byte{})
	return append(nonce, out...), nil
}

func Decrypt(seed Seed, point PublicKey, data []byte) (string, error) {
	key, err := SharedSecret(seed, point)
	if err != nil {
		return "", err
	}
	cipher, err := chacha20poly1305.New(key)
	if err != nil {
		return "", err
	}
	out := make([]byte, 0, len(data)-cipher.Overhead()-cipher.NonceSize())
	nonce := data[:cipher.NonceSize()]
	encryptedData := data[cipher.NonceSize():]
	out, err = cipher.Open(out, nonce, encryptedData, []byte{})
	return string(out), err
}

func Encode(point PublicKey, data []byte) string {
	return "$$" + base64.StdEncoding.EncodeToString(point) + ":" + base64.StdEncoding.EncodeToString(data) + "$$"
}
