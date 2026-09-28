package kademlia

import (
	"fmt"
	"testing"
)

// FIXME: This test doesn't actually test anything. There is only one assertion
// that is included as an example.

func TestRoutingTable(t *testing.T) {
	rt := NewRoutingTable(NewContact(NewKademliaID("FFFFFFFF00000000000000000000000000000000000000000000000000000000"), "localhost:8000"))

	rt.AddContact(NewContact(NewKademliaID("FFFFFFFF00000000000000000000000000000000000000000000000000000000"), "localhost:8001"))
	rt.AddContact(NewContact(NewKademliaID("1111111100000000000000000000000000000000000000000000000000000000"), "localhost:8002"))
	rt.AddContact(NewContact(NewKademliaID("1111111200000000000000000000000000000000000000000000000000000000"), "localhost:8002"))
	rt.AddContact(NewContact(NewKademliaID("1111111300000000000000000000000000000000000000000000000000000000"), "localhost:8002"))
	rt.AddContact(NewContact(NewKademliaID("1111111400000000000000000000000000000000000000000000000000000000"), "localhost:8002"))
	rt.AddContact(NewContact(NewKademliaID("2111111400000000000000000000000000000000000000000000000000000000"), "localhost:8002"))

	contacts := rt.FindClosestContacts(NewKademliaID("2111111400000000000000000000000000000000000000000000000000000000"), 20)
	for i := range contacts {
		fmt.Println(contacts[i].String())
	}

	// TODO: This is just an example. Make more meaningful assertions.
	if len(contacts) != 6 {
		t.Fatalf("Expected 6 contacts but instead got %d", len(contacts))
	}
}

// Tests that findClosestContacts actually returns a valid shortList
func TestFindClosestContactsReturnsClosestContacts(t *testing.T) {
	me := NewContact(
		NewKademliaID(
			"1000000000000000000000000000000000000000000000000000000000000000",
		),
		"localhost:8000",
	)

	routingTable := NewRoutingTable(me)

	contacts := []Contact{
		NewContact(
			NewKademliaID(
				"8000000000000000000000000000000000000000000000000000000000000000",
			),
			"localhost:8001",
		),
		NewContact(
			NewKademliaID(
				"8001000000000000000000000000000000000000000000000000000000000000",
			),
			"localhost:8002",
		),
		NewContact(
			NewKademliaID(
				"9000000000000000000000000000000000000000000000000000000000000000",
			),
			"localhost:8003",
		),
	}

	for _, contact := range contacts {
		routingTable.AddContact(contact)
	}

	target := NewKademliaID(
		"8000000000000000000000000000000000000000000000000000000000000000",
	)

	result := routingTable.FindClosestContacts(target, 2)

	if len(result) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(result))
	}

	expectedIDs := []string{
		"8000000000000000000000000000000000000000000000000000000000000000",
		"8001000000000000000000000000000000000000000000000000000000000000",
	}

	for index, expectedID := range expectedIDs {
		if result[index].ID.String() != expectedID {
			t.Errorf(
				"contact %d: expected %s, got %s",
				index,
				expectedID,
				result[index].ID,
			)
		}
	}

	for index := 1; index < len(result); index++ {
		previousDistance := result[index-1].ID.CalcDistance(target)
		currentDistance := result[index].ID.CalcDistance(target)

		if currentDistance.Less(previousDistance) {
			t.Errorf("contacts are not sorted by XOR distance")
		}
	}
}
