package kademlia

import (
	"errors"
)

const (
	alpha         = 1 // Number of parallel queries
	shortListSize = bucketSize
)

type Kademlia struct {
	Contact      Contact
	RoutingTable *RoutingTable
	Network      *Network
	Data         map[string][]byte
}

func (kademlia *Kademlia) Ping(contact *Contact) error {
	err := kademlia.Network.SendPingMessage(contact)

	if err != nil {
		return err
	}

	kademlia.RoutingTable.AddContact(*contact)
	return nil
}

func (kademlia *Kademlia) LookupContact(target *Contact) ([]Contact, error) {
	if kademlia == nil || kademlia.RoutingTable == nil || kademlia.Network == nil ||
		target == nil || target.ID == nil {
		return nil, errors.New("invalid lookup arguments")
	}

	closest := kademlia.RoutingTable.FindClosestContacts(target.ID, shortListSize)
	candidates := &ContactCandidates{}
	candidates.Append(closest)
	queried := make(map[string]bool)

	// Now compares the shortlist IDs to see if they've improved
	// If improved, we continue the loop, otherwise we break and return the shortlist
	for {
		batch := NextUnqueried(candidates, queried, alpha)
		if len(batch) == 0 {
			break
		}
		before := make([]*KademliaID, len(candidates.contacts))
		for i, candidate := range candidates.contacts {
			before[i] = candidate.ID
		}

		results, err := QueryBatch(kademlia.Network, batch, target.ID)
		if err != nil {
			return nil, err
		}

		for i, result := range results {
			kademlia.RoutingTable.AddContact(batch[i])
			for _, contact := range result {
				MergeClosest(candidates, contact, target.ID, shortListSize)
			}
		}

		improved := len(before) != len(candidates.contacts)
		if !improved {
			for i, candidate := range candidates.contacts {
				if !candidate.ID.Equals(before[i]) {
					improved = true
					break
				}
			}
		}

		if !improved {
			break
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
func NextUnqueried(candidates *ContactCandidates, queried map[string]bool, alpha int) []Contact {
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
func QueryBatch(network *Network, batch []Contact, targetID *KademliaID) ([][]Contact, error) {
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
func MergeClosest(
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

// This get called in listenserver and gets the message. It will get the shortListSize closest contacts and then call for FindReceiverNodes
func (kademlia *Kademlia) FindReceiverNodes(msg Message) error {
	if msg.Target == nil {
		return errors.New("We need a target ID")
	}
	contacts := kademlia.RoutingTable.FindClosestContacts(msg.Target, shortListSize)

	return kademlia.Network.FindReceiverNodes(msg.MessageId, msg.From.Address, contacts)
}

func (kademlia *Kademlia) handleIncomingMessage(msg Message) error {
	switch msg.Type {
	case "PING":
		return kademlia.handlePing(msg)
		//Add case for err. Also add case for value and so on in the future
	case "FIND_NODE":
		return kademlia.FindReceiverNodes(msg)

	case "PING_RETURN":
		kademlia.Network.receive <- msg
		return nil

	case "FIND_NODE_RESPONSE":
		kademlia.Network.receive <- msg
		return nil

	}
	return errors.New("No of those functions exists")
}

func (kademlia *Kademlia) handlePing(msg Message) error {
	kademlia.RoutingTable.AddContact(msg.From)

	message := Message{
		MessageId: msg.MessageId,
		From:      kademlia.Contact,
		To:        msg.From.Address,
		Type:      "PING_RETURN",
	}

	return kademlia.Network.listener.Send(message)

}
