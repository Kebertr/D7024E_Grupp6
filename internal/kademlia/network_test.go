package kademlia

import "testing"

func TestSendFindPingWrongId(t *testing.T) {
	mock := NewMockNetwork()

	node1 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1")
	node2 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "node2")

	routing1 := NewRoutingTable(node1)

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

	kademlia1.Network.serverListen(kademlia1)

	go func() {
		_, err := network2.listener.Recv()
		if err != nil {
			t.Error(err)
			return
		}

		wrongID := createMessageId()

		network1.receive <- Message{
			MessageId: wrongID,
			From:      node2,
			To:        node1.Address,
			Type:      "PING_RETURN",
		}

	}()
	err = network1.SendPingMessage(&node2)
	if err == nil {
		t.Error("Expected a missmatch of id")
	}

	if err.Error() != "The Id do not match" {
		t.Fatalf("Not the error we expected")
	}
}

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

	result, err := network1.SendFindContactMessage(&node2, node4.ID)
	if err != nil {
		t.Error(err)
	}

	for _, contact := range result {
		if !(contact.ID.Equals(node3.ID) || contact.ID.Equals(node4.ID)) {
			t.Error("LookupContact returned unexpected contact")
		}
	}
}

func TestSendFindContactWrongId(t *testing.T) {
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

	kademlia1.Network.serverListen(kademlia1)

	go func() {
		request, err := network2.listener.Recv()
		if err != nil {
			t.Error(err)
			return
		}

		wrongID := createMessageId()

		if err := network2.FindReceiverNodes(wrongID, request.From.Address, nil); err != nil {
			t.Error(err)
		}
	}()

	_, err = network1.SendFindContactMessage(&node2, node4.ID)
	if err == nil {
		t.Error("Expected a missmatch of id")
	}

	if err.Error() != "The Id do not match" {
		t.Fatalf("Not the error we expected")
	}
}
