package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

func copyStream(stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {
	var (
		// size based on io.Copy buffer size
		buf    = make([]byte, 32*1024)
		length = blockSize
	)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			wn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			length += wn

		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	return length, nil
}

func CopyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	const op = "encrypt.CopyDecrypt"
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, fmt.Errorf("%s:%w", op, err)
	}
	// In our case IV will be in the first block.BlockSize() bytes we read from src
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, fmt.Errorf("%s:%w", op, err)
	}
	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)

}

// CopyEncrypt encrypts data and copies it from src to dst
func CopyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	const op = "encrypt.CopyEncrypt"
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, fmt.Errorf("%s:%w", op, err)
	}
	iv := make([]byte, block.BlockSize()) // 16 bytes
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, fmt.Errorf("%s:%w", op, err)
	}
	//  we need to prepend iv to our file so we can decrypt it in future
	if _, err := dst.Write(iv); err != nil {
		return 0, fmt.Errorf("%s:%w", op, err)
	}
	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)
}

func NewEcryptionKey() []byte {
	buf := make([]byte, 32)
	io.ReadFull(rand.Reader, buf)
	return buf
}
