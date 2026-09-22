package kademlia

import (
	"net"
	"sync"

	"google.golang.org/protobuf/proto"
)

type realNetwork struct {
	mu        sync.RWMutex
	listeners map[Address]chan Message
}

func NewRealNetwork() Transport {
	return &realNetwork{
		listeners: make(map[Address]chan Message),
	}
}

func (n *realNetwork) Listen(addr Address) (Connection, error) {
	udpaddr, err := net.ResolveUDPAddr("udp", addr)

	if err != nil {
		return nil, err
	}
	conn, connerr := net.ListenUDP("udp", udpaddr)
	if connerr != nil {
		return nil, connerr
	}
	return &realConnection{conn: conn}, nil
}

type realConnection struct {
	conn *net.UDPConn
}

func (c *realConnection) Send(msg Message) error {
	udpaddr, err := net.ResolveUDPAddr("udp", msg.To)
	if err != nil {
		return err
	}

	msgProto := &MessageProto{
		MessageId: msg.MessageId[:],
		From: &ContactProto{
			Id:      msg.From.ID[:],
			Address: msg.From.Address,
		},
		To:    msg.To,
		Type:  msg.Type,
		Value: msg.Value,
	}

	if msg.Target != nil {
		msgProto.Target = msg.Target[:]
	}

	for i := 0; i < len(msg.Contacts); i++ {
		conProto := &ContactProto{
			Id:      msg.Contacts[i].ID[:],
			Address: msg.Contacts[i].Address,
		}
		msgProto.Contacts = append(msgProto.Contacts, conProto)
	}

	data, err := proto.Marshal(msgProto)

	if err != nil {
		return err
	}
	_, senderr := c.conn.WriteToUDP(data, udpaddr)

	return senderr

}

func (c *realConnection) Recv() (Message, error) {
	buffer := make([]byte, 1000)

	var msgProto MessageProto

	n, _, err := c.conn.ReadFromUDP(buffer)

	if err != nil {
		return Message{}, err
	}

	err = proto.Unmarshal(buffer[:n], &msgProto)

	if err != nil {
		return Message{}, err
	}

	message := Message{
		MessageId: messageId(msgProto.GetMessageId()),
		From: Contact{
			ID:      (*KademliaID)(msgProto.From.Id),
			Address: msgProto.From.Address,
		},
		To:    msgProto.To,
		Type:  msgProto.GetType(),
		Value: msgProto.GetValue(),
	}

	if len(msgProto.Target) > 0 {
		message.Target = (*KademliaID)(msgProto.GetTarget())
	}

	for i := 0; i < len(msgProto.GetContacts()); i++ {
		id := KademliaID(msgProto.GetContacts()[i].GetId())
		contacts := Contact{
			ID:      &id,
			Address: msgProto.Contacts[i].Address,
		}
		message.Contacts = append(message.Contacts, contacts)
	}

	return message, nil

}

func (c *realConnection) Close() error {
	return c.conn.Close()
}
