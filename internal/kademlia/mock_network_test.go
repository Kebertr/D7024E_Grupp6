package kademlia

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

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

	kademlia1.Network.ServerListen(kademlia1)
	kademlia2.Network.ServerListen(kademlia2)

	result, err := network1.SendFindContactMessage(&node2, node4.ID)
	if err != nil {
		t.Error(err)
	}

	for _, contact := range result {
		if !(contact.ID.Equals(node3.ID) || contact.ID.Equals(node4.ID)) {
			t.Error("LookupContact returned unexpected contact")
		}
	}

	kademlia1.Network.listener.Close()
	kademlia2.Network.listener.Close()
}

func Test1000Nodes(t *testing.T) {
	mock := NewMockNetwork()

	nodes := make([]*Kademlia, 1000)
	randlatency := rand.Intn(len(nodes)) / 100

	mock.latency = float64(time.Duration(randlatency) * time.Millisecond)
	mock.packet_loss = 0.001

	for i := 0; i < len(nodes); i++ {
		node := NewContact(NewRandomKademliaID(), strconv.Itoa(i))

		network, err := initNetwork(mock, node)
		if err != nil {
			t.Error(err)
		}

		nodes[i] = &Kademlia{
			Contact:      node,
			RoutingTable: NewRoutingTable(node),
			Network:      network,
			Data:         make(map[string][]byte),
		}

		network.ServerListen(nodes[i])

	}

	success := 0
	failures := 0

	for i := 0; i < len(nodes); i++ {
		randValue := rand.Intn(len(nodes))
		result := nodes[i].Ping(&nodes[randValue].Contact)

		if result != nil {
			failures++
			continue
		}

		success++
	}

	if success+failures != len(nodes) {
		t.Fatalf("They should cover every case")
	}

	successrate := float64(success) / float64(len(nodes))
	failurerate := float64(failures) / float64(len(nodes))

	t.Log("The success rate was", successrate*100, "failureate", failurerate*100)
}
func TestSendFindDataMessageOutcomes(t *testing.T) {
	tests := []struct {
		name      string
		stored    bool
		wantFound bool
	}{
		{name: "empty value is found", stored: true, wantFound: true},
		{name: "missing value returns contacts", wantFound: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock := NewMockNetwork()
			node1 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1")
			node2 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "node2")
			node3 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000003"), "node3")

			network1, err := initNetwork(mock, node1)
			if err != nil {
				t.Fatal(err)
			}
			network2, err := initNetwork(mock, node2)
			if err != nil {
				t.Fatal(err)
			}

			target := NewKademliaID("0000000000000000000000000000000000000000000000000000000000000004")
			data := make(map[string][]byte)
			if test.stored {
				data[target.String()] = []byte{}
			}
			kademlia1 := &Kademlia{Contact: node1, RoutingTable: NewRoutingTable(node1), Network: network1}
			kademlia2 := &Kademlia{Contact: node2, RoutingTable: NewRoutingTable(node2), Network: network2, Data: data}
			kademlia2.RoutingTable.AddContact(node3)
			network1.ServerListen(kademlia1)
			network2.ServerListen(kademlia2)

			value, contacts, found, err := network1.SendFindDataMessage(&node2, target)
			if err != nil {
				t.Fatalf("SendFindDataMessage failed: %v", err)
			}
			if found != test.wantFound {
				t.Fatalf("expected found=%t, got %t", test.wantFound, found)
			}
			if test.wantFound && len(value) != 0 {
				t.Fatalf("expected empty value, got %q", value)
			}
			if !test.wantFound && (len(contacts) != 1 || !contacts[0].ID.Equals(node3.ID)) {
				t.Fatalf("expected node3 as a closer contact, got %v", contacts)
			}
			kademlia1.Network.listener.Close()
			kademlia2.Network.listener.Close()
		})

	}
}

