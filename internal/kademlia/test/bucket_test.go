package kademlia_test

import (
	"testing"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
)

func TestNewBucket(t *testing.T) {
	bucket := kademlia.NewBucket()

	if bucket.Len() != 0 {
		t.Fatalf("expected new bucket to have size 0, got %d", bucket.Len())
	}

	if bucket.Len() > kademlia.BucketSize {
		t.Fatalf("expected bucket size to be at most %d, got %d",
			kademlia.BucketSize, bucket.Len())
	}
}

func TestAddContact(t *testing.T) {
	bucket := kademlia.NewBucket()

	contact := kademlia.NewContact(
		kademlia.NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"),
		"node1",
	)

	bucket.AddContact(contact)

	if bucket.Len() != 1 {
		t.Fatalf("expected bucket to have size 1, got %d", bucket.Len())
	}

	addedContact := bucket.List.Front().Value.(kademlia.Contact)

	if !addedContact.ID.Equals(contact.ID) {
		t.Fatalf("expected contact ID to be %s, got %s",
			contact.ID.String(), addedContact.ID.String())
	}
}

// Cant test GetContactAndCalcDistance without making distance public
