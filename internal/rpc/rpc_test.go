package rpc

import (
	"testing"
	"time"

	"github.com/RasmusKebert/d7024e-tutorial/internal/kademlia"
)

func TestSendPing(t *testing.T) {
	node1 := kademlia.Node{
		ID:   kademlia.ID{},
		Addr: "127.0.0.1",
		Port: 8080,
	}

	node2 := kademlia.Node{
		ID:   kademlia.ID{},
		Addr: "127.0.0.1",
		Port: 8081,
	}

	confirmed := make(chan bool, 1)

	go func() {
		confirmed <- confirmPing(node2.Addr, node2.Port)
	}()

	time.Sleep(100 * time.Millisecond)

	success := sendPing(node1.Addr, node1.Port, node2.Addr, node2.Port)

	if !success {
		t.Errorf("sendPing failed")
	}

	if !<-confirmed {
		t.Errorf("confirmPing failed")
	}

}
