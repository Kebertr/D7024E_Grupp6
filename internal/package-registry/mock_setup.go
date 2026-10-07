package package_registry

import (
	"crypto/rand"
	"crypto/rsa"
)

func addDomains(dns *mockDNS, domains []string) (privateKeys map[string]*rsa.PrivateKey, err error) {
	privateKeys = make(map[string]*rsa.PrivateKey)

	for i := 0; i < len(domains); i++ {
		privateKey, publicKey, err := generateKeys()
		if err != nil {
			return nil, err
		}
		dns.lookupResult[domains[i]] = *publicKey
		privateKeys[domains[i]] = privateKey
	}

	return privateKeys, nil
}
func generateKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	publicKey := &privateKey.PublicKey
	return privateKey, publicKey, nil
}
