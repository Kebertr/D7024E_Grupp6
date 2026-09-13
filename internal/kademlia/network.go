package kademlia

import "errors"

type Network struct {
	transport Transport
	address   Address
	listener  Connection
	receive   chan Message
}

type Transport interface {
	Listen(addr Address) (Connection, error)
}

type Connection interface {
	Send(msg Message) error
	Recv() (Message, error)
	Close() error
}

type Message struct {
	From     Address
	To       Address
	Type     string
	Target   *KademliaID
	Contacts []Contact
}

func (network *Network) SendPingMessage(contact *Contact) {
	// TODO
}

func (network *Network) SendFindContactMessage(contact *Contact, targetId *KademliaID) ([]Contact, error) {
	if contact == nil || targetId == nil {
		return nil, errors.New("Either contact or targetId is nil")
	}

	msg := Message{
		From:   network.address,
		To:     contact.Address,
		Type:   "FIND_NODE",
		Target: targetId,
	}

	err := network.listener.Send(msg)
	if err != nil {
		return nil, err
	}

	response := <-network.receive

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
	go func() {
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
			}
		}
	}()
}

func (network *Network) FindReceiverNodes(toAddress Address, contacts []Contact) error {

	response := Message{
		From:     network.address,
		To:       toAddress,
		Type:     "FIND_NODE_RESPONSE",
		Contacts: contacts,
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
