package package_registry

import (
	"bytes"
	"testing"
)

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

func TestHashBlob(t *testing.T) {
	data := []byte("test")
	instance := &VersionRecord{}
	instance2 := &VersionRecord{}

	err1 := instance.hashBlob(data)
	if err1 != nil {
		t.Fatal("Failed to hash blob:", err1)
	}
	err2 := instance2.hashBlob(data)
	if err2 != nil {
		t.Fatal("Failed to hash blob:", err2)
	}

	if !bytes.Equal(instance.blobHash, instance2.blobHash) {
		t.Fatal("Hashes should be the same")
	}
}

func TestOldRecord(t *testing.T) {

	instance := &VersionRecord{
		tag:                   "test",
		domainName:            "example.com",
		packageName:           "test-package",
		version:               "1.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: []byte("test-previous-version-record"),
		sig:                   []byte("test-signature"),
	}

	instance2 := &VersionRecord{
		tag:                   "test",
		domainName:            "example.com",
		packageName:           "test-package",
		version:               "1.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: []byte("test-previous-version-record"),
		sig:                   []byte("test-signature"),
	}

	err1 := instance.hashOldRecord(instance)
	if err1 != nil {
		t.Fatal("Failed to hash old record:", err1)
	}
	err2 := instance2.hashOldRecord(instance2)
	if err2 != nil {
		t.Fatal("Failed to hash old record:", err2)
	}

	if !bytes.Equal(instance.previousVersionRecord, instance2.previousVersionRecord) {
		t.Fatal("Hashes should be the same")
	}
}
