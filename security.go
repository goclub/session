package sess

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	xrand "github.com/goclub/rand"
	"github.com/google/uuid"
	"log"
	"strings"
)

type Security interface {
	Encrypt (storeKey []byte, securityKey []byte) (sessionID []byte, err error)
	Decrypt (sessionID []byte, securityKey []byte) (storeKey []byte, err error)
}
// 仅限于演示代码时使用的秘钥生成函数，正式环境请自行生成 长度为 32 的 []byte,并保存在配置文件或配置中心中。
func TemporarySecretKey() []byte {
	log.Print("goclub/session: TemporarySecretKey() You are using temporary secret key, make sure it's not running in production environment")
	return []byte(strings.ReplaceAll(uuid.New().String(), "-", ""))
}
type DefaultSecurity struct {}
const viSize = 16
func (DefaultSecurity) Encrypt(storeKey []byte, securityKey []byte) (sessionID []byte, err error) {
	iv, err := xrand.BytesBySeed([]byte("abcdefghijklmnopqrstuvwxyz"), viSize) ; if err != nil {
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

