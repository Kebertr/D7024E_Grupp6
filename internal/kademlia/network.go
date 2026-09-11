package kademlia

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	transport "github.com/RasmusKebert/D7024E_Grupp6/internal/network"
)

// Protocol message names used on the simulated RPC transport.
const (
	MessagePing           = "PING"
	MessagePong           = "PONG"
	MessageFindNode       = "FIND_NODE"
	MessageFindNodeReply  = "FIND_NODE_REPLY"
	MessageFindValue      = "FIND_VALUE"
	MessageFindValueReply = "FIND_VALUE_REPLY"
	MessageStore          = "STORE"
	MessageStoreReply     = "STORE_REPLY"
	MessageProtocolError  = "ERROR"
)

// Network adapts the generic simulated transport to Kademlia RPCs.
// transport.Network remains responsible only for packet delivery and failure
// simulation; this type interprets Kademlia protocol messages.
type Network struct {
	node *Kademlia
}

// NewNetwork creates and registers the protocol adapter for a Kademlia node.
func NewNetwork(node *Kademlia) *Network {
	adapter := &Network{node: node}
	transportNode := node.transportNode
	transportNode.Handle(MessagePing, adapter.handlePing)
	transportNode.Handle(MessagePong, adapter.handlePong)
	transportNode.Handle(MessageFindNode, adapter.handleFindNode)
	transportNode.Handle(MessageFindNodeReply, adapter.handleFindNodeReply)
	transportNode.Handle(MessageFindValue, adapter.handleFindValue)
	transportNode.Handle(MessageFindValueReply, adapter.handleFindValueReply)
	transportNode.Handle(MessageStore, adapter.handleStore)
	transportNode.Handle(MessageStoreReply, adapter.handleStoreReply)
	transportNode.Handle(MessageProtocolError, adapter.handleProtocolError)
	transportNode.Start()
	return adapter
}

// WireContact is the JSON-safe representation of a Kademlia contact.
type WireContact struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

type findNodeRequest struct {
	Target string `json:"target"`
}

type findNodeResponse struct {
	Contacts []WireContact `json:"contacts"`
}

type findValueRequest struct {
	Key string `json:"key"`
}

type findValueResponse struct {
	Found    bool          `json:"found"`
	Key      string        `json:"key,omitempty"`
	Value    []byte        `json:"value,omitempty"`
	Contacts []WireContact `json:"contacts,omitempty"`
}

type storeRequest struct {
	Key   string `json:"key"`
	Value []byte `json:"value"`
}

type protocolError struct {
	Error string `json:"error"`
}

func encode(value any) ([]byte, error) { return json.Marshal(value) }

func decode(data []byte, target any) error {
	// The tutorial transport keeps backward compatibility with string
	// messages by prefixing payloads with "TYPE:". Message.Type is already
	// available, but protocol handlers must accept that legacy envelope too.
	if separator := bytes.IndexByte(data, ':'); separator >= 0 {
		data = data[separator+1:]
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode protocol payload: %w", err)
	}
	return nil
}

func contactToWire(contact Contact) WireContact {
	return WireContact{ID: contact.ID.String(), Address: contact.Address}
}

func wireToContact(contact WireContact) (Contact, error) {
	if contact.ID == "" || contact.Address == "" {
		return Contact{}, fmt.Errorf("contact must contain an ID and address")
	}
	if len(contact.ID) != IDLength*2 {
		return Contact{}, fmt.Errorf("invalid contact ID length")
	}
	if _, err := hex.DecodeString(contact.ID); err != nil {
		return Contact{}, fmt.Errorf("invalid contact ID: %w", err)
	}
	if _, _, err := net.SplitHostPort(contact.Address); err != nil {
		return Contact{}, fmt.Errorf("invalid contact address %q: %w", contact.Address, err)
	}
	return NewContact(NewKademliaID(contact.ID), contact.Address), nil
}

func (network *Network) reply(msg transport.Message, messageType string, value any) error {
	payload, err := encode(value)
	if err != nil {
		return err
	}
	return msg.Reply(messageType, payload)
}

func (network *Network) replyError(msg transport.Message, err error) error {
	return network.reply(msg, MessageProtocolError, protocolError{Error: err.Error()})
}

// SendPingMessage sends a PING RPC to a contact.
func (network *Network) SendPingMessage(contact *Contact) error {
	return network.send(contact, MessagePing, struct{}{})
}

// SendFindContactMessage sends a FIND_NODE RPC for the contact's ID.
func (network *Network) SendFindContactMessage(contact *Contact) error {
	if contact == nil || contact.ID == nil {
		return fmt.Errorf("contact or contact ID is nil")
	}
	return network.send(contact, MessageFindNode, findNodeRequest{Target: contact.ID.String()})
}

// SendFindDataMessage sends a FIND_VALUE RPC to a contact.
func (network *Network) SendFindDataMessage(contact *Contact, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	return network.send(contact, MessageFindValue, findValueRequest{Key: key})
}

