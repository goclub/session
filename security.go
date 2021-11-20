package sess

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	xerr "github.com/goclub/error"
	xrand "github.com/goclub/rand"
	"log"
)

type Security interface {
	Encrypt (storeKey []byte, securityKey []byte) (sessionID []byte, err error)
	Decrypt (sessionID []byte, securityKey []byte) (storeKey []byte, err error)
}
// 仅限于演示代码时使用的秘钥生成函数，正式环境请自行生成 长度为 32 的 []byte,并保存在配置文件或配置中心中。
func TemporarySecretKey() []byte {
	log.Print("goclub/session: TemporarySecretKey() You are using temporary secret key, make sure it's not running in production environment")
	return []byte(`b003534153a14a66adc7ddd0c9e545d8`)
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
		return nil, xerr.New("goclub/session: decrypt sessionID fail")
	}
	iv := sessionID[0:viSize]
	ciphertext:= sessionID[viSize:]
	return securityAesDecrypt(ciphertext, securityKey, iv)
}

var (
	// ErrInvalidBlockSize indicates hash blocksize <= 0.
	ErrInvalidBlockSize = xerr.New("invalid blocksize, sess.HubOption{}.SecureKey is wrong or someone forged a incorrect session")

	// ErrInvalidPKCS7Data indicates bad input to PKCS7 pad or unpad.
	ErrInvalidPKCS7Data = xerr.New("invalid PKCS7 data (empty or not padded), sess.HubOption{}.SecureKey is wrong or someone forged a incorrect session")

	// ErrInvalidPKCS7Padding indicates PKCS7 unpad fails to bad input.
	ErrInvalidPKCS7Padding = xerr.New("invalid padding on input, sess.HubOption{}.SecureKey is wrong or someone forged a incorrect session")
)
// https://gist.github.com/huyinghuan/7bf174017bf54efb91ece04a48589b22
func securityPKCS7Padding(b []byte, blocksize int) ([]byte, error) {
	if blocksize <= 0 {
		return nil, ErrInvalidBlockSize
	}
	if b == nil || len(b) == 0 {
		return nil, ErrInvalidPKCS7Data
	}
	n := blocksize - (len(b) % blocksize)
	pb := make([]byte, len(b)+n)
	copy(pb, b)
	copy(pb[len(b):], bytes.Repeat([]byte{byte(n)}, n))
	return pb, nil
}

func securityPKCS7UnPadding(b []byte, blocksize int) ([]byte, error) {
	if blocksize <= 0 {
		return nil, ErrInvalidBlockSize
	}
	if b == nil || len(b) == 0 {
		return nil, ErrInvalidPKCS7Data
	}
	if len(b)%blocksize != 0 {
		return nil, ErrInvalidPKCS7Padding
	}
	c := b[len(b)-1]
	n := int(c)
	if n == 0 || n > len(b) {
		return nil, ErrInvalidPKCS7Padding
	}
	for i := 0; i < n; i++ {
		if b[len(b)-n+i] != c {
			return nil, ErrInvalidPKCS7Padding
		}
	}
	return b[:len(b)-n], nil
}


func securityAesEncrypt(plaintext []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	plaintext, err = securityPKCS7Padding(plaintext, blockSize) ; if err != nil {
	    return nil, err
	}
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
	origData, err = securityPKCS7UnPadding(origData, aes.BlockSize) ; if err != nil {
	    return nil, err
	}
	return origData, nil
}

