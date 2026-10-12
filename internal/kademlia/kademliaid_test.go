package kademlia

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestIPPortDerivedNodeID(t *testing.T) {
	ip := "127.0.0.1"
	port := 8000

	got := IPPortDerivedNodeID(ip, port)
	wantHash := sha256.Sum256([]byte("127.0.0.1|8000"))
	want := KademliaID(wantHash)
	if !got.Equals(&want) {
		t.Fatalf("expected ID %s, got %s", hex.EncodeToString(want[:]), got)
	}

	if again := IPPortDerivedNodeID(ip, port); !got.Equals(again) {
		t.Fatal("same IP and port should always produce the same node ID")
	}
	if IPPortDerivedNodeID(ip, port+1).Equals(got) {
		t.Fatal("changing the port should change the node ID")
	}
	if IPPortDerivedNodeID("127.0.0.2", port).Equals(got) {
		t.Fatal("changing the IP should change the node ID")
	}
}

func TestValidateNodeID(t *testing.T) {
	id := IPPortDerivedNodeID("127.0.0.1", 8000)

	if err := validateNodeID(id, "127.0.0.1:8000"); err != nil {
		t.Fatalf("expected the derived ID to be accepted: %v", err)
	}
	if err := validateNodeID(NewRandomKademliaID(), "127.0.0.1:8000"); err == nil {
		t.Fatal("expected a non-derived node ID to be rejected")
	}
	if err := validateNodeID(IPPortDerivedNodeID("127.0.0.1", 0), "127.0.0.1:0"); err == nil {
		t.Fatal("expected port zero to be rejected")
	}
	if err := validateNodeID(id, "localhost:8000"); err == nil {
		t.Fatal("expected a hostname instead of an IP address to be rejected")
	}
}
