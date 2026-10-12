package kademlia

import (
	"errors"
	"testing"
)

const idPrefix = "00000000000000000000000000000000000000000000000000000000000000"

func idN(n string) *KademliaID { return NewKademliaID(idPrefix + n) }

func tableHas(rt *RoutingTable, id *KademliaID) bool {
	for _, bucket := range rt.YoinkContacts() {
		for _, c := range bucket {
			if c.ID.Equals(id) {
				return true
			}
		}
	}
	return false
}

func TestFullBucketReturnsOldestAndRemoveWorks(t *testing.T) {
	bucket := NewBucket()
	first := NewContact(NewKademliaID("00000000000000000000000000000000000000000000000000000000000000a0"), "first")
	bucket.AddContact(first)
	for i := 1; i < bucketSize; i++ {
		bucket.AddContact(NewContact(NewKademliaID("00000000000000000000000000000000000000000000000000000000000001"+string(rune('0'+i/10))+string(rune('0'+i%10))), "n"))
	}

	oldest := bucket.AddContact(NewContact(NewKademliaID("0000000000000000000000000000000000000000000000000000000000000fff"), "new"))
	if oldest == nil || !oldest.ID.Equals(first.ID) {
		t.Fatalf("expected the least recently seen contact to be returned, got %v", oldest)
	}
	if bucket.Len() != bucketSize {
		t.Fatalf("full bucket must not grow, got %d", bucket.Len())
	}

	bucket.Remove(first.ID)
	if bucket.Len() != bucketSize-1 {
		t.Fatalf("expected remove to shrink the bucket, got %d", bucket.Len())
	}
}

func TestUpdateContactAndRemoveContact(t *testing.T) {
	me := NewContact(idN("01"), "me")
	other := NewContact(idN("02"), "other")
	k := &Kademlia{Contact: me, RoutingTable: NewRoutingTable(me)}
	rt := k.RoutingTable

	k.updateContact(other)
	if !tableHas(rt, other.ID) {
		t.Fatal("expected contact to be added")
	}
	rt.RemoveContact(other)
	if tableHas(rt, other.ID) {
		t.Fatal("expected contact to be removed")
	}
}

func TestEvictIfDeadReplacesDeadOldest(t *testing.T) {
	mock := NewMockNetwork()
	k1 := lookupTestNode(t, mock, "0000000000000000000000000000000000000000000000000000000000000001", "node1", make(map[string][]byte))
	ghost := NewContact(idN("02"), "ghost") // no listener: pings fail
	newcomer := NewContact(idN("03"), "newcomer")
	k1.RoutingTable.AddContact(ghost)

	k1.evictIfDead(ghost, newcomer)

	if tableHas(k1.RoutingTable, ghost.ID) || !tableHas(k1.RoutingTable, newcomer.ID) {
		t.Fatal("expected dead oldest to be replaced by the newcomer")
	}
}

func TestEvictIfDeadKeepsLiveOldest(t *testing.T) {
	mock := NewMockNetwork()
	k1 := lookupTestNode(t, mock, "0000000000000000000000000000000000000000000000000000000000000001", "node1", make(map[string][]byte))
	k2 := lookupTestNode(t, mock, "0000000000000000000000000000000000000000000000000000000000000002", "node2", make(map[string][]byte))
	newcomer := NewContact(idN("03"), "newcomer")
	k1.RoutingTable.AddContact(k2.Contact)

	k1.evictIfDead(k2.Contact, newcomer)

	if !tableHas(k1.RoutingTable, k2.Contact.ID) || tableHas(k1.RoutingTable, newcomer.ID) {
		t.Fatal("expected live oldest to stay and the newcomer to be dropped")
	}
}

func TestPingFailureRemovesContact(t *testing.T) {
	mock := NewMockNetwork()
	k1 := lookupTestNode(t, mock, "0000000000000000000000000000000000000000000000000000000000000001", "node1", make(map[string][]byte))
	ghost := NewContact(idN("02"), "ghost")
	k1.RoutingTable.AddContact(ghost)

	if err := k1.Ping(&ghost); err == nil {
		t.Fatal("expected ping to a dead node to fail")
	}
	if tableHas(k1.RoutingTable, ghost.ID) {
		t.Fatal("expected failed contact to be removed")
	}
}

func TestDropFailedContactUsesQueryError(t *testing.T) {
	me := NewContact(idN("01"), "me")
	dead := NewContact(idN("02"), "dead")
	k := &Kademlia{Contact: me, RoutingTable: NewRoutingTable(me)}
	k.RoutingTable.AddContact(dead)

	k.dropFailedContact(&QueryError{Contact: dead, Err: errors.New("timeout")})

	if tableHas(k.RoutingTable, dead.ID) {
		t.Fatal("expected contact from QueryError to be removed")
	}
}
