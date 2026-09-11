package network

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Address identifies a node in the simulated network.
type Address struct {
	IP   string
	Port int
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

// Network is the abstraction used to send packets between nodes.
type Network interface {
	Listen(addr Address) (Connection, error)
	Dial(addr Address) (Connection, error)
	Partition(group1, group2 []Address)
	Heal()
	SetPacketLoss(probability float64)
	SetLatency(delay time.Duration)
}

// Connection represents one socket between two peers.
type Connection interface {
	Send(msg Message) error
	Recv() (Message, error)
	Close() error
}

// Message is a generic packet exchanged by nodes.
type Message struct {
	From       Address
	To         Address
	Type       string
	Payload    []byte
	RequestID  uint64
	ResponseTo uint64
	network    Network
}

// Reply sends a response back to the original sender.
func (m Message) Reply(msgType string, data []byte) error {
	if m.network == nil {
		return fmt.Errorf("message has no network attached")
	}

	connection, err := m.network.Dial(m.From)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %v", m.From.String(), err)
	}
	defer connection.Close()

	var payload []byte
	if msgType != "" {
		payload = append([]byte(msgType+":"), data...) //Adds message type at begining
	} else {
		payload = data
	}

	reply := Message{
		From:       m.To,
		To:         m.From,
		Type:       msgType,
		Payload:    payload,
		RequestID:  0,
		ResponseTo: m.RequestID,
		network:    m.network,
	}

	return connection.Send(reply)
}

// ReplyString is a convenience wrapper for text replies.
func (m Message) ReplyString(msgType, data string) error {
	return m.Reply(msgType, []byte(data))
}

type mockConnection struct {
	network *mockNetwork
	addr    Address
	// recv is the connection's inbound message queue; channels let the node
	// block until a packet arrives without polling shared state.
	recv chan Message
	// closed is a broadcast shutdown signal observed by blocked receivers.
	closed chan struct{}
	// mu protects isOpen because Send and Close may run concurrently.
	mu     sync.RWMutex
	isOpen bool
}

func (c *mockConnection) Send(msg Message) error {
	if c.network == nil {
		return fmt.Errorf("connection has no network")
	}
	return c.network.send(msg)
}

func (c *mockConnection) Recv() (Message, error) {
	// Select lets a receiver wait for either a message or connection shutdown.
	select {
	case msg := <-c.recv:
		return msg, nil
	case <-c.closed:
		return Message{}, fmt.Errorf("connection closed")
	}
}

func (c *mockConnection) Close() error {
	// Locking makes closing idempotent and prevents a concurrent Send from
	// observing an inconsistent isOpen value.
	c.mu.Lock()
	if !c.isOpen {
		c.mu.Unlock()
		return nil
	}
	c.isOpen = false
	c.mu.Unlock()

	// Closing the channel wakes every goroutine waiting in Recv.
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}

type mockNetwork struct {
	// mu protects listeners, partitions, and network settings shared by sends
	// and configuration changes from different goroutines.
	mu         sync.RWMutex
	listeners  map[Address]*mockConnection
	partitions []struct{ a, b Address }
	packetLoss float64
	latency    time.Duration
}

func NewMockNetwork() *mockNetwork {
	return &mockNetwork{listeners: make(map[Address]*mockConnection)}
}

func (n *mockNetwork) Listen(addr Address) (Connection, error) {
	// Only one goroutine may register or inspect a listener at a time here.
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.listeners[addr]; exists {
		return nil, fmt.Errorf("address already in use: %s", addr.String())
	}

	conn := &mockConnection{
		network: n,
		addr:    addr,
		// The buffered queue absorbs short bursts so senders do not block on a
		// receiver that is briefly processing another message.
		recv: make(chan Message, 256),
		// This channel is used only as a close notification, so it carries no
		// values and costs little to broadcast shutdown.
		closed: make(chan struct{}),
		isOpen: true,
	}
	n.listeners[addr] = conn //Registers the connection in the network’s address map.
	return conn, nil
}

func (n *mockNetwork) Dial(addr Address) (Connection, error) {
	// A read lock permits concurrent dials while protecting the listener map
	// from a simultaneous Listen or network reconfiguration.
	n.mu.RLock()
	_, exists := n.listeners[addr]
	n.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unknown address: %s", addr.String())
	}

	return &mockConnection{
		network: n,
		addr:    addr,
		// A dialed connection uses a private queue for its sender-side handle;
		// delivery is routed to the destination listener in send.
		recv:   make(chan Message, 1),
		closed: make(chan struct{}),
		isOpen: true,
	}, nil
}

func (n *mockNetwork) SetPacketLoss(probability float64) {
	// Configuration writes are locked so sends cannot read a partially updated
	// packet-loss value.
	n.mu.Lock()
	defer n.mu.Unlock()
	if probability < 0 {
		probability = 0
	}
	if probability > 1 {
		probability = 1
	}
	n.packetLoss = probability
}

func (n *mockNetwork) SetLatency(delay time.Duration) {
	// Protect latency because it is read by send while tests may change it.
	n.mu.Lock()
	defer n.mu.Unlock()
	n.latency = delay
}

func (n *mockNetwork) Partition(group1, group2 []Address) {
	// Partition updates the shared link rules atomically with respect to sends.
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, a := range group1 {
		for _, b := range group2 {
			n.partitions = append(n.partitions, struct{ a, b Address }{a: a, b: b})
		}
	}
}

func (n *mockNetwork) Heal() {
	// Clearing partitions under the lock prevents a send from seeing a partial
	// partition configuration.
	n.mu.Lock()
	defer n.mu.Unlock()
	n.partitions = nil
}

func (n *mockNetwork) send(msg Message) error {
	// Capture the routing state under a read lock, then release it before any
	// delivery delay so unrelated network operations can continue concurrently.
	n.mu.RLock()
	packetLoss := n.packetLoss
	latency := n.latency
	for _, p := range n.partitions {
		if (p.a == msg.From && p.b == msg.To) || (p.b == msg.From && p.a == msg.To) {
			n.mu.RUnlock()
			return fmt.Errorf("network partition blocks traffic between %s and %s", msg.From.String(), msg.To.String())
		}
	}
	conn, exists := n.listeners[msg.To]
	n.mu.RUnlock()
	if !exists {
		return fmt.Errorf("destination is not listening: %s", msg.To.String())
	}

	if packetLoss > 0 && rand.Float64() < packetLoss {
		return fmt.Errorf("packet loss between %s and %s", msg.From.String(), msg.To.String())
	}

	conn.mu.RLock()
	isOpen := conn.isOpen
	conn.mu.RUnlock()
	if !isOpen {
		return fmt.Errorf("destination connection closed: %s", msg.To.String())
	}

	if latency > 0 {
		// AfterFunc schedules delayed delivery in its own goroutine, which models
		// network latency without blocking the caller's send operation.
		time.AfterFunc(latency, func() {
			// The non-blocking select drops a packet when the destination queue is
			// full instead of stalling the simulated network.
			select {
			case conn.recv <- msg:
			default:
			}
		})
		return nil
	}

	// The non-blocking send models a bounded receive queue: a full queue makes
	// delivery fail rather than blocking every sender.
	select {
	case conn.recv <- msg:
		return nil
	default:
		return fmt.Errorf("destination queue full: %s", msg.To.String())
	}
}
