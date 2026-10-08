package package_registry

import "crypto/rsa"

type DNS interface {
	lookup(domain string) (rsa.PublicKey, error)
}
