package internal

import (
	"crypto/rand"
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	pbkdf2KeyLength = 32     // Byte length for derived AES key
	pbkdf2Iter      = 65_536 // Iter difficulty for pbkdf2
)

func DeriveKey(passphrase string) ([]byte, error) {
	iv := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	key := pbkdf2.Key([]byte(passphrase), iv, pbkdf2Iter, pbkdf2KeyLength, sha256.New)

	return key, nil
}
