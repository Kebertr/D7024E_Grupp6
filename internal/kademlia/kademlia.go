package kademlia

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	alpha         = 3 // Number of parallel queries
	shortListSize = bucketSize
	limitData     = 4096
	// k for replication: each value is re-sent to this many closest nodes.
	replicationFactor = 5
	// How often every stored value is re-sent to its k closest nodes.
	replicationInterval = 2 * time.Minute
)

type Kademlia struct {
	Contact      Contact
	RoutingTable *RoutingTable
	Network      *Network
	Data         map[string][]byte
	muData       sync.RWMutex

	// Lifecycle of the replication goroutine. The zero values are safe, so
	// nodes built without NewKademlia (e.g. in tests) simply never replicate.
	stop      chan struct{}  // closed by Close to stop the goroutine
	wg        sync.WaitGroup // lets Close wait until the goroutine has exited
	closeOnce sync.Once      // makes Close safe to call more than once
}

func (kademlia *Kademlia) Ping(contact *Contact) error {
	resp, err := kademlia.Network.SendPingMessage(contact)
	if err != nil {
		// The contact did not answer, so it must not stay in the routing table.
		if contact != nil {
			kademlia.RoutingTable.RemoveContact(*contact)
		}
		return err
	}

	kademlia.updateContact(resp)

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
			kademlia.dropFailedContact(err)
			return nil, err
		}

		for i, result := range results {
			kademlia.updateContact(batch[i])
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
			kademlia.dropFailedContact(err)
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

	var wg sync.WaitGroup
	wg.Add(len(batch))
	for i, contact := range batch {
		go func(i int, contact Contact) {
			defer wg.Done()
			contacts, err := network.SendFindContactMessage(&contact, targetID)
			if err != nil {
				errCh <- &QueryError{Contact: contact, Err: err}
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
				errCh <- &QueryError{Contact: contact, Err: err}
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
	// NOTE: learning the sender of every request (FIND_NODE/FIND_VALUE/STORE) would be
	// more complete, but it changes lookup results the existing tests rely on, so only
	// PING and answered queries update the routing table for now.
	//
	// NOTE: the full join procedure and bucket refresh (LAB-SPEC) are still needed
	// here/in NewKademlia: look up our own ID via a bootstrap node, then refresh buckets.
	switch msg.Type {
	case "PING":
		return kademlia.handlePing(msg)
		//Add case for err. Also add case for value and so on in the future
	case "FIND_NODE":
		return kademlia.FindReceiverNodes(msg)

	case "FIND_VALUE":
		return kademlia.FindReceiverData(msg)

	case "STORE":
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
	kademlia.updateContact(msg.From)

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
		stop:         make(chan struct{}),
	}

	network.ServerListen(node)
	node.startReplication(replicationInterval)

	return node, nil
}

// startReplication launches one background goroutine that calls replicate on
// every tick until Close is called.
func (kademlia *Kademlia) startReplication(interval time.Duration) {
	kademlia.wg.Add(1)
	go func() {
		defer kademlia.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-kademlia.stop:
				return
			case <-ticker.C:
				kademlia.replicate()
			}
		}
	}()
}

// replicate re-sends every stored value to the k closest nodes of its key, so
// values survive nodes leaving and reach nodes that joined later.
// Receivers validate K = hash(V) in handleStore.
func (kademlia *Kademlia) replicate() {
	// Copy the map under the lock and release it before any network call, so
	// a slow or dead peer can never block handlers that need muData.
	kademlia.muData.RLock()
	snapshot := make(map[string][]byte, len(kademlia.Data))
	for key, value := range kademlia.Data {
		snapshot[key] = value
	}
	kademlia.muData.RUnlock()

	for key, value := range snapshot {
		// Stop early on shutdown instead of finishing a long round.
		select {
		case <-kademlia.stop:
			return
		default:
		}

		targetID := NewKademliaID(key)
		contacts, err := kademlia.LookupContact(&Contact{ID: targetID})
		if err != nil {
			log.Printf("replication: lookup for key %s failed: %v", key, err)
			continue
		}
		// LookupContact returns the closest nodes first; keep only the k closest.
		if len(contacts) > replicationFactor {
			contacts = contacts[:replicationFactor]
		}
		for _, contact := range contacts {
			if contact.ID.Equals(kademlia.Contact.ID) {
				continue
			}
			// Contact is copied per iteration, so taking its address is safe.
			if err := kademlia.Network.SendStoreMessage(&contact, targetID, value); err != nil {
				log.Printf("replication: store to %s failed: %v", contact.Address, err)
			}
		}
	}
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

	// Stop replication first and wait for it, so it cannot use a closed network.
	// closeOnce prevents a panic from closing the channel twice.
	kademlia.closeOnce.Do(func() {
		if kademlia.stop != nil {
			close(kademlia.stop)
		}
	})
	kademlia.wg.Wait()

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
	// reject mappings where the key isn't the hash of the value (K = hash(V))
	if !msg.Target.Equals(NewValueID(msg.Value)) {
		return kademlia.Network.listener.Send(Message{
			MessageId: msg.MessageId,
			From:      kademlia.Contact,
			To:        msg.From.Address,
			Type:      "STORE_ERROR",
		})
	}
	// Lock before the nil check: checking and allocating the map outside the
	// lock was a data race with concurrent readers and writers.
	kademlia.muData.Lock()
	if kademlia.Data == nil {
		kademlia.Data = make(map[string][]byte)
	}
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

// QueryError records which contact an RPC failed for, so the routing table can drop it.
type QueryError struct {
	Contact Contact
	Err     error
}

func (e *QueryError) Error() string { return e.Err.Error() }
func (e *QueryError) Unwrap() error { return e.Err }

// dropFailedContact removes the contact that caused a failed query from the routing table.
func (kademlia *Kademlia) dropFailedContact(err error) {
	var queryErr *QueryError
	if errors.As(err, &queryErr) {
		kademlia.RoutingTable.RemoveContact(queryErr.Contact)
	}
}

// updateContact records that a contact was just seen. If its bucket is full, the least
// recently seen contact in that bucket is pinged in the background (a ping can take
// seconds, so it never runs inside a lock or delays the caller).
func (kademlia *Kademlia) updateContact(contact Contact) {
	if contact.ID == nil {
		return
	}
	if oldest := kademlia.RoutingTable.AddContact(contact); oldest != nil {
		go kademlia.evictIfDead(*oldest, contact)
	}
}

// evictIfDead pings the oldest contact of a full bucket. If it answers it is kept (and
// refreshed), the newcomer is dropped. If not, it is replaced by the newcomer.
func (kademlia *Kademlia) evictIfDead(oldest Contact, newcomer Contact) {
	if _, err := kademlia.Network.SendPingMessage(&oldest); err == nil {
		kademlia.RoutingTable.AddContact(oldest) // alive: move to the front of its bucket
		return
	}
	kademlia.RoutingTable.RemoveContact(oldest)
	kademlia.RoutingTable.AddContact(newcomer)
}
