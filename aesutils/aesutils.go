package aesutils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/enceve/crypto/cmac"
)

const (
	LoraRawCmacLen   = 4
	LoraRawHeaderLen = 4
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

func Decrypt(data, key []byte) ([]byte, error) {
	// Check CMAC
	cm := data[len(data)-LoraRawCmacLen:]
	cmacTest, err := prepareCMAC(data[:len(data)-LoraRawCmacLen], key)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("CMAC generation failed: %s", err))
	}

	if !bytes.Equal(cm, cmacTest) {
		return nil, errors.New("incorrect CMAC or Key")
	}

	// Decrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(data)-LoraRawHeaderLen-LoraRawCmacLen)
	mode.CryptBlocks(decrypted, data[LoraRawHeaderLen:len(data)-LoraRawCmacLen])

	// Append header and CMAC
	result := append(data[:LoraRawHeaderLen], decrypted...)
	result = append(result, cm...)
	return result, nil
}

func DecryptLegacy(data, key []byte) ([]byte, error) {
	// Decrypt
	if len(data) < 16 {
		return nil, errors.New("invalid data length for DecryptLegacy")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decLength := len(data) - (len(data) % 16)
	decrypted := make([]byte, decLength)
	mode.CryptBlocks(decrypted, data[:decLength])
	decrypted = append(decrypted, data[decLength:]...)
	return decrypted, nil
}

func prepareCMAC(data, key []byte) ([]byte, error) {
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create CMAC object
	cmacObj, err := cmac.New(block)
	if err != nil {
		return nil, err
	}

	// Update CMAC with parts of the data
	cmacObj.Write(data[:16])
	cmacObj.Write(data[4:])

	// Compute MAC and return the first 4 bytes (MAC length of 4 bytes)
	mac := cmacObj.Sum(nil)
	return mac[:4], nil
}
