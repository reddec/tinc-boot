package boot_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reddec/tinc-boot/tincd/boot"
)

func TestToken_EncryptDecrypt(t *testing.T) {
	const payload = "hell in the world"
	token := boot.Token("hello world")

	encrypted := token.Encrypt([]byte(payload))

	decrypted, err := token.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, payload, string(decrypted))
}
