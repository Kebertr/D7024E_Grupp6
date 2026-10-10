package package_registry

import (
	"crypto/rsa"
	"crypto/sha256"
	"testing"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
)

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
