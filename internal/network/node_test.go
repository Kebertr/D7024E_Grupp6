package network

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNodeBasicCommunication(t *testing.T) {
	network := NewMockNetwork()
	alice, err := NewNode(network, Address{IP: "127.0.0.1", Port: 8080})
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	bob, err := NewNode(network, Address{IP: "127.0.0.1", Port: 8081})
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}
	defer alice.Close()
	defer bob.Close()

	replyCh := make(chan string, 1)
	bob.Handle("hello", func(msg Message) error {
		return msg.ReplyString("reply", "hello back")
	})
	alice.Handle("reply", func(msg Message) error {
		replyCh <- string(msg.Payload)
		return nil
	})

	alice.Start()
	bob.Start()

	if err := alice.SendString(bob.Address(), "hello", "hi"); err != nil {
		t.Fatalf("send hello: %v", err)
	}

	select {
	case payload := <-replyCh:
		if !strings.Contains(payload, "reply:hello back") {
			t.Fatalf("unexpected payload: %q", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for reply")
	}
}

func TestNetworkPacketLossAndLatency(t *testing.T) {
	network := NewMockNetwork()
	network.SetPacketLoss(0)
	network.SetLatency(25 * time.Millisecond)

	alice, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9000})
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	bob, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9001})
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}
	defer alice.Close()
	defer bob.Close()

	recvCh := make(chan time.Time, 1)
	bob.Handle("ping", func(msg Message) error {
		return msg.ReplyString("pong", "ok")
	})
	alice.Handle("pong", func(msg Message) error {
		recvCh <- time.Now()
		return nil
	})

	alice.Start()
	bob.Start()

	start := time.Now()
	if err := alice.SendString(bob.Address(), "ping", "hi"); err != nil {
		t.Fatalf("send ping: %v", err)
	}

	select {
	case <-recvCh:
		if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
			t.Fatalf("expected simulated latency to delay delivery, got %v", elapsed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for delayed message")
	}

	network.SetPacketLoss(1.0)
	if err := alice.SendString(bob.Address(), "ping", "drop-me"); err == nil {
		t.Fatal("expected packet loss to fail the send")
	}
}

func TestMessageCorrelation(t *testing.T) {
	network := NewMockNetwork()
	alice, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9100})
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	bob, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9101})
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}
	defer alice.Close()
	defer bob.Close()

	bob.Handle("request", func(msg Message) error {
		if msg.RequestID == 0 {
			t.Fatal("missing request correlation id")
		}
		return msg.ReplyString("response", "ok")
	})

	respCh := make(chan uint64, 1)
	alice.Handle("response", func(msg Message) error {
		respCh <- msg.ResponseTo
		return nil
	})

	alice.Start()
	bob.Start()

	requestID := alice.nextID()
	msg := Message{
		From:      alice.addr,
		To:        bob.addr,
		Type:      "request",
		Payload:   []byte("request-data"),
		network:   network,
		RequestID: requestID,
	}
	if err := network.send(msg); err != nil {
		t.Fatalf("send request: %v", err)
	}

	select {
	case got := <-respCh:
		if got != requestID {
			t.Fatalf("expected response correlation id %d, got %d", requestID, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for response")
	}
}

func TestSimulatedNetworkWithOneThousandNodes(t *testing.T) {
	const nodeCount = 1000

	network := NewMockNetwork()
	nodes := make([]*Node, 0, nodeCount)
	seenPongs := make([]atomic.Uint32, nodeCount)

	for i := 0; i < nodeCount; i++ {
		addr := Address{IP: "127.0.0.1", Port: 10000 + i}
		node, err := NewNode(network, addr)
		if err != nil {
			t.Fatalf("create node %d: %v", i, err)
		}

		node.Handle("ping", func(msg Message) error {
			return msg.ReplyString("pong", "ok")
		})
		node.Handle("pong", func(msg Message) error {
			seenPongs[i].Add(1)
			return nil
		})

		nodes = append(nodes, node)
		node.Start()
	}

	for i := 0; i < nodeCount; i++ {
		target := (i + 1) % nodeCount
		if err := nodes[i].SendString(nodes[target].Address(), "ping", "hello"); err != nil {
			t.Fatalf("node %d failed to send to node %d: %v", i, target, err)
		}
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		allReceived := true
		for i := 0; i < nodeCount; i++ {
			if seenPongs[i].Load() == 0 {
				allReceived = false
				break
			}
		}
		if allReceived {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected all %d nodes to receive a pong response within timeout", nodeCount)
}
