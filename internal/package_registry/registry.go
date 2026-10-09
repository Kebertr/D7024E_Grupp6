package package_registry

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
	"google.golang.org/protobuf/proto"
)

type registry struct {
	Kademlia *kademlia.Kademlia
	DNS      DNS
}

func (r *registry) showDNS(domain string) (rsa.PublicKey, error) {
	return r.DNS.lookup(domain)
}

func (r *registry) getLatestPointer(domain string, packageName string) (*latestPointer, error) {
	fullName := domain + ":" + packageName + ":latest"

	key := sha256.Sum256([]byte(fullName))

	stored, _, err := r.Kademlia.LookupData(hex.EncodeToString(key[:]))

	if err != nil {
		return nil, err
	}

	var latestProto LatestPointerProto

	errProto := proto.Unmarshal(stored, &latestProto)

	if errProto != nil {
		return nil, errProto
	}

	pointer := &latestPointer{
		tag:               latestProto.Tag,
		domainName:        latestProto.DomainName,
		packageName:       latestProto.PackageName,
		version:           latestProto.Version,
		versionRecordHash: latestProto.VersionRecordHash,
		sig:               latestProto.Sig,
	}

	return pointer, nil

}

func (r *registry) setLatestPointer(vers *VersionRecord, privateKey *rsa.PrivateKey) error {

	vsRecord := &VersionRecordProto{
		Tag:                   vers.tag,
		DomainName:            vers.domainName,
		PackageName:           vers.packageName,
		Version:               vers.version,
		BlobHash:              vers.blobHash,
		PreviousVersionRecord: vers.previousVersionRecord,
		Sig:                   vers.sig,
	}

	recordData, errRecord := proto.Marshal(vsRecord)
	if errRecord != nil {
		return errRecord
	}

	recordHash := sha256.Sum256(recordData)

	versErr := r.Kademlia.Store(recordData)

	if versErr != nil {
		return versErr
	}

	pointProto := &LatestPointerProto{
		Tag:               "latest-pointer",
		DomainName:        vers.domainName,
		PackageName:       vers.packageName,
		Version:           vers.version,
		VersionRecordHash: recordHash[:],
	}

	data, err := proto.Marshal(pointProto)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(data)

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return err
	}

	pointProto.Sig = signature

	bytePointer, err := proto.Marshal(pointProto)
	if err != nil {
		return err
	}

	fullName := vers.domainName + ":" + vers.packageName + ":latest"

	storeErr := r.Kademlia.StorePart2([]byte(fullName), bytePointer)

	if storeErr != nil {
		return storeErr
	}
	return nil
}
