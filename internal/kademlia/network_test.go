package kademlia

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	transport "github.com/RasmusKebert/D7024E_Grupp6/internal/network"
)

func TestProtocolPingAndFindNode(t *testing.T) {
	network := transport.NewMockNetwork()
	alice, err := NewKademlia(network, transport.Address{IP: "127.0.0.1", Port: 12100})
	if err != nil {
		t.Fatal(err)
	}
	defer alice.Close()
	bob, err := NewKademlia(network, transport.Address{IP: "127.0.0.1", Port: 12101})
	if err != nil {
		t.Fatal(err)
	}
	defer bob.Close()

	bobContact := bob.Contact()
	if err := alice.protocol.SendPingMessage(&bobContact); err != nil {
		t.Fatalf("send PING: %v", err)
	}

	waitForProtocol(t, func() bool {
		return len(bob.closestContacts(alice.ID(), 1)) == 1
	})

	if err := alice.protocol.SendFindContactMessage(&bobContact); err != nil {
		t.Fatalf("send FIND_NODE: %v", err)
	}
	waitForProtocol(t, func() bool {
		return len(alice.closestContacts(bob.ID(), 1)) == 1
	})
}

func TestProtocolStoreAndFindValue(t *testing.T) {
	network := transport.NewMockNetwork()
	publisher, err := NewKademlia(network, transport.Address{IP: "127.0.0.1", Port: 12102})
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	receiver, err := NewKademlia(network, transport.Address{IP: "127.0.0.1", Port: 12103})
	if err != nil {
		t.Fatal(err)
	}
	defer receiver.Close()

	value := []byte{0, 1, 2, 255, 3}
	sum := sha256.Sum256(value)
	key := hex.EncodeToString(sum[:])
	receiverContact := receiver.Contact()

	if err := publisher.protocol.SendStoreMessage(&receiverContact, key, value); err != nil {
		t.Fatalf("send STORE: %v", err)
	}
	waitForProtocol(t, func() bool {
		stored := receiver.StoredValues()
		got, ok := stored[key]
		return ok && string(got) == string(value)
	})

	if err := publisher.protocol.SendStoreMessage(&receiverContact, key, []byte("wrong")); err == nil {
		t.Fatal("expected invalid key/value pair to be rejected")
	}
	if err := publisher.protocol.SendFindDataMessage(&receiverContact, key); err != nil {
		t.Fatalf("send FIND_VALUE: %v", err)
	}
}

func waitForProtocol(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("protocol condition was not met before timeout")
}
