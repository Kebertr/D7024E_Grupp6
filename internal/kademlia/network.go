package kademlia

import (
	"crypto/rand"
	"errors"
	"sync"
)

type Network struct {
	transport Transport
	address   Address
	listener  Connection
	receive   chan Message
	wg        sync.WaitGroup
}

type messageId [16]byte

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
	From      Address
	To        Address
	Type      string
	Target    *KademliaID
	Contacts  []Contact
	Value     []byte
}

func (network *Network) SendPingMessage(contact *Contact) (bool, error) {
	id, messerr := createMessageId()
	if messerr != nil {
		return false, messerr
	}

	if contact == nil {
		return false, messerr
	}

	msg := Message{
		MessageId: id,
		From:      network.address,
		To:        contact.Address,
		Type:      "PING",
	}

	err := network.listener.Send(msg)
	if err != nil {
		return false, err
	}

	response := <-network.receive

	if response.MessageId != id {
		return false, errors.New("The Id do not match")
	}

	return true, nil
}

func (network *Network) SendFindContactMessage(contact *Contact, targetId *KademliaID) ([]Contact, error) {

	id, messerr := createMessageId()

	if messerr != nil {
		return nil, messerr
	}
	if contact == nil || targetId == nil {
		return nil, errors.New("Either contact or targetId is nil")
	}

	msg := Message{
		MessageId: id,
		From:      network.address,
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

func (network *Network) SendStoreMessage(data []byte) {
	// TODO
}

// This is for receiving each message and send it to the right function
func (network *Network) serverListen(kademlia *Kademlia) {
	network.wg.Add(1)
	go func() {
		defer network.wg.Done()
		for {
			msg, err := network.listener.Recv()
			if err != nil {
				return
			}

			switch msg.Type {
			//Add case for err. Also add case for value and so on in the future
			case "FIND_NODE":
				kademlia.FindReceiverNodes(msg)

			case "FIND_NODE_RESPONSE":
				network.receive <- msg

			case "PING":
				response := Message{
					MessageId: msg.MessageId,
					From:      network.address,
					To:        msg.From,
					Type:      "PING_RETURN",
				}

				network.listener.Send(response)
			case "PING_RETURN":
				network.receive <- msg
			}
		}
	}()
}

func (network *Network) FindReceiverNodes(Id messageId, toAddress Address, contacts []Contact) error {

	response := Message{
		MessageId: Id,
		From:      network.address,
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
func initNetwork(transport Transport, address Address) (*Network, error) {
	listener, err := transport.Listen(address)
	if err != nil {
		return nil, err
	}

	network := &Network{
		transport: transport,
		address:   address,
		listener:  listener,
		receive:   make(chan Message, 1),
	}

	return network, nil
}

func createMessageId() (messageId, error) {
	var id messageId
	_, err := rand.Read(id[:])

	if err != nil {
		return id, err
	}
	return id, nil
}
