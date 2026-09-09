package kademlia

import (
	"context"
	"errors"
)

const (
	alpha = 3 // Number rof parallel queries
	k     = bucketSize
)

type Kademlia struct {
	Contact      Contact
	RoutingTable *RoutingTable
	Network      *Network
	Data         map[string][]byte
}

func (k *Kademlia) LookupContact(ctx context.Context, target *Contact) ([]Contact, error) {
	if k == nil || k.RoutingTable == nil || k.Network == nil ||
		target == nil || target.ID == nil {
		return nil, errors.New("invalid lookup arguments")
	}

	candidates := k.RoutingTable.FindClosestContacts(target.ID, bucketSize)
	queried := make(map[string]bool)

	for {
		batch := nextUnqueried(candidates, queried, alpha)
		// Stops when every candidate has been queried
		if len(batch) == 0 {
			break
		}

		results, err := queryBatch(k.Network, batch, target.ID)
		// Stop if network query fails
		if err != nil {
			return nil, err
		}

		// Continues until all candidates have been queried
		for _, result := range results {
			for _, contact := range result {
				if contact.ID == nil {
					continue
				}

				candidates = mergeClosest(candidates, contact, target.ID, bucketSize)
				k.RoutingTable.AddContact(contact)
			}
		}
	}

	return candidates, nil
}

func (kademlia *Kademlia) LookupData(hash string) {
	// TODO
}

func (kademlia *Kademlia) Store(data []byte) {
	// TODO
}

// HELPER FUNCTIONS
// ------------------
// returns the next batch of unqueried contacts from the candidates list, up to the specified alpha value.
// It also marks the contacts as queried in the provided map.
func nextUnqueried(candidates []Contact, queried map[string]bool, alpha int) []Contact {
	var batch []Contact
	for _, candidate := range candidates {
		if !queried[candidate.ID.String()] {
			batch = append(batch, candidate)
			queried[candidate.ID.String()] = true
			if len(batch) >= alpha {
				break
			}
		}
	}
	return batch
}

// sends a FindNode request to each contact in the batch concurrently and collects the results.
func queryBatch(network *Network, batch []Contact, targetID *KademliaID) ([][]Contact, error) {
	results := make([][]Contact, len(batch))
	errCh := make(chan error, len(batch))
	resultCh := make(chan struct {
		index    int
		contacts []Contact
	}, len(batch))

	for i, contact := range batch {
		go func(i int, contact Contact) {
			contacts, err := network.SendFindContactMessage(&contact, targetID)
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- struct {
				index    int
				contacts []Contact
			}{i, contacts}
		}(i, contact)
	}

	for i := 0; i < len(batch); i++ {
		select {
		case err := <-errCh:
			return nil, err
		case result := <-resultCh:
			results[result.index] = result.contacts
		}
	}

	return results, nil
}

// merges a new contact into the candidates list, keeping only the closest 'count' contacts to the targetID.
func mergeClosest(
	candidates []Contact,
	newContact Contact,
	targetID *KademliaID,
	count int,
) []Contact {
	if newContact.ID == nil {
		return candidates
	}

	newContact.CalcDistance(targetID)

	for _, candidate := range candidates {
		if candidate.ID.Equals(newContact.ID) {
			return candidates
		}
	}

	candidates = append(candidates, newContact)

	sorted := ContactCandidates{contacts: candidates}
	sorted.Sort()

	if sorted.Len() > count {
		return sorted.contacts[:count]
	}

	return sorted.contacts
}
