package package_registry

import "crypto/rsa"

type registry struct {
	Kademlia *Kademlia,
	DNS	 DNS,
	privateKeys map[String] rsa.PrivateKey
}