package kademlia

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

type mockNetwork struct {
	mu          sync.RWMutex
	listeners   map[Address]chan Message
	latency     float64
	packet_loss float64
}

func NewMockNetwork() *mockNetwork {
	return &mockNetwork{
		listeners:   make(map[Address]chan Message),
		latency:     0,
		packet_loss: 0,
	}
}

func (n *mockNetwork) Listen(addr Address) (Connection, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.listeners[addr]; exists {
		return nil, errors.New("address already in use")
	}
	ch := make(chan Message, 100) // buffered channel
	n.listeners[addr] = ch
	return &mockConnection{addr: addr, network: n, recvCh: ch}, nil
}

type mockConnection struct {
	addr    Address
	network *mockNetwork
	recvCh  chan Message
	mu      sync.RWMutex
	closed  bool
}

// Need to be outside function so Query in kademlia can reach this
var ErrPD = errors.New("Packet dropped")

// !!IMPORTANT!!
// Here is where we set the seed
var rng = rand.New(rand.NewSource(1))
var rngMu sync.Mutex

func randomFloat64() float64 {
	rngMu.Lock()
	defer rngMu.Unlock()
	return rng.Float64()
}
func (c *mockConnection) Send(msg Message) error {
	c.network.mu.RLock()

	ch, exists := c.network.listeners[msg.To]
	if !exists {
		c.network.mu.RUnlock()
		return errors.New("destination address not found")
	}
	if randomFloat64() < c.network.packet_loss {
		c.network.mu.RUnlock()
		return ErrPD
	}

	time.Sleep(time.Duration(c.network.latency))

	// Keep the lock while sending to prevent the channel from being closed
	select {
	case ch <- msg:
		c.network.mu.RUnlock()
		return nil
	default:
		c.network.mu.RUnlock()
		return errors.New("message queue full")
	}
}

func (c *mockConnection) Recv() (Message, error) {
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

func (c *mockConnection) Close() error {
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
