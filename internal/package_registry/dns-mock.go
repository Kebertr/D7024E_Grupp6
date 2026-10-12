package package_registry

import (
	"crypto/rsa"
	"errors"
	sync "sync"
)

type MockDNS struct {
	lookupResult map[string]rsa.PublicKey
	myLookup     sync.RWMutex
}

func NewMockDNS() *MockDNS {
	return &MockDNS{
		lookupResult: make(map[string]rsa.PublicKey),
	}
}

func (m *MockDNS) lookup(domain string) (rsa.PublicKey, error) {
	m.myLookup.RLock()
	defer m.myLookup.RUnlock()
	res, err := m.lookupResult[domain]
	if !err {
		return rsa.PublicKey{}, errors.New("Domain not found")
	}
	return res, nil
}
