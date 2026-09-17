package kademlia

import "testing"

func TestNewBucket(t *testing.T) {
	bucket := NewBucket()

	if bucket.Len() != 0 {
		t.Fatalf("expected new bucket to have size 0, got %d", bucket.Len())
	}

	if bucket.Len() > BucketSize {
		t.Fatalf("expected bucket size to be at most %d, got %d",
			BucketSize, bucket.Len())
	}
}

func TestAddContact(t *testing.T) {
	bucket := NewBucket()

	contact := NewContact(
		NewKademliaID("0000000000000000000000000000000000000000000000000000000000000001"),
		"node1",
	)

	bucket.AddContact(contact)

	if bucket.Len() != 1 {
		t.Fatalf("expected bucket to have size 1, got %d", bucket.Len())
	}

	addedContact := bucket.List.Front().Value.(Contact)

	if !addedContact.ID.Equals(contact.ID) {
		t.Fatalf("expected contact ID to be %s, got %s",
			contact.ID.String(), addedContact.ID.String())
	}
}

// Cant test GetContactAndCalcDistance without making distance public
