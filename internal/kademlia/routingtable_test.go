package kademlia

import (
	"fmt"
	reflect "reflect"
	"sync"
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

// Tests that yoink returns correct
func TestYoink(t *testing.T) {
	me := NewContact(
		NewKademliaID(
			"1000000000000000000000000000000000000000000000000000000000000000",
		),
		"localhost:8000",
	)

	routingTable := NewRoutingTable(me)

	first := NewContact(
		NewKademliaID("8000000000000000000000000000000000000000000000000000000000000000"),
		"localhost:8001",
	)
	second := NewContact(
		NewKademliaID("4000000000000000000000000000000000000000000000000000000000000000"),
		"localhost:8002",
	)
	third := NewContact(
		NewKademliaID("C0000000000000000000000000000000000000000000000000000000000000"),
		"localhost:8003",
	)

	routingTable.AddContact(first)
	routingTable.AddContact(second)
	routingTable.AddContact(third)
	result := routingTable.YoinkContacts()

	if len(result) != IDLength*8 {
		t.Fatalf("expected %d contacts, got %d", IDLength*8, len(result))
	}

	expected := map[int][]Contact{
		routingTable.getBucketIndex(first.ID): {
			third, // inserted last, so it is first
			first,
		},
		routingTable.getBucketIndex(second.ID): {
			second,
		},
	}
	for bucketIndex, expectedContacts := range expected {
		if !reflect.DeepEqual(result[bucketIndex], expectedContacts) {
			t.Errorf(
				"bucket %d: expected %#v, got %#v",
				bucketIndex,
				expectedContacts,
				result[bucketIndex],
			)
		}
	}

	for bucketIndex, contacts := range result {
		if _, expectedBucket := expected[bucketIndex]; !expectedBucket && len(contacts) != 0 {
			t.Errorf("unexpected contacts in bucket %d: %#v", bucketIndex, contacts)
		}
	}

}

// Tests that calls yoinkC while contacts are added same time
func TestYoinkRace(t *testing.T) {
	routingTable := NewRoutingTable(NewContact(
		NewKademliaID("1000000000000000000000000000000000000000000000000000000000000000"),
		"localhost:8000",
	))

	var waitGroup sync.WaitGroup
	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		for i := 0; i < 1000; i++ {
			routingTable.AddContact(NewContact(
				NewKademliaID(fmt.Sprintf("%064x", i)),
				fmt.Sprintf("localhost:%d", 8001+i),
			))
		}
	}()

	go func() {
		defer waitGroup.Done()
		for i := 0; i < 1000; i++ {
			routingTable.YoinkContacts()
		}
	}()

	waitGroup.Wait()
}
