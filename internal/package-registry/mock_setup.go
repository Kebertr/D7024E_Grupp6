package package_registry

import (
	"crypto/rand"
	"crypto/rsa"
	"log"
)

func generateKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal("Failed to generate RSA key:", err)
		return nil, nil, err
	}
	publicKey := &privateKey.PublicKey
	return privateKey, publicKey, nil
}