func TestSendStoreMessage(t *testing.T) {
	mock := NewMockNetwork()

	node1 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1")
	node2 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "node2")

	network1, err := initNetwork(mock, node1)
	if err != nil {
		t.Fatal(err)
	}

	network2, err := initNetwork(mock, node2)
	if err != nil {
		t.Fatal(err)
	}

	kademlia1 := &Kademlia{
		Contact:      node1,
		RoutingTable: NewRoutingTable(node1),
		Network:      network1,
	}
	kademlia2 := &Kademlia{
		Contact:      node2,
		RoutingTable: NewRoutingTable(node2),
		Network:      network2,
	}
	network1.ServerListen(kademlia1)
	network2.ServerListen(kademlia2)

	data := []byte("value")
	target := NewValueID(data)

	if err := network1.SendStoreMessage(&node2, target, data); err != nil {
		t.Fatalf("SendStoreMessage failed: %v", err)
	}

	stored, ok := kademlia2.Data[target.String()]
	if !ok {
		t.Fatal("expected receiver to store the value")
	}
	if string(stored) != string(data) {
		t.Fatalf("expected stored value %q, got %q", data, stored)
	}

	kademlia1.Network.listener.Close()
	kademlia2.Network.listener.Close()
}

func TestSendStoreMessageRejectsMismatchedTarget(t *testing.T) {
	mock := NewMockNetwork()
	node1 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"), "node1")
	node2 := NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"), "node2")

	network1, err := initNetwork(mock, node1)
	if err != nil {
		t.Fatal(err)
	}
	network2, err := initNetwork(mock, node2)
	if err != nil {
		t.Fatal(err)
	}

	kademlia1 := &Kademlia{Contact: node1, RoutingTable: NewRoutingTable(node1), Network: network1}
	kademlia2 := &Kademlia{Contact: node2, RoutingTable: NewRoutingTable(node2), Network: network2}
	network1.ServerListen(kademlia1)
	network2.ServerListen(kademlia2)

	wrongTarget := NewKademliaID("0000000000000000000000000000000000000000000000000000000000000003")
	if err := network1.SendStoreMessage(&node2, wrongTarget, []byte("value")); err == nil {
		t.Fatal("expected the receiver to reject a target that does not hash from the value")
	}

	kademlia1.Network.listener.Close()
	kademlia2.Network.listener.Close()
}

func TestSendStoreMessageRejectsOversizedValue(t *testing.T) {
	contact := NewContact(
		NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"),
		"node2",
	)
	data := make([]byte, MaxValueSize+1)

	err := (&Network{}).SendStoreMessage(&contact, NewValueID(data), data)
	if err == nil {
		t.Fatal("expected oversized value to be rejected before sending")
	}
}

func TestHandleStoreRejectsOversizedValue(t *testing.T) {
	mock := NewMockNetwork()
	sender := NewContact(
		NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"),
		"sender",
	)
	receiver := NewContact(
		NewKademliaID("0000000000000000000000000000000000000000000000000000000000000002"),
		"receiver",
	)
	senderNetwork, err := initNetwork(mock, sender)
	if err != nil {
		t.Fatal(err)
	}
	receiverNetwork, err := initNetwork(mock, receiver)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, MaxValueSize+1)
	kademlia := &Kademlia{
		Contact:      receiver,
		RoutingTable: NewRoutingTable(receiver),
		Network:      receiverNetwork,
	}

	if err := kademlia.handleStore(Message{
		MessageId: createMessageId(),
		From:      sender,
		Target:    NewValueID(data),
		Value:     data,
	}); err != nil {
		t.Fatalf("handleStore failed to reject oversized value: %v", err)
	}
	response, err := senderNetwork.listener.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if response.Type != "STORE_ERROR" {
		t.Fatalf("expected STORE_ERROR, got %q", response.Type)
	}
	if len(kademlia.Data) != 0 {
		t.Fatal("receiver stored the oversized value")
	}
}
