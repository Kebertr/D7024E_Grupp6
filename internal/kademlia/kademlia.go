package kademlia

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
)

const (
	alpha         = 3 // Number of parallel queries
	shortListSize = bucketSize
	limitData     = 4096
)

type Kademlia struct {
	Contact      Contact
	RoutingTable *RoutingTable
	Network      *Network
	Data         map[string][]byte
	LookupStats  LookupStats
	muData       sync.RWMutex
}

// Allows atomic memory operations (count++)
// on shared variables that happens indivisibly
type LookupStats struct {
	totalProbes atomic.Int64
	totalHops   atomic.Int64
	successes   atomic.Int64
	failures    atomic.Int64
}

func (kademlia *Kademlia) Ping(contact *Contact) error {
	resp, err := kademlia.Network.SendPingMessage(contact)
	if err != nil {
		return err
	}

	kademlia.RoutingTable.AddContact(resp)

	return nil
}

func (kademlia *Kademlia) LookupContact(target *Contact) (contacts []Contact, err error) {
	defer func() {
		success := false
		if err == nil && target != nil && target.ID != nil {
			for _, contact := range contacts {
				if contact.ID != nil && contact.ID.Equals(target.ID) {
					success = true
					break
				}
			}
		}

		if kademlia != nil {
			kademlia.recordLookup(success)
		}
	}()

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
		kademlia.LookupStats.totalProbes.Add(int64(len(batch)))
		before := make([]*KademliaID, len(candidates.contacts))
		for i, candidate := range candidates.contacts {
			before[i] = candidate.ID
		}

		results, err := QueryBatch(kademlia.Network, batch, target.ID)
		if err != nil {
			return nil, err
		}
		for _, result := range results {
			if result != nil {
				kademlia.LookupStats.totalHops.Add(1)
				break
			}
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

func (kademlia *Kademlia) LookupData(hash string) ([]byte, Contact, error) {
	if kademlia == nil || kademlia.RoutingTable == nil || kademlia.Network == nil {
		return nil, kademlia.Contact, errors.New("invalid lookup arguments")
	}

	targetID := NewKademliaID(hash)
	if targetID == nil {
		return nil, kademlia.Contact, errors.New("invalid data hash")
	}

	kademlia.muData.RLock()
	value, ok := kademlia.Data[targetID.String()]
	kademlia.muData.RUnlock()
	if ok {
		// a client must check K = hash(V) before trusting a value.
		if NewValueID(value).Equals(targetID) {
			return value, kademlia.Contact, nil
		}
		// Discard the corrupt local value, report it, and fall back to the network lookup.
		log.Printf("error: local value for key %s does not match its hash, discarding", hash)
	}

	candidates := &ContactCandidates{}
	candidates.Append(kademlia.RoutingTable.FindClosestContacts(targetID, shortListSize))
	queried := make(map[string]bool)

	for {
		batch := NextUnqueried(candidates, queried, alpha)
		if len(batch) == 0 {
			return nil, kademlia.Contact, errors.New("data not found")
		}

		results, err := QueryDataBatch(kademlia.Network, batch, targetID)
		if err != nil {
			return nil, kademlia.Contact, err
		}
		for i, result := range results {
			if result.found {
				// Only accept the value if its hash equals the requested key.
				if NewValueID(result.value).Equals(targetID) {
					return result.value, batch[i], nil
				}
				// Discard the bad value, report it, and keep searching other nodes.
				log.Printf("error: node %s returned a value that does not match key %s, discarding",
					batch[i].Address, hash)
				continue
			}
			for _, contact := range result.contacts {
				MergeClosest(candidates, contact, targetID, shortListSize)
			}
		}
	}
}

func (kademlia *Kademlia) Store(data []byte) error {
	if kademlia == nil || kademlia.RoutingTable == nil || kademlia.Network == nil {
		return errors.New("invalid store arguments")
	}

	if len(data) > limitData {
		return errors.New("data exceeds maximum size limit")
	}

	targetID := NewValueID(data)
	target := Contact{ID: targetID}
	contacts, err := kademlia.LookupContact(&target)
	if err != nil {
		return err
	}

	// The node performing the lookup can itself be one of the k closest nodes.
	candidates := &ContactCandidates{}
	candidates.Append(contacts)
	MergeClosest(candidates, kademlia.Contact, targetID, shortListSize)

	var storeErr error
	for _, contact := range candidates.GetContacts(candidates.Len()) {
		if contact.ID.Equals(kademlia.Contact.ID) {
			kademlia.muData.Lock()

			if kademlia.Data == nil {
				kademlia.Data = make(map[string][]byte)
			}
			kademlia.Data[targetID.String()] = data
			kademlia.muData.Unlock()
			continue
		}
		if err := kademlia.Network.SendStoreMessage(&contact, targetID, data); err != nil {
			storeErr = err
		}
	}
	return storeErr
}

func (kademlia *Kademlia) StorePart2(key []byte, data []byte) error {
	if kademlia == nil || kademlia.RoutingTable == nil || kademlia.Network == nil {
		return errors.New("invalid store arguments")
	}

	if len(data) > limitData {
		return errors.New("data exceeds maximum size limit")
	}

	targetID := NewValueID(key)
	target := Contact{ID: targetID}
	contacts, err := kademlia.LookupContact(&target)
	if err != nil {
		return err
	}

	// The node performing the lookup can itself be one of the k closest nodes.
	candidates := &ContactCandidates{}
	candidates.Append(contacts)
	MergeClosest(candidates, kademlia.Contact, targetID, shortListSize)

	var storeErr error
	for _, contact := range candidates.GetContacts(candidates.Len()) {
		if contact.ID.Equals(kademlia.Contact.ID) {
			kademlia.muData.Lock()

			if kademlia.Data == nil {
				kademlia.Data = make(map[string][]byte)
			}
			kademlia.Data[targetID.String()] = data
			kademlia.muData.Unlock()
			continue
		}
		if err := kademlia.Network.SendStorePublishMessage(&contact, targetID, data); err != nil {
			storeErr = err
		}
	}
	return storeErr
}

// HELPER FUNCTIONS
// ------------------

func (kademlia *Kademlia) recordLookup(success bool) {
	if success {
		kademlia.LookupStats.successes.Add(1)
		return
	}

	kademlia.LookupStats.failures.Add(1)

}

func (kademlia *Kademlia) GetStats() (successes, failures, probes int64) {
	return kademlia.LookupStats.successes.Load(),
		kademlia.LookupStats.failures.Load(),
		kademlia.LookupStats.totalProbes.Load()
}

func (kademlia *Kademlia) GetHopStats() int64 {
	return kademlia.LookupStats.totalHops.Load()
}

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

	var wg sync.WaitGroup
	wg.Add(len(batch))
	for i, contact := range batch {
		go func(i int, contact Contact) {
			defer wg.Done()
			// Query concurrently within the batch
			contacts, err := network.SendFindContactMessage(&contact, targetID)
			if err != nil {
				if errors.Is(err, ErrPD) || errors.Is(err, ErrRPCTimeout) {
					return

				}
				errCh <- err
				return
			}
			results[i] = contacts
		}(i, contact)

	}
	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
		return results, nil
	}
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

	case "STORE", "STORE_PUBLISH":
		// reject mappings where the key isn't the hash of the value (K = hash(V))
		if !msg.Target.Equals(NewValueID(msg.Value)) && msg.Type == "STORE" {
			return kademlia.Network.listener.Send(Message{
				MessageId: msg.MessageId,
				From:      kademlia.Contact,
				To:        msg.From.Address,
				Type:      "STORE_ERROR",
			})
		}
		return kademlia.handleStore(msg)
	case "PING_RETURN":
		kademlia.Network.recmu.Lock()
		respChannel := kademlia.Network.receive[msg.MessageId]
		kademlia.Network.recmu.Unlock()
		respChannel <- msg
		return nil

	case "FIND_NODE_RESPONSE":
		kademlia.Network.recmu.Lock()
		respChannel := kademlia.Network.receive[msg.MessageId]
		kademlia.Network.recmu.Unlock()
		respChannel <- msg
		return nil

	case "FIND_VALUE_RESPONSE":
		kademlia.Network.recmu.Lock()
		respChannel := kademlia.Network.receive[msg.MessageId]
		kademlia.Network.recmu.Unlock()
		respChannel <- msg
		return nil

	case "STORE_RESPONSE":
		kademlia.Network.recmu.Lock()
		respChannel := kademlia.Network.receive[msg.MessageId]
		kademlia.Network.recmu.Unlock()
		respChannel <- msg
		return nil

	case "STORE_ERROR":
		kademlia.Network.recmu.Lock()
		respChannel := kademlia.Network.receive[msg.MessageId]
		kademlia.Network.recmu.Unlock()
		respChannel <- msg
		return nil
	}
	return errors.New("No of those functions exists")
}

