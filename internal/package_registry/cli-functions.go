package package_registry

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
	"google.golang.org/protobuf/proto"
)

func (r *registry) publish(domain string, packageUse string, version string, file []byte, force bool, previousVersion string, privateKey rsa.PrivateKey) error {
	errStore := r.Kademlia.Store(file)

	if errStore != nil {
		return errStore
	}

	blob := sha256.Sum256(file)

	pointer, errGetPointer := r.getLatestPointer(domain, packageUse)

	var oldVers *VersionRecord
	var previous []byte
	if errGetPointer != nil {
		if errGetPointer.Error() == "data not found" {
			if previousVersion != "" {
				return errors.New("Conno tget previous version if this is the first")
			}
			//If there is no previous in latestPointer
			previous = make([]byte, sha256.Size)
		} else {
			return errGetPointer
		}
	} else {

		firstPrev, _, err := r.Kademlia.LookupData(
			hex.EncodeToString(pointer.versionRecordHash),
		)

		if err != nil {
			return err
		}

		var recordProto VersionRecordProto

		errProto := proto.Unmarshal(firstPrev, &recordProto)

		if errProto != nil {
			return errProto
		}

		oldVers = &VersionRecord{
			tag:                   recordProto.Tag,
			domainName:            recordProto.DomainName,
			packageName:           recordProto.PackageName,
			version:               recordProto.Version,
			blobHash:              recordProto.BlobHash,
			previousVersionRecord: recordProto.PreviousVersionRecord,
			sig:                   recordProto.Sig,
		}

		if previousVersion == "" {
			previous = pointer.versionRecordHash
		} else {
			previousAns, err := r.FindRightVersion(domain, packageUse, previousVersion, *pointer)

			if err != nil {
				return errors.New("Problem with going down and finding the version")
			}
			previous = previousAns
		}
	}

	newVers := &VersionRecord{
		tag:                   "version-record",
		domainName:            domain,
		packageName:           packageUse,
		version:               version,
		blobHash:              blob[:],
		previousVersionRecord: previous,
	}

	errSign := newVers.sign(&privateKey)

	if errSign != nil {
		return errSign
	}

	if !force && oldVers != nil {
		errComp := r.CompareVersionRecords(newVers, oldVers)

		if errComp != nil {
			return errComp
		}
	}

	errSet := r.setLatestPointer(newVers, &privateKey)

	if errSet != nil {
		return errSet
	}
	return nil
}

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

func ShowVersionPackage(node *kademlia.Kademlia, dns DNS, domain, packageUse string) (string, error) {
	reg := &registry{
		Kademlia: node,
		DNS:      dns,
	}
	return reg.showVersionPackage(domain, packageUse)
}

func ShowDNS(node *kademlia.Kademlia, dns DNS, domain string) (rsa.PublicKey, error) {
	reg := &registry{
		Kademlia: node,
		DNS:      dns,
	}
	return reg.showDNS(domain)
}
