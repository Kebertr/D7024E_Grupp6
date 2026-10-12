package package_registry

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"google.golang.org/protobuf/proto"
)

func (r *Registry) Publish(domain string, packageUse string, version string, file []byte, force bool, previousVersion string, privateKey rsa.PrivateKey) error {
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

func (r *Registry) Install(domain string, packageUse string, version string) ([]byte, error) {
	pointer, err := r.getLatestPointer(domain, packageUse)
	if err != nil {
		return []byte{}, err
	}

	var previous []byte

	if version == "latest" {
		previous = pointer.versionRecordHash
	} else {
		previousAns, errPrev := r.FindRightVersion(domain, packageUse, version, *pointer)

		if errPrev != nil {
			return nil, errPrev
		}

		previous = previousAns
	}

	recordBytes, _, err := r.Kademlia.LookupData(
		hex.EncodeToString(previous),
	)
	if err != nil {
		return nil, err
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

func (r *Registry) ShowVersionPackage(domain string, packageUse string) (string, error) {
	res, err := r.getLatestPointer(domain, packageUse)
	if err != nil {
		return "", err
	}

	return res.version, nil

}

func (r *Registry) ShowDNS(domain string) (rsa.PublicKey, error) {
	return r.DNS.lookup(domain)
}
