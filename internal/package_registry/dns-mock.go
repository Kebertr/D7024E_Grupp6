package package_registry

import (
	"crypto/rsa"
	"errors"
)

type MockDNS struct {
	lookupResult map[string]rsa.PublicKey
}

func NewMockDNS() *MockDNS {
	return &MockDNS{
		lookupResult: make(map[string]rsa.PublicKey),
	}
}

func (m *MockDNS) lookup(domain string) (rsa.PublicKey, error) {
	res, err := m.lookupResult[domain]
	if !err {
		return rsa.PublicKey{}, errors.New("Domain not found")
	}
	return res, nil
}
