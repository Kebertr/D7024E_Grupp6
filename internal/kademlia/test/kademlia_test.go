package kademlia_test

import (
	"testing"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
)

func TestNextUnqueried(t *testing.T) {
	node1 := kademlia.NewContact(kademlia.NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000001",
	), "node1")

	node2 := kademlia.NewContact(kademlia.NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000002",
	), "node2")

	node3 := kademlia.NewContact(kademlia.NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000003",
	), "node3")

	candidates := &kademlia.ContactCandidates{}
	candidates.Append([]kademlia.Contact{node1, node2, node3})

	queried := map[string]bool{}

	result := kademlia.NextUnqueried(candidates, queried, 1)

	if len(result) != 1 {
		t.Fatalf("We expected length of 1 since alpha is 0")
	}

	if !result[0].ID.Equals(node1.ID) {
		t.Error("expected node1")
	}
}

func TestNextUnqueriedNoone(t *testing.T) {
	node1 := kademlia.NewContact(kademlia.NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000001",
	), "node1")

	node2 := kademlia.NewContact(kademlia.NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000002",
	), "node2")

	node3 := kademlia.NewContact(kademlia.NewKademliaID(
		"0000000000000000000000000000000000000000000000000000000000000003",
	), "node3")

	candidates := &kademlia.ContactCandidates{}
	candidates.Append([]kademlia.Contact{node1, node2, node3})

	queried := map[string]bool{
		node1.ID.String(): true,
		node2.ID.String(): true,
		node3.ID.String(): true,
	}

	result := kademlia.NextUnqueried(candidates, queried, 1)

	if len(result) != 0 {
		t.Fatalf("We expected length of 0 since all already are queried ")
	}
}

func TestLookupContact(t *testing.T) {
	mock := kademlia.NewMockNetwork()

	node1 := kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1")
	node2 := kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "node2")
	node3 := kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000003"), "node3")
	node4 := kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000004"), "node4")

	routing1 := kademlia.NewRoutingTable(node1)
	routing2 := kademlia.NewRoutingTable(node2)
	routing3 := kademlia.NewRoutingTable(node3)
	routing4 := kademlia.NewRoutingTable(node4)

	routing1.AddContact(node2)
	routing2.AddContact(node3)
	routing3.AddContact(node4)

	network1, err := kademlia.InitNetwork(mock, node1.Address)
	if err != nil {
		t.Error(err)
	}

	network2, err := kademlia.InitNetwork(mock, node2.Address)
	if err != nil {
		t.Error(err)
	}

	network3, err := kademlia.InitNetwork(mock, node3.Address)
	if err != nil {
		t.Error(err)
	}

	network4, err := kademlia.InitNetwork(mock, node4.Address)
	if err != nil {
		t.Error(err)
	}

	kademlia1 := &kademlia.Kademlia{
		Contact:      node1,
		RoutingTable: routing1,
		Network:      network1,
		Data:         make(map[string][]byte),
	}

	kademlia2 := &kademlia.Kademlia{
		Contact:      node2,
		RoutingTable: routing2,
		Network:      network2,
		Data:         make(map[string][]byte),
	}

	kademlia3 := &kademlia.Kademlia{
		Contact:      node3,
		RoutingTable: routing3,
		Network:      network3,
		Data:         make(map[string][]byte),
	}

	kademlia4 := &kademlia.Kademlia{
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
	transport := kademlia.NewMockNetwork()
	contacts := []kademlia.Contact{
		kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1"),
		kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "node2"),
		kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000003"), "node3"),
		kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000004"), "node4"),
		kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000005"), "node5"),
		kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000006"), "node6"),
		kademlia.NewContact(kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000007"), "node7"),
	}

	networks := make([]*kademlia.Network, len(contacts))
	nodes := make([]*kademlia.Kademlia, len(contacts))
	for index, contact := range contacts {
		network, err := kademlia.InitNetwork(transport, contact.Address)
		if err != nil {
			t.Fatal(err)
		}
		networks[index] = network
		nodes[index] = &kademlia.Kademlia{
			Contact:      contact,
			RoutingTable: kademlia.NewRoutingTable(contact),
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
			result, _ := test.node.LookupContact(test.target)
			if result != nil {
				t.Fatalf("expected nil result, got %v", result)
			}
		})
	}
}
