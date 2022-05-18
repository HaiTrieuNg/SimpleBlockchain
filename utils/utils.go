package utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
)

func Hash(s string) []byte {
	h := sha256.New()
	h.Write([]byte(s))
	return h.Sum(nil)

}

func GenerateKeypair() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		//errors.New("Error generating key\n")
		fmt.Println("Error generating key\n")
		panic(err)
	}
	return key
}

func Sign(privKey *rsa.PrivateKey, msg string) []byte {
	sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, Hash(msg)[:])
	if err != nil {
		//errors.New("Error signing\n")
		fmt.Println("Error signing\n")
		panic(err)
	}
	return sig
}

func VerifySignature(pubKey *rsa.PublicKey, msg string, sig []byte) bool {
	err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, Hash(msg)[:], sig)
	if err != nil {
		return false
	}
	return true
}

func CalcAddress(pubKey *rsa.PublicKey) string {
	stringKey, err := json.Marshal(pubKey)
	if err != nil {
		panic(err)
	}
	return b64.StdEncoding.EncodeToString(Hash(string(stringKey)))
}

func AddressMatchesKey(addr string, pubKey *rsa.PublicKey) bool {
	return addr == CalcAddress(pubKey)
}
