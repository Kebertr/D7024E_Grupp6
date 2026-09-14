package kademlia_test

import (
	"context"
	"testing"

	kademlia "github.com/RasmusKebert/d7024e-tutorial/internal/kademlia"
)

// Tests create a Kademlia node with a routing table and network
// Adds three contacts to routing table
// Calls LookupContact(&target)
// Asserts that three contacts are returned and that they are returned in the expected xor distance order

func TestLookupContactReturnsClosestContacts(t *testing.T) {
	me := kademlia.NewContact(
		kademlia.NewKademliaID("1000000000000000000000000000000000000000000000000000000000000000"),
		"localhost:8000",
	)
	node := &kademlia.Kademlia{
		Contact:      me,
		RoutingTable: kademlia.NewRoutingTable(me),
		Network:      &kademlia.Network{},
	}
	target := kademlia.NewContact(
		kademlia.NewKademliaID("8000000000000000000000000000000000000000000000000000000000000000"),
		"localhost:9000",
	)
	expected := []kademlia.Contact{
		target,
		kademlia.NewContact(
			kademlia.NewKademliaID("8001000000000000000000000000000000000000000000000000000000000000"),
			"localhost:8001",
		),
		kademlia.NewContact(
			kademlia.NewKademliaID("8002000000000000000000000000000000000000000000000000000000000000"),
			"localhost:8002",
		),
	}

	for _, contact := range expected {
		node.RoutingTable.AddContact(contact)
	}

	result, err := node.LookupContact(context.Background(), &target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != len(expected) {
		t.Fatalf("expected %d contacts, got %d", len(expected), len(result))
	}

	for index, expectedContact := range expected {
		if !result[index].ID.Equals(expectedContact.ID) {
			t.Errorf(
				"contact %d: expected ID %s, got %s",
				index,
				expectedContact.ID,
				result[index].ID,
			)
		}
	}
}

func TestLookupContactReturnsNilForInvalidInput(t *testing.T) {
	me := kademlia.NewContact(
		kademlia.NewKademliaID("1000000000000000000000000000000000000000000000000000000000000000"),
		"localhost:8000",
	)
	node := &kademlia.Kademlia{
		Contact:      me,
		RoutingTable: kademlia.NewRoutingTable(me),
		Network:      &kademlia.Network{},
	}

	tests := []struct {
		name   string
		node   *kademlia.Kademlia
		target *kademlia.Contact
	}{
		{name: "nil node", node: nil, target: &me},
		{name: "nil routing table", node: &kademlia.Kademlia{}, target: &me},
		{name: "nil target", node: node, target: nil},
		{
			name:   "nil target ID",
			node:   node,
			target: &kademlia.Contact{Address: "localhost:9000"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, _ := test.node.LookupContact(context.Background(), test.target)
			if result != nil {
				t.Fatalf("expected nil result, got %v", result)
			}
		})
	}
}
