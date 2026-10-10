package package_registry

import (
	"bytes"
	"crypto/rsa"
	"encoding/hex"
	"testing"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
	"google.golang.org/protobuf/proto"
)

func TestLatestPointer(t *testing.T) {

	dns := &mockDNS{lookupResult: make(map[string]rsa.PublicKey)}
	domains := []string{"example.com"}

	privateKey, err := addDomains(dns, domains)
	if err != nil {
		t.Fatal("Failed to add domains to DNS:", err)
	}

	node, kadErr := kademlia.NewKademlia(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "127.0.0.1:8080")

	if kadErr != nil {
		t.Error(kadErr)
	}

	defer node.Close()

	reg := &registry{
		Kademlia: node,
		DNS:      dns,
	}

	instance := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "1.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: []byte("test-previous-version-record"),
		sig:                   []byte("test-signature"),
	}

	setErr := reg.setLatestPointer(instance, privateKey[domains[0]])

	if setErr != nil {
		t.Error(setErr)
	}

	res, getErr := reg.getLatestPointer(domains[0], "test-package")

	if getErr != nil {
		t.Error(getErr)
	}

	if res.version != "1.0.0" {
		t.Error("This should be the version")
	}

	if res.tag != "latest-pointer" {
		t.Error("It should always be latest pointer")
	}

	blob := []byte("test-blob-hash")

	recordBytes, _, err := reg.Kademlia.LookupData(
		hex.EncodeToString(res.versionRecordHash),
	)
	if err != nil {
		t.Error(err)
	}

	var recordProto VersionRecordProto

	errProto := proto.Unmarshal(recordBytes, &recordProto)

	if errProto != nil {
		t.Error(errProto)
	}

	blobHash := recordProto.BlobHash

	if !bytes.Equal(blobHash, blob) {
		t.Error("They should be equal the blobs")
	}

}

func TestVerifyingWorng(t *testing.T) {
	dns := &mockDNS{lookupResult: make(map[string]rsa.PublicKey)}
	domains := []string{"example.com"}

	privateKey, err := addDomains(dns, domains)
	if err != nil {
		t.Fatal("Failed to add domains to DNS:", err)
	}

	node, kadErr := kademlia.NewKademlia(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "127.0.0.1:8080")

	if kadErr != nil {
		t.Error(kadErr)
	}

	defer node.Close()

	reg := &registry{
		Kademlia: node,
		DNS:      dns,
	}

	instance := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "1.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: []byte("test-previous-version-record"),
		sig:                   []byte("test-signature"),
	}

	setErr := reg.setLatestPointer(instance, privateKey[domains[0]])

	if setErr != nil {
		t.Error(setErr)
	}

	res, getErr := reg.getLatestPointer(domains[0], "test-package")

	if getErr != nil {
		t.Error(getErr)
	}

	res.packageName = "fail"

	publicKey, puberr := reg.DNS.lookup(domains[0])

	if puberr != nil {
		t.Error(puberr)
	}

	errVerify := res.verifySignaturePointer(&publicKey)

	if errVerify == nil {
		t.Error("This should fail, since varification is wrong")
	}

}
