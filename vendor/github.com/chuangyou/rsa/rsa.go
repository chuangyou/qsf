package rsa

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"

	"encoding/base64"
	"encoding/pem"
	"errors"
)

func GenRsaKey(bits int) (pubKey string, priKey string, err error) {
	var (
		derStream  []byte
		block      *pem.Block
		w          *bytes.Buffer
		privateKey *rsa.PrivateKey
		publicKey  *rsa.PublicKey
		derPkix    []byte
		w2         *bytes.Buffer
	)
	privateKey, err = rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return
	}
	derStream = x509.MarshalPKCS1PrivateKey(privateKey)
	block = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	w = bytes.NewBuffer([]byte(nil))
	err = pem.Encode(w, block)
	if err != nil {
		return
	}
	priKey = string(w.Bytes())
	// 生成公钥文件
	publicKey = &privateKey.PublicKey
	derPkix, err = x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return
	}

	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	w2 = bytes.NewBuffer(nil) //bufio.NewWriter()
	err = pem.Encode(w2, block)
	if err != nil {
		return
	}
	pubKey = string(w2.Bytes())
	return

}

func RsaEncrypt(origData, publicKey []byte) ([]byte, error) {
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("public key error")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub := pubInterface.(*rsa.PublicKey)
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData)
}

func RsaDecrypt(ciphertext, privateKey []byte) ([]byte, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("private key error!")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}
func Base64Encode(src []byte) string {
	return base64.URLEncoding.EncodeToString(src)
}

func Base64Decode(src string) []byte {
	decode, _ := base64.URLEncoding.DecodeString(src)
	return decode
}
