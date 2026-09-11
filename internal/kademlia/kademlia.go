package kademlia

import (
	"fmt"
	"sync"

	transport "github.com/RasmusKebert/D7024E_Grupp6/internal/network"
)

type Kademlia struct {
	me *Contact

	// transportNode owns the simulated communication endpoint. Kademlia uses
	// it without depending on a concrete socket implementation.
	transportNode *transport.Node
	protocol      *Network

	routingTable *RoutingTable
	// routingMu protects routing-table reads and writes performed by RPC handlers.
	routingMu sync.RWMutex

	// dataStore contains values owned by this node, indexed by their SHA-256
	// key. The mutex protects it when future RPC handlers access it concurrently.
	dataStore map[string][]byte
	dataMu    sync.RWMutex

	// These parameters are kept on the node so lookup and replication behavior
	// can be adjusted without changing the node implementation.
	alpha int
	k     int
}

const (
	defaultAlpha = 3
	defaultK     = 10
)

// NewKademlia creates a node backed by the supplied simulated network.
func NewKademlia(network transport.Network, address transport.Address) (*Kademlia, error) {
	transportNode, err := transport.NewNode(network, address)
	if err != nil {
		return nil, fmt.Errorf("create Kademlia node: %w", err)
	}

	id := NewKademliaID(transportNode.ID)
	me := NewContact(id, address.String())

	node := &Kademlia{
		me:            &me,
		transportNode: transportNode,
		routingTable:  NewRoutingTable(me),
		dataStore:     make(map[string][]byte),
		alpha:         defaultAlpha,
		k:             defaultK,
	}
	node.protocol = NewNetwork(node)
	return node, nil
}

// ID returns this node's Kademlia identifier.
func (kademlia *Kademlia) ID() *KademliaID {
	return kademlia.me.ID
}

// Address returns this node's network address in IP:port form.
func (kademlia *Kademlia) Address() string {
	return kademlia.me.Address
}

// Contact returns the node's local contact information.
func (kademlia *Kademlia) Contact() Contact {
	return *kademlia.me
}

// RoutingTable returns the routing table owned by this node.
func (kademlia *Kademlia) RoutingTable() *RoutingTable {
	return kademlia.routingTable
}

// Transport returns the underlying abstract transport node. Protocol code can
// register handlers and send messages through this abstraction.
func (kademlia *Kademlia) Transport() *transport.Node {
	return kademlia.transportNode
}

// SetLookupParameters changes the lookup parallelism and replication factor.
func (kademlia *Kademlia) SetLookupParameters(alpha, replicationFactor int) error {
	if alpha < 1 {
		return fmt.Errorf("alpha must be at least 1")
	}
	if replicationFactor < 1 {
		return fmt.Errorf("replication factor must be at least 1")
	}
	kademlia.alpha = alpha
	kademlia.k = replicationFactor
	return nil
}

// LookupParameters returns the node's configured lookup and replication values.
func (kademlia *Kademlia) LookupParameters() (alpha, replicationFactor int) {
	return kademlia.alpha, kademlia.k
}

// Close shuts down the node's transport endpoint.
func (kademlia *Kademlia) Close() error {
	return kademlia.transportNode.Close()
}

// StoredValues returns a copy of the node's local datastore.
func (kademlia *Kademlia) StoredValues() map[string][]byte {
	kademlia.dataMu.RLock()
	defer kademlia.dataMu.RUnlock()

	values := make(map[string][]byte, len(kademlia.dataStore))
	for key, value := range kademlia.dataStore {
		values[key] = append([]byte(nil), value...)
	}
	return values
}

func (kademlia *Kademlia) addContact(contact Contact) {
	kademlia.routingMu.Lock()
	defer kademlia.routingMu.Unlock()
	kademlia.routingTable.AddContact(contact)
}

func (kademlia *Kademlia) closestContacts(target *KademliaID, count int) []Contact {
	kademlia.routingMu.RLock()
	defer kademlia.routingMu.RUnlock()
	return kademlia.routingTable.FindClosestContacts(target, count)
}

// storeLocal validates and stores one value received through a STORE RPC.
// The copy prevents a caller from changing the value after it is stored.
func (kademlia *Kademlia) storeLocal(key string, value []byte) error {
	if err := validateKeyValue(key, value); err != nil {
		return err
	}

	kademlia.dataMu.Lock()
	defer kademlia.dataMu.Unlock()
	kademlia.dataStore[key] = append([]byte(nil), value...)
	return nil
}

func (kademlia *Kademlia) LookupContact(target *Contact) {
	// TODO
}

func (kademlia *Kademlia) LookupData(hash string) {
	// TODO
}

func (kademlia *Kademlia) Store(data []byte) {
	// TODO
}
