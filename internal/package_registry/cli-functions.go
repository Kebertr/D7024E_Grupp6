package package_registry

import (
	"crypto/rsa"
	"encoding/hex"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
	"google.golang.org/protobuf/proto"
)

func (r *registry) install(domain string, packageUse string, version string) ([]byte, error) {
	pointer, err := r.getLatestPointer(domain, packageUse)
	if err != nil {
		return []byte{}, err
	}

	recordBytes, _, err := r.Kademlia.LookupData(
		hex.EncodeToString(pointer.versionRecordHash),
	)
	if err != nil {
		return []byte{}, err
	}

	var recordProto VersionRecordProto

	errProto := proto.Unmarshal(recordBytes, &recordProto)

	if errProto != nil {
		return []byte{}, errProto
	}

	blobHash := recordProto.BlobHash

	res, _, err := r.Kademlia.LookupData(
		hex.EncodeToString(blobHash),
	)
	if err != nil {
		return []byte{}, err
	}

	return res, nil

}

func (r *registry) showVersionPackage(domain string, packageUse string) (string, error) {
	res, err := r.getLatestPointer(domain, packageUse)
	if err != nil {
		return "", err
	}

	return res.version, nil

}

func (r *registry) showDNS(domain string) (rsa.PublicKey, error) {
	return r.DNS.lookup(domain)
}

// Wrappers
// Tried to change publicity to public did not work beacuse go is shit

func Install(node *kademlia.Kademlia, dns DNS, domain, packageUse, version string) ([]byte, error) {
	reg := &registry{
		Kademlia: node,
		DNS:      dns,
	}
	return reg.install(domain, packageUse, version)
}
