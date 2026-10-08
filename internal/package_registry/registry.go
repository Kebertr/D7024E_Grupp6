package package_registry

import (
	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
)

type registry struct {
	Kademlia kademlia.Kademlia
	DNS      DNS
}
