package main

import (
	"fmt"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
)

var (
	BuildVersion string = ""
	BuildTime    string = ""
)

func main() {
	fmt.Println("Pretending to run the kademlia app...")
	// Using stuff from the kademlia package here. Something like...
	id := kademlia.NewKademliaID("FFFFFFFF00000000000000000000000000000000")
	contact := kademlia.NewContact(id, "localhost:8000")
	fmt.Println(contact.String())
	fmt.Printf("%v\n", contact)
}
