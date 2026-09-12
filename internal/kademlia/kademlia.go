package kademlia

import (
	"errors"
)

const (
	alpha = 1 // Number of parallel queries
	k     = bucketSize
)

type Kademlia struct {
	Contact      Contact
	RoutingTable *RoutingTable
	Network      *Network
	Data         map[string][]byte
}

func (kademlia *Kademlia) LookupContact(target *Contact) ([]Contact, error) {
	if kademlia == nil || kademlia.RoutingTable == nil || kademlia.Network == nil ||
		target == nil || target.ID == nil {
		return nil, errors.New("invalid lookup arguments")
	}

	closest := kademlia.RoutingTable.FindClosestContacts(target.ID, k)
	candidates := &ContactCandidates{}
	candidates.Append(closest)
	queried := make(map[string]bool)

	for {
		batch := nextUnqueried(candidates, queried, alpha)
		// Stops when every candidate has been queried
		if len(batch) == 0 {
			break
		}

		results, err := queryBatch(kademlia.Network, batch, target.ID)
		// Stop if network query fails
		if err != nil {
			return nil, err
		}

		// Continues until all candidates have been queried
		for i, result := range results {
			kademlia.RoutingTable.AddContact(batch[i])
			for _, contact := range result {
				if contact.ID == nil {
					continue
				}

				mergeClosest(candidates, contact, target.ID, k)
			}
		}
	}

	return candidates.contacts, nil
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
func nextUnqueried(candidates *ContactCandidates, queried map[string]bool, alpha int) []Contact {
	var batch []Contact
	for _, candidate := range candidates.contacts {
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
	candidates *ContactCandidates,
	newContact Contact,
	targetID *KademliaID,
	count int,
) {
	if newContact.ID == nil {
		return
	}

	newContact.CalcDistance(targetID)

	for _, candidate := range candidates.contacts {
		if candidate.ID.Equals(newContact.ID) {
			return
		}
	}

	candidates.contacts = append(candidates.contacts, newContact)

	candidates.Sort()

	if candidates.Len() > count {
		candidates.contacts = candidates.contacts[:count]
	}
}
