package kademlia

import "testing"

func TestPingRealNetowrk(t *testing.T) {
	real := NewRealNetwork()

	node1 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "127.0.0.1:8080")
	node2 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "127.0.0.1:8081")

	routing1 := NewRoutingTable(node1)
	routing2 := NewRoutingTable(node2)

	network1, err := initNetwork(real, node1)
	if err != nil {
		t.Error(err)
	}

	network2, err := initNetwork(real, node2)
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

	kademlia1.Network.ServerListen(kademlia1)
	kademlia2.Network.ServerListen(kademlia2)

	result := kademlia1.Ping(&kademlia2.Contact)

	if result != nil {
		t.Fatalf("Ping failed")
	}

	kademlia1.Network.listener.Close()
	kademlia2.Network.listener.Close()

}

func TestLookupContactRealNetwork(t *testing.T) {
	mock := NewRealNetwork()

	node1 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "127.0.0.1:8080")
	node2 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "127.0.0.1:8081")
	node3 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000003"), "127.0.0.1:8082")
	node4 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000004"), "127.0.0.1:8083")

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
		if contact.ID.Equals(node4.ID) {
			worked = true
		}
	}
	if !worked {
		t.Error("It should have found node 4")
	}

	kademlia1.Network.listener.Close()
	kademlia2.Network.listener.Close()
	kademlia3.Network.listener.Close()
	kademlia4.Network.listener.Close()
}
