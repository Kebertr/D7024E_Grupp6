package package_registry

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
	"google.golang.org/protobuf/proto"
)

type registry struct {
	Kademlia *kademlia.Kademlia
	DNS      DNS
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

	publicKey, dnsErr := r.DNS.lookup(vers.domainName)

	if dnsErr != nil {
		return dnsErr
	}

	point := &latestPointer{
		tag:               "latest-pointer",
		domainName:        vers.domainName,
		packageName:       vers.packageName,
		version:           vers.version,
		versionRecordHash: recordHash[:],
		sig:               signature,
	}

	ver := point.verifySignaturePointer(&publicKey)

	if ver != nil {
		return ver
	}

	fullName := vers.domainName + ":" + vers.packageName + ":latest"

	storeErr := r.Kademlia.StorePart2([]byte(fullName), bytePointer)

	if storeErr != nil {
		return storeErr
	}
	return nil
}

func (point *latestPointer) verifySignaturePointer(publicKey *rsa.PublicKey) error {
	vsRecord := &LatestPointerProto{
		Tag:               point.tag,
		DomainName:        point.domainName,
		PackageName:       point.packageName,
		Version:           point.version,
		VersionRecordHash: point.versionRecordHash,
	}
	data, err := proto.Marshal(vsRecord)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(data)

	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], point.sig)
	if err != nil {
		return err
	}
	return nil
}

func compareVersions(newVersion string, oldVersion string) (bool, error) {
	newInts := strings.Split(newVersion, ".")
	oldInts := strings.Split(oldVersion, ".")

	if len(newInts) != len(oldInts) {
		return false, errors.New("The versions should be equal in size")
	}

	for i := 0; i < len(newInts); i++ {
		newNum, err := strconv.Atoi(newInts[i])
		if err != nil {
			return false, err
		}

		oldNum, err := strconv.Atoi(oldInts[i])
		if err != nil {
			return false, err
		}

		if newNum > oldNum {
			return true, nil
		}

		if newNum < oldNum {
			return false, nil
		}
	}

	//Here they are equal
	return false, nil
}

func (r *registry) CompareVersionRecords(newVers *VersionRecord, oldVers *VersionRecord) error {

	if newVers.domainName != oldVers.domainName || newVers.packageName != oldVers.packageName {
		return errors.New("The domainame and package name need to be the same")
	}
	okVersion, errVersion := compareVersions(newVers.version, oldVers.version)

	if errVersion != nil {
		return errVersion
	}

	if !okVersion {
		return errors.New("The version needs to be newer")
	}

	vsRecord := &VersionRecordProto{
		Tag:                   oldVers.tag,
		DomainName:            oldVers.domainName,
		PackageName:           oldVers.packageName,
		Version:               oldVers.version,
		BlobHash:              oldVers.blobHash,
		PreviousVersionRecord: oldVers.previousVersionRecord,
		Sig:                   oldVers.sig,
	}

	data, err := proto.Marshal(vsRecord)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(data)

	check := newVers

	//For handling forks and catch up. Check the previous records for the version
	for {
		if check.domainName != oldVers.domainName || check.packageName != oldVers.packageName {
			return errors.New("The domainame and package name need to be the same")
		}
		//To check if it is the first version
		zeroHash := make([]byte, sha256.Size)

		if bytes.Equal(check.previousVersionRecord, zeroHash) {
			return errors.New("reached first version")
		}

		publicKey, errPublic := r.DNS.lookup(check.domainName)

		if errPublic != nil {
			return errPublic
		}

		errVerify := check.verifySignature(&publicKey)

		if errVerify != nil {
			return errVerify
		}

		if bytes.Equal(check.previousVersionRecord, hash[:]) {
			return nil
		}

		firstPrev, _, err := r.Kademlia.LookupData(
			hex.EncodeToString(check.previousVersionRecord),
		)

		if err != nil {
			return err
		}

		var recordProto VersionRecordProto

		errProto := proto.Unmarshal(firstPrev, &recordProto)

		if errProto != nil {
			return errProto
		}

		prev := &VersionRecord{
			tag:                   recordProto.Tag,
			domainName:            recordProto.DomainName,
			packageName:           recordProto.PackageName,
			version:               recordProto.Version,
			blobHash:              recordProto.BlobHash,
			previousVersionRecord: recordProto.PreviousVersionRecord,
			sig:                   recordProto.Sig,
		}

		okVersion, errVersion := compareVersions(check.version, prev.version)

		if errVersion != nil {
			return errVersion
		}

		if !okVersion {
			return errors.New("The version needs to be newer")
		}

		check = prev

	}
}
