// Package rsautil Функция для RSA
package rsautil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/utils/errutil"
	"math/big"
	"net"
	"os"
	"time"
)

var label = []byte("OAEP Encrypted")

func ReadPublicKey(publicKeyPath string) (*rsa.PublicKey, error) {

	pub, err := os.ReadFile(publicKeyPath)
	if err != nil {
		err = fmt.Errorf("path: %s, error: %w", publicKeyPath, err)
		return nil, errutil.WrapError(err)
	}

	pubPem, _ := pem.Decode(pub)
	if pubPem == nil {
		return nil, errutil.WrapError(err)
	}

	if pubPem.Type != "RSA PUBLIC KEY" && pubPem.Type != "PUBLIC KEY" {
		err = fmt.Errorf("RSA public key is of the wrong type, Pem Type :%s", pubPem.Type)
		return nil, errutil.WrapError(err)
	}

	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKIXPublicKey(pubPem.Bytes); err != nil {
		err = fmt.Errorf("unable to parse RSA public key: %w", err)
		return nil, errutil.WrapError(err)
	}

	var pubKey *rsa.PublicKey
	var ok bool
	if pubKey, ok = parsedKey.(*rsa.PublicKey); !ok {
		err = fmt.Errorf("unable to parse RSA public key: %w", err)
		return nil, errutil.WrapError(err)
	}

	return pubKey, nil

}

func Encrypt(publicKey *rsa.PublicKey, message []byte) ([]byte, error) {

	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, publicKey, message, label)
	if err != nil {
		return nil, errutil.WrapError(err)
	}
	return ciphertext, nil
}

func ReadPrivateKey(privateKeyFile string) (*rsa.PrivateKey, error) {

	priv, err := os.ReadFile(privateKeyFile)
	if err != nil {
		err = fmt.Errorf("path: %s, error: %w", privateKeyFile, err)
		return nil, errutil.WrapError(err)
	}

	privPem, _ := pem.Decode(priv)
	var privPemBytes []byte
	if privPem.Type != "RSA PRIVATE KEY" {
		err = fmt.Errorf("RSA private key is of the wrong type :%s", privPem.Type)
		return nil, errutil.WrapError(err)
	}
	privPemBytes = privPem.Bytes

	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPemBytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes); err != nil { // note this returns type `interface{}`
			err = fmt.Errorf("unable to parse RSA private key %w", err)
			return nil, errutil.WrapError(err)
		}
	}

	var privateKey *rsa.PrivateKey
	var ok bool
	privateKey, ok = parsedKey.(*rsa.PrivateKey)
	if !ok {
		err = fmt.Errorf("unable to parse RSA private key %w", err)
		return nil, errutil.WrapError(err)
	}

	return privateKey, nil

}

func Decrypt(privateKey *rsa.PrivateKey, message []byte) ([]byte, error) {

	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, privateKey, message, label)
	if err != nil {
		return nil, errutil.WrapError(fmt.Errorf("%w, message: \"%s\"", err, message))
	}
	return plaintext, nil

}

func GenerateCertificateHTTPS(pathCert, pathRSAPrivateKey string) error {

	templateCert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{"Yandex.Praktikum"},
			Country:      []string{"RU"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, templateCert, templateCert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	var certPEM bytes.Buffer
	pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	var privateKeyPEM bytes.Buffer
	pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	err = os.WriteFile(pathCert, certPEM.Bytes(), 0666)
	if err != nil {
		return err
	}

	err = os.WriteFile(pathRSAPrivateKey, privateKeyPEM.Bytes(), 0666)
	if err != nil {
		return err
	}

	return nil
}
