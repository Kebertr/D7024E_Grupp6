package kademlia

import (
	"errors"
	"sync"
)

type Network struct {
	transport Transport
	contact   Contact
	listener  Connection
	receive   chan Message
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
}

func (network *Network) SendPingMessage(contact *Contact) error {
	id := createMessageId()

	if contact == nil {
		return errors.New("There should be a node")
	}

	msg := Message{
		MessageId: id,
		From:      network.contact,
		To:        contact.Address,
		Type:      "PING",
	}

	err := network.listener.Send(msg)
	if err != nil {
		return err
	}

	response := <-network.receive

	if response.MessageId != id {
		return errors.New("The Id do not match")
	}

	return nil
}

func (network *Network) SendFindContactMessage(contact *Contact, targetId *KademliaID) ([]Contact, error) {

	id := createMessageId()

	if contact == nil || targetId == nil {
		return nil, errors.New("Either contact or targetId is nil")
	}

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

	response := <-network.receive

	if response.MessageId != id {
		return nil, errors.New("The Id do not match")
	}

	return response.Contacts, nil

}

func (network *Network) SendFindDataMessage(hash string) {
	// TODO
}

func (network *Network) SendStoreMessage(contact *Contact, target *KademliaID, data []byte) error {
	if contact == nil || target == nil {
		return errors.New("contact and target are required")
	}

	id := createMessageId()

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

	response := <-network.receive // Should we have timeouts? Risk of waiting endlessly
	if response.MessageId != id || response.Type != "STORE_RESPONSE" {
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
		receive:   make(chan Message, 1),
	}

	return network, nil
}

func createMessageId() messageId {
	Id := NewRandomKademliaID()
	return messageId(*Id)
}
