package rsautil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	error2 "github.com/s-turchinskiy/metrics/internal/common/error"
	"os"
)

func ReadPublicKey(publicKeyPath string) (*rsa.PublicKey, error) {

	pub, err := os.ReadFile(publicKeyPath)
	if err != nil {
		err = fmt.Errorf("path: %s, error: %w", publicKeyPath, err)
		return nil, error2.WrapError(err)
	}

	pubPem, _ := pem.Decode(pub)
	if pubPem == nil {
		return nil, error2.WrapError(err)
	}

	if pubPem.Type != "RSA PUBLIC KEY" && pubPem.Type != "PUBLIC KEY" {
		err := fmt.Errorf("RSA public key is of the wrong type, Pem Type :%s", pubPem.Type)
		return nil, error2.WrapError(err)
	}

	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKIXPublicKey(pubPem.Bytes); err != nil {
		err = fmt.Errorf("unable to parse RSA public key: %w", err)
		return nil, error2.WrapError(err)
	}

	var pubKey *rsa.PublicKey
	var ok bool
	if pubKey, ok = parsedKey.(*rsa.PublicKey); !ok {
		err = fmt.Errorf("unable to parse RSA public key: %w", err)
		return nil, error2.WrapError(err)
	}

	return pubKey, nil

}

func Encrypt(publicKey *rsa.PublicKey, message []byte) ([]byte, error) {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, publicKey, message, label)
	if err != nil {
		return nil, error2.WrapError(err)
	}
	return ciphertext, nil
}

func ReadPrivateKey(privateKeyFile string) (*rsa.PrivateKey, error) {

	priv, err := os.ReadFile(privateKeyFile)
	if err != nil {
		err = fmt.Errorf("path: %s, error: %w", privateKeyFile, err)
		return nil, error2.WrapError(err)
	}

	privPem, _ := pem.Decode(priv)
	var privPemBytes []byte
	if privPem.Type != "RSA PRIVATE KEY" {
		err = fmt.Errorf("RSA private key is of the wrong type :%s", privPem.Type)
		return nil, error2.WrapError(err)
	}
	privPemBytes = privPem.Bytes

	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPemBytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes); err != nil { // note this returns type `interface{}`
			err = fmt.Errorf("unable to parse RSA private key %w", err)
			return nil, error2.WrapError(err)
		}
	}

	var privateKey *rsa.PrivateKey
	var ok bool
	privateKey, ok = parsedKey.(*rsa.PrivateKey)
	if !ok {
		err = fmt.Errorf("unable to parse RSA private key %w", err)
		return nil, error2.WrapError(err)
	}

	return privateKey, nil

}

func Decrypt(privateKey *rsa.PrivateKey, message []byte) ([]byte, error) {

	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, privateKey, message, label)
	if err != nil {
		return nil, error2.WrapError(fmt.Errorf("%w, message: \"%s\"", err, message))
	}
	return plaintext, nil

}
