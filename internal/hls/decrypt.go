package hls

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
)

// decrypt undoes AES-128-CBC with PKCS#7 padding. Without an explicit IV the
// segment's sequence number is the IV, as the HLS spec says.
func decrypt(data, key []byte, k *Key, seq int64) ([]byte, error) {
	if len(data)%aes.BlockSize != 0 || len(data) == 0 {
		return nil, errors.New("encrypted segment has a bad length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	iv := k.IV
	if iv == nil {
		iv = make([]byte, 16)
		binary.BigEndian.PutUint64(iv[8:], uint64(seq))
	}
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
	pad := int(out[len(out)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(out) {
		return nil, errors.New("bad padding: wrong key?")
	}
	return out[:len(out)-pad], nil
}
