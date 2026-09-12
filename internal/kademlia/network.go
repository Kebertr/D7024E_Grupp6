package kademlia

type Network struct {
}

func Listen(ip string, port int) {
	// TODO
}

func (network *Network) SendPingMessage(contact *Contact) {
	// TODO
}

func (network *Network) SendFindContactMessage(contact *Contact, target *KademliaID) ([]Contact, error) {
	// TODO FIX
	// It returns no contacts, so the lookup can only inspect the contacts
	// already in the local routing table. It will not discover remote contacts.
	return nil, nil // Temporary so lookup does not generate errors
}

func (network *Network) SendFindDataMessage(hash string) {
	// TODO
}

func (network *Network) SendStoreMessage(data []byte) {
	// TODO
}
