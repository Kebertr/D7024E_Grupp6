package package_registry

import (
	"crypto/rsa"
	"crypto/sha256"
	"testing"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
)

func TestPublish(t *testing.T) {

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

	file := []byte("first-version-data")

	errPublish := reg.publish(domains[0], "package", "1.0.0", file, false, "", *privateKey[domains[0]])

	if errPublish != nil {
		t.Error(errPublish)
	}

	pointer, err := reg.getLatestPointer(domains[0], "package")

	if pointer.version != "1.0.0" {
		t.Error("It is the wrong version")
	}

}

func TestPublishforce(t *testing.T) {

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

	file := []byte("first-version-data")

	errPublish := reg.publish(domains[0], "package", "1.0.0", file, false, "", *privateKey[domains[0]])

	if errPublish != nil {
		t.Error(errPublish)
	}

	file1 := []byte("first-version-data2")

	errPublish1 := reg.publish(domains[0], "package", "1.0.0", file1, false, "", *privateKey[domains[0]])

	if errPublish1 == nil {
		t.Error("This should give error since force is falsed")
	}

	errPublish2 := reg.publish(domains[0], "package", "1.0.0", file1, true, "", *privateKey[domains[0]])

	if errPublish2 != nil {
		t.Error(errPublish2)
	}
}

func TestPublishManualPrev(t *testing.T) {

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

	file := []byte("first-version-data")

	errPublish := reg.publish(domains[0], "package", "1.0.0", file, false, "", *privateKey[domains[0]])

	if errPublish != nil {
		t.Error(errPublish)
	}

	file1 := []byte("first-version-data2")

	errPublish1 := reg.publish(domains[0], "package", "2.0.0", file1, false, "", *privateKey[domains[0]])

	if errPublish1 != nil {
		t.Error(errPublish1)
	}

	errPublish2 := reg.publish(domains[0], "package", "3.0.0", file1, false, "", *privateKey[domains[0]])

	if errPublish2 != nil {
		t.Error(errPublish2)
	}

	errPublish3 := reg.publish(domains[0], "package", "3.0.0", file1, false, "1.0.0", *privateKey[domains[0]])

	if errPublish2 != nil {
		t.Error(errPublish3)
	}
}
func TestInstall(t *testing.T) {

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

	blob := []byte("test-blob-hash")

	errStore := reg.Kademlia.Store(blob)

	if errStore != nil {
		t.Error(errStore)
	}

	blobHash := sha256.Sum256(blob)

	instance := &VersionRecord{
		tag:                   "test",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "1.0.0",
		blobHash:              blobHash[:],
		previousVersionRecord: []byte("test-previous-version-record"),
		sig:                   []byte("test-signature"),
	}

	setErr := reg.setLatestPointer(instance, privateKey[domains[0]])

	if setErr != nil {
		t.Error(setErr)
	}

	res, errInstall := reg.install(domains[0], instance.packageName, "latest")

	if errInstall != nil {
		t.Error(errInstall)
	}

	if string(res) != "test-blob-hash" {
		t.Error("This should be equal to what I wrote u there")
	}
}

func TestInstallSpecificVersion(t *testing.T) {

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

	blob := []byte("test-blob-hash")
	blob2 := []byte("test-blob-hash1")
	blob3 := []byte("test-blob-hash2")

	errStore := reg.Kademlia.Store(blob)

	if errStore != nil {
		t.Error(errStore)
	}
	errStore2 := reg.Kademlia.Store(blob2)

	if errStore2 != nil {
		t.Error(errStore2)
	}

	errstore3 := reg.Kademlia.Store(blob3)
	if errstore3 != nil {
		t.Error(errstore3)
	}

	blobHash1 := sha256.Sum256(blob)

	instance1 := &VersionRecord{
		tag:                   "version-record",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "1.0.0",
		blobHash:              blobHash1[:],
		previousVersionRecord: make([]byte, sha256.Size),
	}

	instance1.sign(privateKey[domains[0]])
	reg.setLatestPointer(instance1, privateKey[domains[0]])

	pointer1, _ := reg.getLatestPointer(domains[0], "test-package")

	blobHash2 := sha256.Sum256(blob2)

	instance2 := &VersionRecord{
		tag:                   "version-record",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "2.0.0",
		blobHash:              blobHash2[:],
		previousVersionRecord: pointer1.versionRecordHash,
	}

	instance2.sign(privateKey[domains[0]])
	reg.setLatestPointer(instance2, privateKey[domains[0]])

	pointer2, _ := reg.getLatestPointer(domains[0], "test-package")

	blobHash3 := sha256.Sum256(blob3)

	instance3 := &VersionRecord{
		tag:                   "version-record",
		domainName:            domains[0],
		packageName:           "test-package",
		version:               "3.0.0",
		blobHash:              blobHash3[:],
		previousVersionRecord: pointer2.versionRecordHash,
	}

	instance3.sign(privateKey[domains[0]])
	reg.setLatestPointer(instance3, privateKey[domains[0]])

	res, errInstall := reg.install(domains[0], instance3.packageName, "1.0.0")

	if errInstall != nil {
		t.Error(errInstall)
	}

	if string(res) != "test-blob-hash" {
		t.Error("This should be equal to what I wrote u there")
	}

	res2, errInstall2 := reg.install(domains[0], instance3.packageName, "2.0.0")

	if errInstall2 != nil {
		t.Error(errInstall2)
	}

	if string(res2) != "test-blob-hash1" {
		t.Error("This should be equal to what I wrote u there")
	}
}

func TestVersionPackage(t *testing.T) {

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

	version, errVersion := reg.showVersionPackage(domains[0], instance.packageName)

	if errVersion != nil {
		t.Error(errVersion)
	}

	if version != "1.0.0" {
		t.Error("This should be the version")
	}
}

func TestShowDNS(t *testing.T) {

	dns := &mockDNS{lookupResult: make(map[string]rsa.PublicKey)}
	domains := []string{"example.com"}

	_, err := addDomains(dns, domains)
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

	dnsShow, errdns := reg.showDNS(domains[0])

	if errdns != nil {
		t.Error(errdns)
	}

	dnsLook, errdnsLook := reg.DNS.lookup(domains[0])

	if errdnsLook != nil {
		t.Error(errdnsLook)
	}

	if dnsShow != dnsLook {
		t.Error("They should really be equal")
	}
}
