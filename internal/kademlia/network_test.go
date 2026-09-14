package kademlia

import "testing"

func TestSendFindContactMessage(t *testing.T) {
	mock := NewMockNetwork()

	node1 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1")
	node2 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "node2")
	node3 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000003"), "node3")
	node4 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000004"), "node4")

	routing1 := NewRoutingTable(node1)
	routing2 := NewRoutingTable(node2)

	routing1.AddContact(node2)
	routing2.AddContact(node3)
	routing2.AddContact(node4)

	network1, err := InitNetwork(mock, node1.Address)
	if err != nil {
		t.Error(err)
	}

	network2, err := InitNetwork(mock, node2.Address)
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

	result, err := network1.SendFindContactMessage(&node2, node4.ID)
	if err != nil {
		t.Error(err)
	}

	for _, contact := range result {
		t.Log(contact)
		if !(contact.ID.Equals(node3.ID) || contact.ID.Equals(node4.ID)) {
			t.Error("LookupContact returned unexpected contact")
		}
	}
}
