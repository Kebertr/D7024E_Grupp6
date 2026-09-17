package kademlia

import (
	"testing"
)

func TestNextUnqueried(t *testing.T) {
	node1 := NewContact(NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000001",
	), "node1")

	node2 := NewContact(NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000002",
	), "node2")

	node3 := NewContact(NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000003",
	), "node3")

	candidates := &ContactCandidates{}
	candidates.Append([]Contact{node1, node2, node3})

	queried := map[string]bool{}

	result := NextUnqueried(candidates, queried, 1)

	if len(result) != 1 {
		t.Fatalf("We expected length of 1 since alpha is 0")
	}

	if !result[0].ID.Equals(node1.ID) {
		t.Error("expected node1")
	}
}

func TestNextUnqueriedNoone(t *testing.T) {
	node1 := NewContact(NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000001",
	), "node1")

	node2 := NewContact(NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000002",
	), "node2")

	node3 := NewContact(NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000003",
	), "node3")

	candidates := &ContactCandidates{}
	candidates.Append([]Contact{node1, node2, node3})

	queried := map[string]bool{
		node1.ID.String(): true,
		node2.ID.String(): true,
		node3.ID.String(): true,
	}

	result := NextUnqueried(candidates, queried, 1)

	if len(result) != 0 {
		t.Fatalf("We expected length of 0 since all already are queried ")
	}
}

func TestLookupContact(t *testing.T) {
	mock := NewMockNetwork()

	node1 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1")
	node2 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "node2")
	node3 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000003"), "node3")
	node4 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000004"), "node4")

	routing1 := NewRoutingTable(node1)
	routing2 := NewRoutingTable(node2)
	routing3 := NewRoutingTable(node3)
	routing4 := NewRoutingTable(node4)

	routing1.AddContact(node2)
	routing2.AddContact(node3)
	routing3.AddContact(node4)

	network1, err := InitNetwork(mock, node1.Address)
	if err != nil {
		t.Error(err)
	}

	network2, err := InitNetwork(mock, node2.Address)
	if err != nil {
		t.Error(err)
	}

	network3, err := InitNetwork(mock, node3.Address)
	if err != nil {
		t.Error(err)
	}

	network4, err := InitNetwork(mock, node4.Address)
	if err != nil {
		t.Error(err)
	}

	kademlia1 := &Kademlia{
		Contact:      node1,
		RoutingTable: routing1,
		Network:      network1,
		Data:         make(map[string][]byte),
	}

	kademlia2 := &Kademlia{
		Contact:      node2,
		RoutingTable: routing2,
		Network:      network2,
		Data:         make(map[string][]byte),
	}

	kademlia3 := &Kademlia{
		Contact:      node3,
		RoutingTable: routing3,
		Network:      network3,
		Data:         make(map[string][]byte),
	}

	kademlia4 := &Kademlia{
		Contact:      node4,
		RoutingTable: routing4,
		Network:      network4,
		Data:         make(map[string][]byte),
	}

	kademlia1.Network.ServerListen(kademlia1)
	kademlia2.Network.ServerListen(kademlia2)
	kademlia3.Network.ServerListen(kademlia3)
	kademlia4.Network.ServerListen(kademlia4)

	result, err := kademlia1.LookupContact(&node4)
	if err != nil {
		t.Error(err)
	}

	worked := false

	for _, contact := range result {
		t.Log(contact)
		if contact.ID.Equals(node4.ID) {
			worked = true
			break
		}
	}
	if !worked {
		t.Error("It should have found node 4")
	}
}

