package kademlia

import (
	"errors"
	"sync"
	"time"
)

type Network struct {
	transport Transport
	contact   Contact
	listener  Connection
	recmu     sync.RWMutex
	receive   map[messageId]chan Message
	wg        sync.WaitGroup
}

type Address = string
type messageId [32]byte

type Transport interface {
	Listen(addr Address) (Connection, error)
}

type Connection interface {
	Send(msg Message) error
	Recv() (Message, error)
	Close() error
}

type Message struct {
	MessageId messageId
	From      Contact
	To        Address
	Type      string
	Target    *KademliaID
	Contacts  []Contact
	Value     []byte
	Found     bool
}

func (network *Network) SendPingMessage(contact *Contact) (Contact, error) {
	id := createMessageId()

	if contact == nil {
		return Contact{}, errors.New("There should be a node")
	}

	respChannel := make(chan Message, 1)
	//We need to lock it since otherwise it might been written to multiple times and read from
	network.recmu.Lock()
	network.receive[id] = respChannel
	network.recmu.Unlock()

	msg := Message{
		MessageId: id,
		From:      network.contact,
		To:        contact.Address,
		Type:      "PING",
	}
	err := network.listener.Send(msg)
	if err != nil {
		return Contact{}, err
	}

	select {
	case response := <-respChannel:
		if response.MessageId != id {
			return Contact{}, errors.New("The Id do not match")
		}
		return response.From, nil
	case <-time.After(2 * time.Second):
		return Contact{}, errors.New("ping timed out")
	}
}

func (network *Network) SendFindContactMessage(contact *Contact, targetId *KademliaID) ([]Contact, error) {

	id := createMessageId()

	if contact == nil || targetId == nil {
		return nil, errors.New("Either contact or targetId is nil")
	}
	respChannel := make(chan Message, 1)
	network.recmu.Lock()
	network.receive[id] = respChannel
	network.recmu.Unlock()

	msg := Message{
		MessageId: id,
		From:      network.contact,
		To:        contact.Address,
		Type:      "FIND_NODE",
		Target:    targetId,
	}

	err := network.listener.Send(msg)
	if err != nil {
		return nil, err
	}

	response := <-respChannel

	if response.MessageId != id {
		return nil, errors.New("The Id do not match")
	}

	return response.Contacts, nil

}

func (network *Network) SendFindDataMessage(contact *Contact, target *KademliaID) ([]byte, []Contact, bool, error) {
	if contact == nil || target == nil {
		return nil, nil, false, errors.New("contact and target are required")
	}

	id := createMessageId()

	respChannel := make(chan Message, 1)
	network.recmu.Lock()
	network.receive[id] = respChannel
	network.recmu.Unlock()

	msg := Message{
		MessageId: id,
		From:      network.contact,
		To:        contact.Address,
		Type:      "FIND_VALUE",
		Target:    target,
	}

	err := network.listener.Send(msg)
	if err != nil {
		return nil, nil, false, err
	}

	response := <-respChannel
	if response.MessageId != id {
		return nil, nil, false, errors.New("The Id do not match")
	}
	if response.Type != "FIND_VALUE_RESPONSE" {
		return nil, nil, false, errors.New("invalid find data response")
	}
	if response.Found {
		if len(response.Value) > MaxValueSize {
			return nil, nil, false, errors.New("value exceeds 4 KiB limit")
		}
		return append([]byte(nil), response.Value...), nil, true, nil
	}
	return nil, response.Contacts, false, nil
}

func (network *Network) SendStoreMessage(contact *Contact, target *KademliaID, data []byte) error {
	if contact == nil {
		return errors.New("contact is required")
	}
	if len(data) > MaxValueSize {
		return errors.New("value exceeds 4 KiB limit")
	}

	id := createMessageId()

	respChannel := make(chan Message, 1)
	network.recmu.Lock()
	network.receive[id] = respChannel
	network.recmu.Unlock()

	msg := Message{
		MessageId: id,
		From:      network.contact,
		To:        contact.Address,
		Type:      "STORE",
		Target:    target,
		Value:     append([]byte(nil), data...),
	}

	err := network.listener.Send(msg)
	if err != nil {
		return err
	}

	response := <-respChannel // Should we have timeouts? Risk of waiting endlessly
	if response.MessageId != id || response.Type != "STORE_RESPONSE" {
		if response.MessageId == id && response.Type == "STORE_ERROR" {
			return errors.New("store rejected by receiver")
		}
		return errors.New("invalid store response")
	}
	return nil
}

// This is for receiving each message and send it to the right function
func (network *Network) ServerListen(kademlia *Kademlia) {
	go func() {
		for {
			msg, err := network.listener.Recv()
			if err != nil {
				return
			}
			go kademlia.handleIncomingMessage(msg)
		}
	}()
}

func (network *Network) FindReceiverNodes(Id messageId, toAddress Address, contacts []Contact) error {

	response := Message{
		MessageId: Id,
		From:      network.contact,
		To:        toAddress,
		Type:      "FIND_NODE_RESPONSE",
		Contacts:  contacts,
	}

	err := network.listener.Send(response)
	if err != nil {
		return err
	}
	return nil
}

// Initializes the network. Good for not initalizing it in every test
func initNetwork(transport Transport, contact Contact) (*Network, error) {
	listener, err := transport.Listen(contact.Address)
	if err != nil {
		return nil, err
	}

	network := &Network{
		transport: transport,
		contact:   contact,
		listener:  listener,
		receive:   make(map[messageId]chan Message),
	}

	return network, nil
}

func (network *Network) Close() error {
	if network == nil || network.listener == nil {
		return nil
	}

	return network.listener.Close()
}

func createMessageId() messageId {
	Id := NewRandomKademliaID()
	return messageId(*Id)
}
