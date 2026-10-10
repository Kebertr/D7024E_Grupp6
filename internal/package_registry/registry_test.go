package package_registry

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
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

func TestOkRecord(t *testing.T) {

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
		previousVersionRecord: make([]byte, sha256.Size),
	}

	errInstance := instance.sign(privateKey[domains[0]])

	if errInstance != nil {
		t.Error(errInstance)
	}

	protoInstance := &VersionRecordProto{
		Tag:                   instance.tag,
		DomainName:            instance.domainName,
		PackageName:           instance.packageName,
		Version:               instance.version,
		BlobHash:              instance.blobHash,
		PreviousVersionRecord: instance.previousVersionRecord,
		Sig:                   instance.sig,
	}

	data, err := proto.Marshal(protoInstance)
	if err != nil {
		t.Fatal(err)
	}

	instanceHash := sha256.Sum256(data)

	instance2 := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "2.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: instanceHash[:],
	}

	errInstance = instance2.sign(privateKey[domains[0]])

	if errInstance != nil {
		t.Error(errInstance)
	}

	resErr := reg.CompareVersionRecords(instance2, instance)

	if resErr != nil {
		t.Error("This should be ok")
	}
}

func TestFailVersion(t *testing.T) {

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
		version:               "2.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: make([]byte, sha256.Size),
	}

	errInstance := instance.sign(privateKey[domains[0]])

	if errInstance != nil {
		t.Error(errInstance)
	}

	protoInstance := &VersionRecordProto{
		Tag:                   instance.tag,
		DomainName:            instance.domainName,
		PackageName:           instance.packageName,
		Version:               instance.version,
		BlobHash:              instance.blobHash,
		PreviousVersionRecord: instance.previousVersionRecord,
		Sig:                   instance.sig,
	}

	data, err := proto.Marshal(protoInstance)
	if err != nil {
		t.Fatal(err)
	}

	instanceHash := sha256.Sum256(data)

	instance2 := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "2.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: instanceHash[:],
	}

	errInstance = instance2.sign(privateKey[domains[0]])

	if errInstance != nil {
		t.Error(errInstance)
	}

	resErr := reg.CompareVersionRecords(instance2, instance)

	if resErr == nil {
		t.Error("This should fail since the version is the same")
	}
}

func TestWrongDomain(t *testing.T) {

	dns := &mockDNS{lookupResult: make(map[string]rsa.PublicKey)}
	domains := []string{"example.com", "test.com"}

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
		previousVersionRecord: make([]byte, sha256.Size),
	}

	errInstance := instance.sign(privateKey[domains[0]])

	if errInstance != nil {
		t.Error(errInstance)
	}

	protoInstance := &VersionRecordProto{
		Tag:                   instance.tag,
		DomainName:            instance.domainName,
		PackageName:           instance.packageName,
		Version:               instance.version,
		BlobHash:              instance.blobHash,
		PreviousVersionRecord: instance.previousVersionRecord,
		Sig:                   instance.sig,
	}

	data, err := proto.Marshal(protoInstance)
	if err != nil {
		t.Fatal(err)
	}

	instanceHash := sha256.Sum256(data)

	instance2 := &VersionRecord{
		tag:                   "test",
		domainName:            domains[1],
		packageName:           "test-package",
		version:               "2.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: instanceHash[:],
	}

	errInstance = instance2.sign(privateKey[domains[1]])

	if errInstance != nil {
		t.Error(errInstance)
	}

	resErr := reg.CompareVersionRecords(instance2, instance)

	if resErr == nil {
		t.Error("This should fail since the domain is wrong")
	}
}

