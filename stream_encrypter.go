package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"hash"
	"io"
)

// NewStreamEncrypter creates a new stream encrypter
func NewStreamEncrypter(encKey, macKey []byte, plainText io.Reader) (*StreamEncrypter, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, block.BlockSize())
	_, err = rand.Read(iv)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, iv)
	mac := hmac.New(sha256.New, macKey)
	return &StreamEncrypter{
		Source: plainText,
		Block:  block,
		Stream: stream,
		Mac:    mac,
		IV:     iv,
	}, nil
}

// StreamEncrypter is an encrypter for a stream of data with authentication
type StreamEncrypter struct {
	Source io.Reader
	Block  cipher.Block
	Stream cipher.Stream
	Mac    hash.Hash
	IV     []byte
}

// Read encrypts the bytes of the inner reader and places them into p
func (s *StreamEncrypter) Read(p []byte) (int, error) {
	n, readErr := s.Source.Read(p)
	if n > 0 {
		s.Stream.XORKeyStream(p[:n], p[:n])
		err := writeHash(s.Mac, p[:n])
		if err != nil {
			return n, err
		}
		return n, readErr
	}
	return 0, io.EOF
}

func writeHash(mac hash.Hash, p []byte) error {
	m, err := mac.Write(p)
	if err != nil {
		return err
	}
	if m != len(p) {
		return errors.New("could not write all bytes to hmac")
	}
	return nil
}