func (kademlia *Kademlia) FindReceiverData(msg Message) error {
	if msg.Target == nil {
		return errors.New("find value message has no target")
	}

	kademlia.muData.Lock()
	value, ok := kademlia.Data[msg.Target.String()]
	kademlia.muData.Unlock()
	if ok {
		message := Message{
			MessageId: msg.MessageId,
			From:      kademlia.Contact,
			To:        msg.From.Address,
			Type:      "FIND_VALUE_RESPONSE",
			Value:     value,
			Found:     true,
		}
		return kademlia.Network.listener.Send(message)
	}

	contacts := kademlia.RoutingTable.FindClosestContacts(msg.Target, shortListSize)
	message := Message{
		MessageId: msg.MessageId,
		From:      kademlia.Contact,
		To:        msg.From.Address,
		Type:      "FIND_VALUE_RESPONSE",
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

func NewKademlia(id *KademliaID, address string) (*Kademlia, error) {
	if err := validateNodeID(id, address); err != nil {
		return nil, err
	}

	contact := NewContact(id, address)

	network, err := initNetwork(NewRealNetwork(), contact)
	if err != nil {
		return nil, err
	}

	node := &Kademlia{
		Contact:      contact,
		RoutingTable: NewRoutingTable(contact),
		Network:      network,
		Data:         make(map[string][]byte),
	}

	network.ServerListen(node)

	return node, nil
}

func validateNodeID(id *KademliaID, address string) error {
	// SplitHostPort safely separates the host and port, including bracketed IPv6 addresses.
	host, portString, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid node address %q: %w", address, err)
	}

	port, err := strconv.Atoi(portString)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("node address must use a port between 1 and 65535")
	}

	// ParseIP rejects hostnames and normalizes the IP before checking its derived ID.
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("node address host %q must be an IP address", host)
	}

	// Port zero is rejected because the OS-assigned port would differ from the
	// port used to derive the node ID.
	expectedID := IPPortDerivedNodeID(ip.String(), port)
	if id == nil || !id.Equals(expectedID) {
		return errors.New("node ID must be the SHA-256 hash of the node IP and port")
	}

	return nil
}

func (kademlia *Kademlia) Close() error {
	if kademlia == nil || kademlia.Network == nil {
		return nil
	}

	return kademlia.Network.Close()
}

func (kademlia *Kademlia) handleStore(msg Message) error {
	if msg.Target == nil {
		return kademlia.Network.listener.Send(Message{
			MessageId: msg.MessageId,
			From:      kademlia.Contact,
			To:        msg.From.Address,
			Type:      "STORE_ERROR",
		})
	}
	if kademlia.Data == nil {
		kademlia.Data = make(map[string][]byte)
	}
	kademlia.muData.Lock()

	kademlia.Data[msg.Target.String()] = msg.Value
	kademlia.muData.Unlock()

	message := Message{
		MessageId: msg.MessageId,
		From:      kademlia.Contact,
		To:        msg.From.Address,
		Type:      "STORE_RESPONSE",
	}

	return kademlia.Network.listener.Send(message)
}
