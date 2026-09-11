package kademlia

import (
	"testing"

	transport "github.com/RasmusKebert/D7024E_Grupp6/internal/network"
)

func TestNewKademliaCreatesNodeFoundation(t *testing.T) {
	network := transport.NewMockNetwork()
	node, err := NewKademlia(network, transport.Address{IP: "127.0.0.1", Port: 12000})
	if err != nil {
		t.Fatalf("create Kademlia node: %v", err)
	}
	defer node.Close()

	if node.ID() == nil || node.ID().String() != node.Transport().ID {
		t.Fatalf("Kademlia ID does not match transport node ID")
	}
	if node.Contact().Address != "127.0.0.1:12000" {
		t.Fatalf("unexpected node address: %s", node.Contact().Address)
	}
	if node.RoutingTable() == nil {
		t.Fatal("expected routing table")
	}

	alpha, replicationFactor := node.LookupParameters()
	if alpha != defaultAlpha || replicationFactor != defaultK {
		t.Fatalf("unexpected defaults: alpha=%d k=%d", alpha, replicationFactor)
	}
}

func TestKademliaLookupParametersAreAdjustable(t *testing.T) {
	network := transport.NewMockNetwork()
	node, err := NewKademlia(network, transport.Address{IP: "127.0.0.1", Port: 12001})
	if err != nil {
		t.Fatalf("create Kademlia node: %v", err)
	}
	defer node.Close()

	if err := node.SetLookupParameters(5, 12); err != nil {
		t.Fatalf("set lookup parameters: %v", err)
	}
	alpha, replicationFactor := node.LookupParameters()
	if alpha != 5 || replicationFactor != 12 {
		t.Fatalf("unexpected parameters: alpha=%d k=%d", alpha, replicationFactor)
	}

	if err := node.SetLookupParameters(0, 10); err == nil {
		t.Fatal("expected invalid alpha to be rejected")
	}
	if err := node.SetLookupParameters(3, 0); err == nil {
		t.Fatal("expected invalid replication factor to be rejected")
	}
}