func TestSevenNodes(t *testing.T) {
	transport := NewMockNetwork()
	contacts := []Contact{
		NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1"),
		NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "node2"),
		NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000003"), "node3"),
		NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000004"), "node4"),
		NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000005"), "node5"),
		NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000006"), "node6"),
		NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000007"), "node7"),
	}

	networks := make([]*Network, len(contacts))
	nodes := make([]*Kademlia, len(contacts))
	for index, contact := range contacts {
		network, err := InitNetwork(transport, contact.Address)
		if err != nil {
			t.Fatal(err)
		}
		networks[index] = network
		nodes[index] = &Kademlia{
			Contact:      contact,
			RoutingTable: NewRoutingTable(contact),
			Network:      network,
		}
	}

	nodes[0].RoutingTable.AddContact(contacts[1])
	nodes[1].RoutingTable.AddContact(contacts[2])
	nodes[2].RoutingTable.AddContact(contacts[3])
	nodes[3].RoutingTable.AddContact(contacts[1])
	nodes[3].RoutingTable.AddContact(contacts[4])
	nodes[4].RoutingTable.AddContact(contacts[5])
	nodes[5].RoutingTable.AddContact(contacts[6])

	for index, network := range networks {
		network.ServerListen(nodes[index])
	}

	result, err := nodes[0].LookupContact(&contacts[3])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Creates a map of expected contacts,
	// will be something like: ....0002:true, where left value is contact ID and right value is wether it is expected to be in the result or not.

	expected := make(map[string]bool)
	for _, contact := range contacts {
		if !contact.ID.Equals(contacts[0].ID) {
			expected[contact.ID.String()] = true
		}
	}

	if len(result) != len(expected) {
		t.Fatalf("expected %d contacts, got %d", len(expected), len(result))
	}

	for _, contact := range result {
		if !expected[contact.ID.String()] {
			t.Errorf("unexpected contact returned: %s", contact.ID)
		}
	}

	targetID := contacts[3].ID

	for index := 1; index < len(result); index++ {
		previousDistance := result[index-1].ID.CalcDistance(targetID)
		currentDistance := result[index].ID.CalcDistance(targetID)

		if currentDistance.Less(previousDistance) {
			t.Errorf("contacts are not sorted by XOR distance")
		}
	}
}

func TestInvalidLookupContact(t *testing.T) {
	me := NewContact(
		NewKademliaID("1000000000000000000000000000000000000000000000000000000000000000"),
		"localhost:8000",
	)
	node := Kademlia{
		Contact:      me,
		RoutingTable: NewRoutingTable(me),
		Network:      &Network{},
	}

	tests := []struct {
		name   string
		node   *Kademlia
		target *Contact
	}{
		{name: "nil node", node: nil, target: &me},
		{name: "nil routing table", node: &Kademlia{}, target: &me},
		{name: "nil target", node: &node, target: nil},
		{
			name:   "nil target ID",
			node:   &node,
			target: &Contact{Address: "localhost:9000"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, _ := test.node.LookupContact(test.target)
			if result != nil {
				t.Fatalf("expected nil result, got %v", result)
			}
		})
	}
}

func TestQueryBatch(t *testing.T) {
	transport := NewMockNetwork()
	network, err := InitNetwork(transport, "node1")
	if err != nil {
		t.Fatal(err)
	}

	unregistered := NewContact(
		NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"),
		"missing",
	)

	_, err = QueryBatch(
		network,
		[]Contact{unregistered},
		unregistered.ID,
	)

	if err == nil {
		t.Fatal("expected query error")
	}
}

func TestMergeClosest(t *testing.T) {
	targetID := NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000000",
	)

	t.Run("nil contact ID", func(t *testing.T) {
		candidates := &ContactCandidates{}

		MergeClosest(
			candidates,
			Contact{Address: "missing"},
			targetID,
			2,
		)

		if candidates.Len() != 0 {
			t.Fatalf("expected 0 candidates, got %d", candidates.Len())
		}
	})

	t.Run("duplicate contact", func(t *testing.T) {
		contact := NewContact(
			NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"),
			"node1",
		)

		candidates := &ContactCandidates{}
		candidates.Append([]Contact{contact})

		MergeClosest(candidates, contact, targetID, 2)

		if candidates.Len() != 1 {
			t.Fatalf("expected 1 candidate, got %d", candidates.Len())
		}
	})

	t.Run("shortlist short", func(t *testing.T) {
		contact1 := NewContact(
			NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"),
			"node1",
		)
		contact2 := NewContact(
			NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"),
			"node2",
		)
		contact3 := NewContact(
			NewKademliaID("0000000000000000000000000000000000000000000000000000000000000003"),
			"node3",
		)

		candidates := &ContactCandidates{}
		candidates.Append([]Contact{contact1, contact2})

		existing := candidates.GetContacts(2)
		for index := range existing {
			existing[index].CalcDistance(targetID)
		}

		MergeClosest(candidates, contact3, targetID, 2)

		if candidates.Len() != 2 {
			t.Fatalf("expected 2 candidates, got %d", candidates.Len())
		}

		result := candidates.GetContacts(2)
		if !result[0].ID.Equals(contact1.ID) ||
			!result[1].ID.Equals(contact2.ID) {
			t.Fatal("expected the two closest contacts to remain")
		}
	})
}
