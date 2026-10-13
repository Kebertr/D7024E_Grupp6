package package_registry

import (
	"crypto/rand"
	"crypto/rsa"
)

func AddDomains(dns *MockDNS, domains []string) (privateKeys map[string]*rsa.PrivateKey, err error) {
	privateKeys = make(map[string]*rsa.PrivateKey)

	for i := 0; i < len(domains); i++ {
		privateKey, publicKey, err := GenerateKeys()
		if err != nil {
			return nil, err
		}
		dns.myLookup.Lock()
		dns.lookupResult[domains[i]] = *publicKey
		dns.myLookup.Unlock()
		privateKeys[domains[i]] = privateKey
	}

	return privateKeys, nil
}
func GenerateKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	publicKey := &privateKey.PublicKey
	return privateKey, publicKey, nil
}

func (m *MockDNS) AddDomainCli(domain string, publicKey rsa.PublicKey) {
	m.myLookup.Lock()
	defer m.myLookup.Unlock()
	m.lookupResult[domain] = publicKey
}
