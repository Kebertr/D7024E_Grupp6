package package_registry

import (
	"crypto/rsa"
	"testing"
)

func TestGenerateKeys(t *testing.T) {
	privateKey, publicKey, err := generateKeys()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	if privateKey == nil || publicKey == nil {
		t.Fatal("Generated keys are nil")
	}
	t.Log(privateKey, "and the public key", publicKey)
}

func TestAddDomainsToDNS(t *testing.T) {
	dns := &mockDNS{lookupResult: make(map[string]rsa.PublicKey)}
	domains := []string{"example.com", "test.com", "mydomain.org"}
	privateKeys, err := addDomains(dns, domains)
	if err != nil {
		t.Fatal("Failed to add domains to DNS:", err)
	}

	if len(privateKeys) != 3 {
		t.Fatal("Expected 3 private keys")
	}

	for i := 0; i < len(domains); i++ {
		domain := domains[i]
		_, err := dns.lookup(domain)
		if err != nil {
			t.Fatal("Failed to lookup domain:", domain, err)
		}

		if privateKeys[domain] == nil {
			t.Fatal("Private key for domain is nil:", domain)
		}
	}
}
