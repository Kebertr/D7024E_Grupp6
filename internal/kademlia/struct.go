package kademlia

type ID [32]byte // Since we need 256 bits for the ID. 256/8 = 32

type Node struct {
	ID       ID
	Addr     string
	Port     int
	contacts []contacts
}

type contacts struct {
	ID   ID
	Addr string
	Port int
}
