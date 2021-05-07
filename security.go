package sess

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	xrand "github.com/goclub/rand"
)

type Security interface {
	Encrypt (storeKey []byte, securityKey []byte) (sessionID []byte, err error)
	Decrypt (sessionID []byte, securityKey []byte) (storeKey []byte, err error)
}
type DefaultSecurity struct {}
const viSize = 16
func (DefaultSecurity) Encrypt(storeKey []byte, securityKey []byte) (sessionID []byte, err error) {
	iv, err := bytesBySeed([]byte("abcdefghijklmnopqrstuvwxyz"), viSize) ; if err != nil {
		return
	}
	result, err := securityAesEncrypt(storeKey, securityKey, iv) ; if err != nil {
	    return
	}
	result = append(iv, result...)
	enc := base64.URLEncoding
	buf := make([]byte, enc.EncodedLen(len(result)))
	enc.Encode(buf, result)
	return buf,nil
}
func (DefaultSecurity) Decrypt(sessionID []byte, securityKey []byte) (storeKey []byte, err error) {
	enc := base64.URLEncoding
	dbuf := make([]byte, enc.DecodedLen(len(sessionID)))
	n, err := enc.Decode(dbuf, []byte(sessionID)) ; if err != nil {
	    return
	}
	sessionID = dbuf[:n]
	if len(sessionID) < viSize {
		return nil, errors.New("goclub/session: decrypt sessionID fail")
	}
	iv := sessionID[0:viSize]
	ciphertext:= sessionID[viSize:]
	return securityAesDecrypt(ciphertext, securityKey, iv)
}

func securityPKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func securityPKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}


func securityAesEncrypt(plaintext []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	plaintext = securityPKCS7Padding(plaintext, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, iv)
	crypted := make([]byte, len(plaintext))
	blockMode.CryptBlocks(crypted, plaintext)
	return crypted, nil
}

func securityAesDecrypt(ciphertext []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, iv[:blockSize])
	origData := make([]byte, len(ciphertext))
	blockMode.CryptBlocks(origData, ciphertext)
	origData = securityPKCS7UnPadding(origData)
	return origData, nil
}

func bytesBySeed(seed []byte, size int) ([]byte, error) {
	var result []byte
	for i:=0; i<size; i++ {
		randIndex, err := xrand.Int64(int64(len(seed))) ; if err != nil {
			return nil, err
		}
		result = append(result, seed[randIndex])
	}
	return result, nil
}