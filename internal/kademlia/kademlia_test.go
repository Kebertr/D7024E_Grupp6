package kademlia

import "testing"

func TestPing(t *testing.T) {
	mock := NewMockNetwork()

	node1 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1")
	node2 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "node2")

	routing1 := NewRoutingTable(node1)
	routing2 := NewRoutingTable(node2)

	network1, err := initNetwork(mock, node1)
	if err != nil {
		t.Error(err)
	}

	network2, err := initNetwork(mock, node2)
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

	kademlia1.Network.serverListen(kademlia1)
	kademlia2.Network.serverListen(kademlia2)

	result := kademlia1.Ping(&kademlia2.Contact)

	if result != nil {
		t.Fatalf("Ping failed")
	}

}

func TestPingErrors(t *testing.T) {
	mock := NewMockNetwork()

	node1 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1")

	routing1 := NewRoutingTable(node1)

	network1, err := initNetwork(mock, node1)
	if err != nil {
		t.Error(err)
	}

	kademlia1 := &Kademlia{
		Contact:      node1,
		RoutingTable: routing1,
		Network:      network1,
		Data:         make(map[string][]byte),
	}

	result := kademlia1.Ping(nil)

	if result == nil {
		t.Fatalf("Expected result to be nil. Since there is no destination address")
	}
}

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

	result := nextUnqueried(candidates, queried, 1)

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

	result := nextUnqueried(candidates, queried, 1)

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

	network1, err := initNetwork(mock, node1)
	if err != nil {
		t.Error(err)
	}

	network2, err := initNetwork(mock, node2)
	if err != nil {
		t.Error(err)
	}

	network3, err := initNetwork(mock, node3)
	if err != nil {
		t.Error(err)
	}

	network4, err := initNetwork(mock, node4)
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

	kademlia1.Network.serverListen(kademlia1)
	kademlia2.Network.serverListen(kademlia2)
	kademlia3.Network.serverListen(kademlia3)
	kademlia4.Network.serverListen(kademlia4)

	result, err := kademlia1.LookupContact(&node4)
	if err != nil {
		t.Error(err)
	}

	worked := false

	for _, contact := range result {
		if contact.ID.Equals(node4.ID) {
			worked = true
		}
	}
	if !worked {
		t.Error("It should have found node 4")
	}
}