func TestGettingMiddleVersions(t *testing.T) {

	dns := &mockDNS{lookupResult: make(map[string]rsa.PublicKey)}
	domains := []string{"example.com"}

	privateKey, err := addDomains(dns, domains)
	if err != nil {
		t.Fatal("Failed to add domains to DNS:", err)
	}

	node, kadErr := kademlia.NewKademlia(
		kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"),
		"127.0.0.1:8080",
	)

	if kadErr != nil {
		t.Fatal(kadErr)
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
		previousVersionRecord: make([]byte, sha256.Size),
	}

	errInstance := instance.sign(privateKey[domains[0]])
	if errInstance != nil {
		t.Fatal(errInstance)
	}

	protoInstance := &VersionRecordProto{
		Tag:                   instance.tag,
		DomainName:            instance.domainName,
		PackageName:           instance.packageName,
		Version:               instance.version,
		BlobHash:              instance.blobHash,
		PreviousVersionRecord: instance.previousVersionRecord,
		Sig:                   instance.sig,
	}

	data, err := proto.Marshal(protoInstance)
	if err != nil {
		t.Fatal(err)
	}

	instanceHash := sha256.Sum256(data)

	if err := reg.Kademlia.Store(data); err != nil {
		t.Fatal(err)
	}

	instance2 := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "2.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: instanceHash[:],
	}

	errInstance = instance2.sign(privateKey[domains[0]])
	if errInstance != nil {
		t.Fatal(errInstance)
	}

	protoInstance2 := &VersionRecordProto{
		Tag:                   instance2.tag,
		DomainName:            instance2.domainName,
		PackageName:           instance2.packageName,
		Version:               instance2.version,
		BlobHash:              instance2.blobHash,
		PreviousVersionRecord: instance2.previousVersionRecord,
		Sig:                   instance2.sig,
	}

	data2, err := proto.Marshal(protoInstance2)
	if err != nil {
		t.Fatal(err)
	}

	instance2Hash := sha256.Sum256(data2)

	if err := reg.Kademlia.Store(data2); err != nil {
		t.Fatal(err)
	}

	instance3 := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "3.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: instance2Hash[:],
	}

	errInstance = instance3.sign(privateKey[domains[0]])
	if errInstance != nil {
		t.Fatal(errInstance)
	}

	resErr := reg.CompareVersionRecords(instance3, instance)

	if resErr != nil {
		t.Errorf("This should succeed as catch-up")
	}
}

func TestFork(t *testing.T) {

	dns := &mockDNS{lookupResult: make(map[string]rsa.PublicKey)}
	domains := []string{"example.com"}

	privateKey, err := addDomains(dns, domains)
	if err != nil {
		t.Fatal("Failed to add domains to DNS:", err)
	}

	node, kadErr := kademlia.NewKademlia(
		kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"),
		"127.0.0.1:8080",
	)

	if kadErr != nil {
		t.Fatal(kadErr)
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
		previousVersionRecord: make([]byte, sha256.Size),
	}

	if err := instance.sign(privateKey[domains[0]]); err != nil {
		t.Fatal(err)
	}

	protoInstance := &VersionRecordProto{
		Tag:                   instance.tag,
		DomainName:            instance.domainName,
		PackageName:           instance.packageName,
		Version:               instance.version,
		BlobHash:              instance.blobHash,
		PreviousVersionRecord: instance.previousVersionRecord,
		Sig:                   instance.sig,
	}

	data, err := proto.Marshal(protoInstance)
	if err != nil {
		t.Fatal(err)
	}

	instanceHash := sha256.Sum256(data)

	if err := reg.Kademlia.Store(data); err != nil {
		t.Fatal(err)
	}

	instance2 := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "2.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: instanceHash[:],
	}

	if err := instance2.sign(privateKey[domains[0]]); err != nil {
		t.Fatal(err)
	}

	protoInstance2 := &VersionRecordProto{
		Tag:                   instance2.tag,
		DomainName:            instance2.domainName,
		PackageName:           instance2.packageName,
		Version:               instance2.version,
		BlobHash:              instance2.blobHash,
		PreviousVersionRecord: instance2.previousVersionRecord,
		Sig:                   instance2.sig,
	}

	data2, err := proto.Marshal(protoInstance2)
	if err != nil {
		t.Fatal(err)
	}

	instance2Hash := sha256.Sum256(data2)

	if err := reg.Kademlia.Store(data2); err != nil {
		t.Fatal(err)
	}

	instance3 := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "3.0.0",
		blobHash:              []byte("test-blob-hash"),
		previousVersionRecord: instance2Hash[:],
	}

	if err := instance3.sign(privateKey[domains[0]]); err != nil {
		t.Fatal(err)
	}

	fork2 := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "2.5.0",
		blobHash:              []byte("fork-blob"),
		previousVersionRecord: instanceHash[:],
	}

	if err := fork2.sign(privateKey[domains[0]]); err != nil {
		t.Fatal(err)
	}

	protoFork2 := &VersionRecordProto{
		Tag:                   fork2.tag,
		DomainName:            fork2.domainName,
		PackageName:           fork2.packageName,
		Version:               fork2.version,
		BlobHash:              fork2.blobHash,
		PreviousVersionRecord: fork2.previousVersionRecord,
		Sig:                   fork2.sig,
	}

	forkData2, err := proto.Marshal(protoFork2)
	if err != nil {
		t.Fatal(err)
	}

	fork2Hash := sha256.Sum256(forkData2)

	if err := reg.Kademlia.Store(forkData2); err != nil {
		t.Fatal(err)
	}

	fork3 := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "4.0.0",
		blobHash:              []byte("fork-blob"),
		previousVersionRecord: fork2Hash[:],
	}

	if err := fork3.sign(privateKey[domains[0]]); err != nil {
		t.Fatal(err)
	}

	resErr := reg.CompareVersionRecords(fork3, instance3)

	if resErr == nil {
		t.Error("This should fail because the incoming history is a fork")
	}
}
