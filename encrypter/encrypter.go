package encrypter

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"github.com/chmike/cmac-go"
)

var nonce byte = 0
var iv = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func Encrypt(address string, data, key []byte, opts byte) ([]byte, error) {
	lengthInByte := []byte{byte(len(data))}
	optsInByte := []byte{opts}
	nonceInByte := []byte{nonce}
	nonce = (nonce + 1) & 0xFF

	data = append(optsInByte, append(nonceInByte, append(lengthInByte, data...)...)...)

	// Encrypt data
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data)%aes.BlockSize != 0 {
		padding := make([]byte, aes.BlockSize-len(data)%aes.BlockSize)
		data = append(data, padding...)
	}
	encrypted := make([]byte, len(data))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, data)

	// Decode address and combine it with encrypted data
	addressBytes, err := hex.DecodeString(address)
	if err != nil {
		return nil, err
	}
	encryptedData := append(addressBytes, encrypted...)

	// Get and Add CMAC
	mac, err := prepareCMAC(encryptedData, key)
	if err != nil {
		return nil, err
	}
	encryptedData = append(encryptedData, mac...)

	return encryptedData, nil
}

func prepareCMAC(data, key []byte) ([]byte, error) {
	// Create a new CMAC object with the given key and AES block size
	cm, err := cmac.New(aes.NewCipher, key)
	if err != nil {
		return nil, err
	}

	cm.Write(data[:16])
	cm.Write(data[4:])

	mac := cm.Sum(nil)

	return mac, nil
}
