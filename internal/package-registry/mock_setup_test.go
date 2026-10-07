package package_registry

import "testing"

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
