package kademlia

import "sync"

const bucketSize = 20
const BucketSize = bucketSize

// RoutingTable definition
// keeps a refrence contact of me and an array of buckets
type RoutingTable struct {
	me      Contact
	buckets [IDLength * 8]*Bucket
	muRoute sync.RWMutex
}

// NewRoutingTable returns a new instance of a RoutingTable
func NewRoutingTable(me Contact) *RoutingTable {
	routingTable := &RoutingTable{}
	for i := 0; i < IDLength*8; i++ {
		routingTable.buckets[i] = NewBucket()
	}
	routingTable.me = me
	return routingTable
}

// AddContact add a new contact to the correct Bucket.
// It returns the least recently seen contact if the bucket is full (see Bucket.AddContact),
// otherwise nil. The caller must ping it outside this lock (see Kademlia.updateContact).
func (routingTable *RoutingTable) AddContact(contact Contact) *Contact {
	// Nil check only: existing tests deliberately add arbitrary IDs directly to the table.
	// Self is filtered in Kademlia.updateContact instead.
	if contact.ID == nil {
		return nil
	}
	routingTable.muRoute.Lock()
	defer routingTable.muRoute.Unlock()
	bucketIndex := routingTable.getBucketIndex(contact.ID)
	bucket := routingTable.buckets[bucketIndex]
	return bucket.AddContact(contact)
}

// RemoveContact removes a contact (e.g. one that stopped responding) from its bucket.
func (routingTable *RoutingTable) RemoveContact(contact Contact) {
	if contact.ID == nil {
		return
	}
	routingTable.muRoute.Lock()
	defer routingTable.muRoute.Unlock()
	routingTable.buckets[routingTable.getBucketIndex(contact.ID)].Remove(contact.ID)
}

// FindClosestContacts finds the count closest Contacts to the target in the RoutingTable
func (routingTable *RoutingTable) FindClosestContacts(target *KademliaID, count int) []Contact {
	routingTable.muRoute.RLock()
	defer routingTable.muRoute.RUnlock()
	var candidates ContactCandidates
	bucketIndex := routingTable.getBucketIndex(target)
	bucket := routingTable.buckets[bucketIndex]

	candidates.Append(bucket.GetContactAndCalcDistance(target))

	for i := 1; (bucketIndex-i >= 0 || bucketIndex+i < IDLength*8) && candidates.Len() < count; i++ {
		if bucketIndex-i >= 0 {
			bucket = routingTable.buckets[bucketIndex-i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
		if bucketIndex+i < IDLength*8 {
			bucket = routingTable.buckets[bucketIndex+i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
	}

	candidates.Sort()

	if count > candidates.Len() {
		count = candidates.Len()
	}

	return candidates.GetContacts(count)
}

// getBucketIndex get the correct Bucket index for the KademliaID
func (routingTable *RoutingTable) getBucketIndex(id *KademliaID) int {
	distance := id.CalcDistance(routingTable.me.ID)
	for i := 0; i < IDLength; i++ {
		for j := 0; j < 8; j++ {
			if (distance[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}

	return IDLength*8 - 1
}

// yoink returns a snapshot of all routing-table buckets.
func (routingTable *RoutingTable) YoinkContacts() [][]Contact {
	// Read lock: AddContact/RemoveContact modify the bucket lists concurrently.
	routingTable.muRoute.RLock()
	defer routingTable.muRoute.RUnlock()
	buckets := make([][]Contact, len(routingTable.buckets))

	for index, bucket := range routingTable.buckets {
		element := bucket.List.Front()

		for element != nil {
			contact := element.Value.(Contact)
			buckets[index] = append(buckets[index], contact)

			element = element.Next()
		}
	}

	return buckets
}
