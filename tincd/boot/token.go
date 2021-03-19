package boot

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

type Token string

func (t Token) Encrypt(data []byte) []byte {
	tokenData := sha256.Sum256([]byte(t)) // normalize to 32 bytes
	crypter, err := chacha20poly1305.NewX(tokenData[:])
	if err != nil {
		panic(err)
	}

	var nounce [chacha20poly1305.NonceSizeX]byte
	_, err = io.ReadFull(rand.Reader, nounce[:])
	if err != nil {
		panic(err)
	}

	encrypted := crypter.Seal(nil, nounce[:], data, nil)

	box := make([]byte, len(nounce)+len(encrypted))
	copy(box, nounce[:])
	copy(box[len(nounce):], encrypted)
	return box
}

func (t Token) Decrypt(data []byte) ([]byte, error) {
	tokenData := sha256.Sum256([]byte(t))
	crypter, err := chacha20poly1305.NewX(tokenData[:])
	if err != nil {
		return nil, err
	}
	if len(data) < chacha20poly1305.NonceSizeX {
		return nil, fmt.Errorf("package is too small")
	}
	nounce := data[:chacha20poly1305.NonceSizeX]
	return crypter.Open(nil, nounce, data[chacha20poly1305.NonceSizeX:], nil)
}
