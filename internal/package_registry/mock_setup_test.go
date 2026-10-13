package package_registry

import (
	"crypto/rsa"
	sync "sync"
	"testing"
)

func TestGenerateKeys(t *testing.T) {
	privateKey, publicKey, err := GenerateKeys()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	if privateKey == nil || publicKey == nil {
		t.Fatal("Generated keys are nil")
	}
	t.Log(privateKey, "and the public key", publicKey)
}

func TestAddDomainsToDNS(t *testing.T) {
	dns := &MockDNS{lookupResult: make(map[string]rsa.PublicKey)}
	domains := []string{"example.com", "test.com", "mydomain.org"}
	privateKeys, err := AddDomains(dns, domains)
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

func TestNewMock(t *testing.T) {
	dns := NewMockDNS()

	if dns == nil {
		t.Error("It sohuld not be nil")
	}

	if dns.lookupResult == nil {
		t.Error("lookupserulst should give soemthing")
	}

}

func TestAddDomainCli(t *testing.T) {
	dns := NewMockDNS()

	_, publicKey, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	dns.AddDomainCli("example.com", *publicKey)

	_, errLook := dns.lookup("example.com")
	if errLook != nil {
		t.Fatal(err)
	}
}

func TestMockDNSRace(t *testing.T) {
	dns := NewMockDNS()

	_, publicKey, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			dns.AddDomainCli("example.com", *publicKey)
		}()

		go func() {
			defer wg.Done()
			_, _ = dns.lookup("example.com")
		}()
	}

	wg.Wait()
}
