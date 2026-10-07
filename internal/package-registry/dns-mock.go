package package_registry

import (
	"crypto/rsa"
	"errors"
)

type mockDNS struct {
	lookupResult map[string]rsa.PublicKey
}

func (m *mockDNS) lookup(domain string) (rsa.PublicKey, error) {
	res, err := m.lookupResult[domain]
	if !err {
		return rsa.PublicKey{}, errors.New("Domain not found")
	}
	return res, nil
}
