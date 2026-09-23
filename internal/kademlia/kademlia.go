package kademlia

import (
	"crypto/sha256"
	"errors"
	sync "sync"
)

const (
	alpha         = 3 // Number of parallel queries
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

func (kademlia *Kademlia) LookupData(hash string) ([]byte, error) {
	if kademlia == nil || kademlia.RoutingTable == nil || kademlia.Network == nil {
		return nil, errors.New("invalid lookup arguments")
	}

	targetID := NewKademliaID(hash)
	if targetID == nil {
		return nil, errors.New("invalid data hash")
	}
	if kademlia.Data != nil {
		if value, ok := kademlia.Data[targetID.String()]; ok {
			return append([]byte(nil), value...), nil
		}
	}

	candidates := &ContactCandidates{}
	candidates.Append(kademlia.RoutingTable.FindClosestContacts(targetID, shortListSize))
	queried := make(map[string]bool)

	for {
		batch := NextUnqueried(candidates, queried, alpha)
		if len(batch) == 0 {
			return nil, errors.New("data not found")
		}

		results, err := QueryDataBatch(kademlia.Network, batch, targetID)
		if err != nil {
			return nil, err
		}
		for _, result := range results {
			if result.found {
				return result.value, nil
			}
			for _, contact := range result.contacts {
				MergeClosest(candidates, contact, targetID, shortListSize)
			}
		}
	}
}

func (kademlia *Kademlia) Store(data []byte) {
	if kademlia == nil || kademlia.RoutingTable == nil || kademlia.Network == nil {
		return
	}

	targetID := hashData(data)
	target := Contact{ID: targetID}
	contacts, err := kademlia.LookupContact(&target)
	if err != nil {
		return
	}

	// The node performing the lookup can itself be one of the k closest nodes.
	candidates := &ContactCandidates{}
	candidates.Append(contacts)
	MergeClosest(candidates, kademlia.Contact, targetID, shortListSize)

	for _, contact := range candidates.GetContacts(candidates.Len()) {
		if contact.ID.Equals(kademlia.Contact.ID) {
			if kademlia.Data == nil {
				kademlia.Data = make(map[string][]byte)
			}
			kademlia.Data[targetID.String()] = append([]byte(nil), data...)
			continue
		}
		_ = kademlia.Network.SendStoreMessage(&contact, targetID, data)
	}
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

// Takes <key,value> pair and makes 256 bit KadmeliaID
func hashData(data []byte) *KademliaID {
	hash := sha256.Sum256(data)
	targetID := KademliaID(hash)
	return &targetID
}

// sends a FindNode request to each contact in the batch concurrently and collects the results.
func QueryBatch(network *Network, batch []Contact, targetID *KademliaID) ([][]Contact, error) {
	results := make([][]Contact, len(batch))
	errCh := make(chan error, len(batch))
	resultCh := make(chan struct {
		index    int
		contacts []Contact
	}, len(batch))

	var wg sync.WaitGroup
	wg.Add(len(batch))
	for i, contact := range batch {
		go func(i int, contact Contact) {
			defer wg.Done()
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
	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
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

type dataQueryResult struct {
	value    []byte
	contacts []Contact
	found    bool
}

func QueryDataBatch(network *Network, batch []Contact, targetID *KademliaID) ([]dataQueryResult, error) {
	results := make([]dataQueryResult, len(batch))
	errCh := make(chan error, len(batch))
	resultCh := make(chan struct {
		index  int
		result dataQueryResult
	}, len(batch))

	var wg sync.WaitGroup
	wg.Add(len(batch))
	for i, contact := range batch {
		go func(i int, contact Contact) {
			defer wg.Done()
			value, contacts, found, err := network.SendFindDataMessage(&contact, targetID)
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- struct {
				index  int
				result dataQueryResult
			}{i, dataQueryResult{value: value, contacts: contacts, found: found}}
		}(i, contact)
	}
	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
	}

	for i := 0; i < len(batch); i++ {
		select {
		case err := <-errCh:
			return nil, err
		case result := <-resultCh:
			results[result.index] = result.result
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

	case "FIND_VALUE":
		return kademlia.FindReceiverData(msg)

	case "PING_RETURN":
		kademlia.Network.receive <- msg
		return nil

	case "FIND_NODE_RESPONSE":
		kademlia.Network.receive <- msg
		return nil

	case "FIND_VALUE_RESPONSE":
		kademlia.Network.receive <- msg
		return nil

	case "STORE":
		return kademlia.handleStore(msg)

	case "STORE_RESPONSE":
		kademlia.Network.receive <- msg
		return nil
	}
	return errors.New("No of those functions exists")
}

func (kademlia *Kademlia) FindReceiverData(msg Message) error {
	if msg.Target == nil {
		return errors.New("find value message has no target")
	}

	if value, ok := kademlia.Data[msg.Target.String()]; ok {
		message := Message{
			MessageId: msg.MessageId,
			From:      kademlia.Contact,
			To:        msg.From.Address,
			Type:      "FIND_VALUE_RESPONSE",
			Value:     append([]byte(nil), value...),
		}
		return kademlia.Network.listener.Send(message)
	}

	contacts := kademlia.RoutingTable.FindClosestContacts(msg.Target, shortListSize)
	message := Message{
		MessageId: msg.MessageId,
		From:      kademlia.Contact,
		To:        msg.From.Address,
		Type:      "FIND_NODE_RESPONSE",
		Contacts:  contacts,
	}
	return kademlia.Network.listener.Send(message)
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

func (kademlia *Kademlia) handleStore(msg Message) error {
	if msg.Target == nil {
		return errors.New("store message has no target")
	}
	if kademlia.Data == nil {
		kademlia.Data = make(map[string][]byte)
	}

	kademlia.Data[msg.Target.String()] = append([]byte(nil), msg.Value...)

	message := Message{
		MessageId: msg.MessageId,
		From:      kademlia.Contact,
		To:        msg.From.Address,
		Type:      "STORE_RESPONSE",
	}

	return kademlia.Network.listener.Send(message)
}
