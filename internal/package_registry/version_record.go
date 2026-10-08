package package_registry

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"

	"google.golang.org/protobuf/proto"
)

type VersionRecord struct {
	tag                   string
	domainName            string
	packageName           string
	version               string
	blobHash              []byte
	previousVersionRecord []byte
	sig                   []byte
}

func (vers *VersionRecord) hashingValues() (hash [32]byte, err error) {
	vsRecord := &VersionRecordProto{
		Tag:                   vers.tag,
		DomainName:            vers.domainName,
		PackageName:           vers.packageName,
		Version:               vers.version,
		BlobHash:              vers.blobHash,
		PreviousVersionRecord: vers.previousVersionRecord,
	}
	data, err := proto.Marshal(vsRecord)
	if err != nil {
		return [32]byte{}, err
	}

	hash = sha256.Sum256(data)
	return hash, nil
}
func (vers *VersionRecord) sign(privateKey *rsa.PrivateKey) error {

	hash, err := vers.hashingValues()
	if err != nil {
		return err
	}
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		panic("Our digital pen ran out of ink!, I took this message from a guide lol")
	}
	vers.sig = signature
	return nil
}

func (vers *VersionRecord) verifySignature(publicKey *rsa.PublicKey) error {
	hash, err := vers.hashingValues()
	if err != nil {
		return err
	}
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], vers.sig)
	if err != nil {
		return err
	}
	return nil
}

func (vers *VersionRecord) hashBlob(blob []byte) error {
	hash := sha256.Sum256(blob)
	vers.blobHash = hash[:]
	return nil
}

func (vers *VersionRecord) hashOldRecord(oldRecord *VersionRecord) error {
	vsRecord := &VersionRecordProto{
		Tag:                   oldRecord.tag,
		DomainName:            oldRecord.domainName,
		PackageName:           oldRecord.packageName,
		Version:               oldRecord.version,
		BlobHash:              oldRecord.blobHash,
		PreviousVersionRecord: oldRecord.previousVersionRecord,
		Sig:                   oldRecord.sig,
	}
	data, err := proto.Marshal(vsRecord)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(data)

	vers.previousVersionRecord = hash[:]
	return nil
}
