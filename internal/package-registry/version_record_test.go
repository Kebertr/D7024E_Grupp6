package package_registry

import "testing"

func TestSignAndValidation(t *testing.T) {
	privateKey, publicKey, err := generateKeys()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	instance := &VersionRecord{
		tag:         "test-tag",
		domainName:  "example.com",
		packageName: "test-package",
		version:     "1.0.0",
		blobHash:    []byte("test-blob-hash"),
	}

	errsign := instance.sign(privateKey)
	if errsign != nil {
		t.Fatal("Failed to sign version record:", errsign)
	}

	errverify := instance.verifySignature(publicKey)
	if errverify != nil {
		t.Fatal("Failed to validate version record:", errverify)
	}
}

func TestFailPublickey(t *testing.T) {
	privateKey, _, err := generateKeys()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	instance := &VersionRecord{
		tag:         "test",
		domainName:  "example.com",
		packageName: "test-package",
		version:     "1.0.0",
		blobHash:    []byte("test-blob-hash"),
	}

	errsign := instance.sign(privateKey)
	if errsign != nil {
		t.Fatal("Failed to sign version record:", errsign)
	}

	_, fakePublicKey, err := generateKeys()
	if err != nil {
		t.Fatalf("Failed to generate fake public key: %v", err)
	}

	errverify := instance.verifySignature(fakePublicKey)
	if errverify == nil {
		t.Fatal("Since it it wrong public key this should fail:", errverify)
	}
}

func TestFailprivateKey(t *testing.T) {
	_, publicKey, err := generateKeys()

	if err != nil {
		t.Fatal("Failed to generate keys", err)
	}

	fakePrivateKey, _, err := generateKeys()

	instance := &VersionRecord{
		tag:         "test",
		domainName:  "example.com",
		packageName: "test-package",
		version:     "1.0.0",
		blobHash:    []byte("test-blob-hash"),
	}

	errsign := instance.sign(fakePrivateKey)
	if errsign != nil {
		t.Fatal("Failed to sign version record:", errsign)
	}
	errverify := instance.verifySignature(publicKey)
	if errverify == nil {
		t.Fatal("Since it is the wrong private key this should fail:", errverify)
	}
}

func TestFailChangeTag(t *testing.T) {
	privateKey, publicKey, err := generateKeys()

	if err != nil {
		t.Fatal("Failed to generate keys", err)
	}

	instance := &VersionRecord{
		tag:         "test",
		domainName:  "example.com",
		packageName: "test-package",
		version:     "1.0.0",
		blobHash:    []byte("test-blob-hash"),
	}

	errsign := instance.sign(privateKey)
	if errsign != nil {
		t.Fatal("Failed to sign version record:", errsign)
	}

	instance = &VersionRecord{
		tag:         "test1",
		domainName:  "example.com",
		packageName: "test-package",
		version:     "1.0.0",
		blobHash:    []byte("test-blob-hash"),
	}

	errverify := instance.verifySignature(publicKey)
	if errverify == nil {
		t.Fatal("Since the tag has been changed this should fail:", errverify)
	}
}
