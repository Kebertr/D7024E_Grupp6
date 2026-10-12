package kademlia

import (
	stdList "container/list"
)

// bucket definition
// contains a List
type Bucket struct {
	List *stdList.List
}

// newBucket returns a new instance of a bucket
func NewBucket() *Bucket {
	bucket := &Bucket{}
	bucket.List = stdList.New()
	return bucket
}

// AddContact adds the Contact to the front of the bucket
// or moves it to the front of the bucket if it already existed.
// The front is the most recently seen contact and the back is the least recently seen.
// If the contact is new and the bucket is full, nothing is added and the least recently
// seen contact is returned so the caller can ping it and decide on eviction.
func (bucket *Bucket) AddContact(contact Contact) *Contact {
	var element *stdList.Element
	for e := bucket.List.Front(); e != nil; e = e.Next() {
		nodeID := e.Value.(Contact).ID

		if (contact).ID.Equals(nodeID) {
			element = e
		}
	}

	if element == nil {
		if bucket.List.Len() < bucketSize {
			bucket.List.PushFront(contact)
		} else {
			oldest := bucket.List.Back().Value.(Contact)
			return &oldest
		}
	} else {
		bucket.List.MoveToFront(element)
	}
	return nil
}

// Remove deletes the contact with the given ID from the bucket, if present.
func (bucket *Bucket) Remove(id *KademliaID) {
	for e := bucket.List.Front(); e != nil; e = e.Next() {
		if e.Value.(Contact).ID.Equals(id) {
			bucket.List.Remove(e)
			return
		}
	}
}

// GetContactAndCalcDistance returns an array of Contacts where
// the distance has already been calculated
func (bucket *Bucket) GetContactAndCalcDistance(target *KademliaID) []Contact {
	var contacts []Contact

	for elt := bucket.List.Front(); elt != nil; elt = elt.Next() {
		contact := elt.Value.(Contact)
		contact.CalcDistance(target)
		contacts = append(contacts, contact)
	}

	return contacts
}

// Len return the size of the bucket
func (bucket *Bucket) Len() int {
	return bucket.List.Len()
}
