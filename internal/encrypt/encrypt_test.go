package encrypt

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_CopyEncrypt_CopyDecrypt(t *testing.T) {
	message := "Pupa and Lupa"
	// Encrypting message and coping to the dst
	src := bytes.NewReader([]byte(message))
	dst := new(bytes.Buffer)
	key := NewEcryptionKey()
	_, err := CopyEncrypt(key, src, dst)
	require.Nil(t, err)

	// Decrypting message and coping to the out

	out := new(bytes.Buffer)
	_, err = CopyDecrypt(key, dst, out)
	require.Nil(t, err)
	// Checking whether data was corrupted
	require.Equal(t, out.String(), message)
}
