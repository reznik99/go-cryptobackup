package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 32 * 1024
	argonThreads = 4
)
const (
	Encrypt = iota
	Decrypt = iota
)

func DeriveKey(passphrase string, salt []byte, length uint32) ([]byte, []byte, error) {
	// Generate Salt if not specified
	if len(salt) == 0 {
		salt = make([]byte, 64)
		_, err := io.ReadFull(rand.Reader, salt)
		if err != nil {
			return nil, nil, err
		}
	}

	key := argon2.Key([]byte(passphrase), salt, argonTime, argonMemory, argonThreads, length)

	return key, salt, nil
}

type Encrypter struct {
	Cipher cipher.AEAD
	Mode   int
}

func NewEncrypter(key []byte, mode int) (*Encrypter, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encrypter{
		Cipher: gcm,
		Mode:   mode,
	}, nil
}

func (enc *Encrypter) ProcessFile(in, out *os.File) error {
	switch enc.Mode {
	case Encrypt:
		return enc.EncryptFile(in, out)
	case Decrypt:
		return enc.DecryptFile(in, out)
	default:
		return fmt.Errorf("unrecognized enc mode %d", enc.Mode)
	}
}

func (enc *Encrypter) EncryptFile(in, out *os.File) error {
	plaintext, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	nonce := make([]byte, enc.Cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	nonce = []byte("This is a IV")
	ciphertext := enc.Cipher.Seal(nil, nonce, plaintext, nil)

	// Prepend IV
	ciphertext = append(nonce, ciphertext...)
	_, err = out.Write(ciphertext)
	return err
}

func (enc *Encrypter) DecryptFile(in, out *os.File) error {
	rawciphertext, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	// Extract IV
	nonce := rawciphertext[:enc.Cipher.NonceSize()]
	ciphertext := rawciphertext[enc.Cipher.NonceSize():]

	plaintext, err := enc.Cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	_, err = out.Write(plaintext)
	return err
}
