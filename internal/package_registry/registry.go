package package_registry

import (
	"crypto/rsa"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
)

type registry struct {
	Kademlia kademlia.Kademlia
	DNS      DNS
}

func (r *registry) showDNS(domain string) (rsa.PublicKey, error) {
	return r.DNS.lookup(domain)
}