// SendStoreMessage sends a STORE RPC containing an explicit content key.
func (network *Network) SendStoreMessage(contact *Contact, key string, value []byte) error {
	if err := validateKeyValue(key, value); err != nil {
		return err
	}
	return network.send(contact, MessageStore, storeRequest{Key: key, Value: value})
}

func (network *Network) send(contact *Contact, messageType string, payload any) error {
	if contact == nil {
		return fmt.Errorf("contact is nil")
	}
	data, err := encode(payload)
	if err != nil {
		return err
	}
	address, err := parseAddress(contact.Address)
	if err != nil {
		return err
	}
	return network.node.transportNode.Send(address, messageType, data)
}

func (network *Network) handlePing(msg transport.Message) error {
	network.rememberSender(msg)
	return network.reply(msg, MessagePong, struct{}{})
}

func (network *Network) handlePong(msg transport.Message) error {
	network.rememberSender(msg)
	return nil
}

func (network *Network) handleFindNode(msg transport.Message) error {
	var request findNodeRequest
	if err := decode(msg.Payload, &request); err != nil {
		return network.replyError(msg, err)
	}
	target, err := parseID(request.Target)
	if err != nil {
		return network.replyError(msg, err)
	}
	contacts := network.node.closestContacts(target, network.node.k)
	response := findNodeResponse{Contacts: make([]WireContact, 0, len(contacts))}
	for _, contact := range contacts {
		response.Contacts = append(response.Contacts, contactToWire(contact))
	}
	network.rememberSender(msg)
	return network.reply(msg, MessageFindNodeReply, response)
}

func (network *Network) handleFindNodeReply(msg transport.Message) error {
	var response findNodeResponse
	if err := decode(msg.Payload, &response); err != nil {
		return err
	}
	for _, wireContact := range response.Contacts {
		contact, err := wireToContact(wireContact)
		if err != nil {
			return err
		}
		network.node.addContact(contact)
	}
	return nil
}

func (network *Network) handleFindValue(msg transport.Message) error {
	var request findValueRequest
	if err := decode(msg.Payload, &request); err != nil {
		return network.replyError(msg, err)
	}
	if err := validateKey(request.Key); err != nil {
		return network.replyError(msg, err)
	}
	network.node.dataMu.RLock()
	value, found := network.node.dataStore[request.Key]
	value = append([]byte(nil), value...)
	network.node.dataMu.RUnlock()
	response := findValueResponse{Found: found, Key: request.Key, Value: value}
	//Return closer contacts if value was not found.
	if !found {
		keyID, _ := parseID(request.Key)
		contacts := network.node.closestContacts(keyID, network.node.k)
		for _, contact := range contacts {
			response.Contacts = append(response.Contacts, contactToWire(contact))
		}
	}
	network.rememberSender(msg)
	return network.reply(msg, MessageFindValueReply, response)
}

func (network *Network) handleFindValueReply(msg transport.Message) error {
	var response findValueResponse
	if err := decode(msg.Payload, &response); err != nil {
		return err
	}
	//Value validation
	if response.Found {
		if err := validateKeyValue(response.Key, response.Value); err != nil {
			return err
		}
		return nil
	}
	for _, wireContact := range response.Contacts {
		contact, err := wireToContact(wireContact)
		if err != nil {
			return err
		}
		network.node.addContact(contact)
	}
	return nil
}

func (network *Network) handleStore(msg transport.Message) error {
	var request storeRequest
	if err := decode(msg.Payload, &request); err != nil {
		return network.replyError(msg, err)
	}
	if err := network.node.storeLocal(request.Key, request.Value); err != nil {
		return network.replyError(msg, err)
	}
	network.rememberSender(msg)
	return network.reply(msg, MessageStoreReply, struct{}{})
}

func (network *Network) handleStoreReply(msg transport.Message) error { return nil }

func (network *Network) handleProtocolError(msg transport.Message) error {
	var response protocolError
	return decode(msg.Payload, &response)
}

func (network *Network) rememberSender(msg transport.Message) {
	// The transport message contains the sender's address, so derive the same
	// deterministic ID used when that sender constructed its node.
	sum := sha256.Sum256([]byte(msg.From.String()))
	contact := NewContact(NewKademliaID(hex.EncodeToString(sum[:])), msg.From.String())
	network.node.addContact(contact)
}

func parseID(value string) (*KademliaID, error) {
	if len(value) != IDLength*2 {
		return nil, fmt.Errorf("invalid Kademlia ID length")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return nil, fmt.Errorf("invalid Kademlia ID: %w", err)
	}
	return NewKademliaID(value), nil
}

func validateKey(key string) error {
	if _, err := parseID(key); err != nil {
		return fmt.Errorf("invalid SHA-256 key: %w", err)
	}
	return nil
}

func validateKeyValue(key string, value []byte) error {
	sum := sha256.Sum256(value)
	if hex.EncodeToString(sum[:]) != key {
		return fmt.Errorf("key does not match SHA-256 hash of value")
	}
	return nil
}

func parseAddress(address string) (transport.Address, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return transport.Address{}, fmt.Errorf("invalid address %q: %w", address, err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return transport.Address{}, fmt.Errorf("invalid port %q", port)
	}
	return transport.Address{IP: host, Port: portNumber}, nil
}
