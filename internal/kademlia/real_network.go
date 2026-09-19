package kademlia

import (
	"errors"
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
	c.mu.RLock()
	if c.closed || c.recvCh == nil {
		c.mu.RUnlock()
		return Message{}, errors.New("connection not listening")
	}
	ch := c.recvCh
	c.mu.RUnlock()

	msg, ok := <-ch
	if !ok {
		return Message{}, errors.New("connection closed")
	}
	return msg, nil
}

func (c *realConnection) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil // Already closed
	}
	c.closed = true
	c.mu.Unlock()

	c.network.mu.Lock()
	defer c.network.mu.Unlock()

	if c.recvCh != nil {
		close(c.recvCh)
		delete(c.network.listeners, c.addr)
		c.recvCh = nil
	}
	return nil
}
