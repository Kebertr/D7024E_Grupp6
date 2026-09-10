package rpc

import (
	"net"
	"testing"

	"github.com/RasmusKebert/d7024e-tutorial/internal/kademlia"
)

func TestSendPingBadAddressDestination(t *testing.T) {

	resp := sendPing("127.0.0.1:8080", "127.0.0.1:-1")

	if resp {
		t.Errorf("sendPing should have failed with bad destination address")
	}
}

func TestSendPingBadAddressSource(t *testing.T) {

	resp := sendPing("127.0.0.1:-1", "127.0.0.1:8081")

	if resp {
		t.Errorf("sendPing should have failed with bad source address")
	}
}

func TestSendPingDeadlineExceeded(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:8081")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp := sendPing("127.0.0.1:8080", "127.0.0.1:8081")
	if resp {
		t.Errorf("sendPing should have failed with deadline exceeded")
	}
}

func TestSendPingWrongReturnSize(t *testing.T) {
	node1 := kademlia.NewContact(kademlia.NewRandomKademliaID(), "127.0.0.1:8080")

	node2 := kademlia.NewContact(kademlia.NewRandomKademliaID(), "127.0.0.1:8081")

	conn, err := net.ListenPacket("udp", node2.Address)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	go func() {
		_, sender, err := conn.ReadFrom(make([]byte, 16))
		if err != nil {
			t.Error(err)
		}
		conn.WriteTo([]byte("WrongTest"), sender)
	}()

	res := sendPing(node1.Address, node2.Address)

	if res {
		t.Error("It should not be true since the return is wrong size")
	}
}

func TestConfirmPingWrongReturnSize(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:8081")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	confirmed := make(chan bool, 1)

	go func() {
		confirmed <- confirmPing(conn)
	}()

	fromAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
	if err != nil {
		t.Fatal(err)
	}

	fullAddress, err := net.ResolveUDPAddr("udp", "127.0.0.1:8081")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := net.DialUDP("udp", fromAddr, fullAddress)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Close()

	_, err = resp.Write([]byte("TestWrong"))
	if err != nil {
		t.Fatal(err)
	}

	if <-confirmed {
		t.Error("It should not be true since the return is wrong size")
	}
}

func TestConfirmPingDeadlineExceeded(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:8081")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	res := confirmPing(conn)

	if res {
		t.Error("The deadline should have been exceeded here. Since it will wait")
	}
}
func TestSendPing(t *testing.T) {
	node1 := kademlia.NewContact(kademlia.NewRandomKademliaID(), "127.0.0.1:8080")

	node2 := kademlia.NewContact(kademlia.NewRandomKademliaID(), "127.0.0.1:8081")

	conn, err := net.ListenPacket("udp", node2.Address)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	confirmed := make(chan bool, 1)

	go func() {
		confirmed <- confirmPing(conn)
	}()

	success := sendPing(node1.Address, node2.Address)

	if !success {
		t.Errorf("sendPing failed")
	}

	if !<-confirmed {
		t.Errorf("confirmPing failed")
	}
}
