package package_registry

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
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
	tagByte, err := json.Marshal(vers.tag)
	if err != nil {
		return [32]byte{}, err
	}
	domainNameByte, err := json.Marshal(vers.domainName)
	if err != nil {
		return [32]byte{}, err
	}
	packageNameByte, err := json.Marshal(vers.packageName)
	if err != nil {
		return [32]byte{}, err
	}
	versionByte, err := json.Marshal(vers.version)
	if err != nil {
		return [32]byte{}, err
	}

	data := make([]byte, 0)
	data = append(data, tagByte...)
	data = append(data, domainNameByte...)
	data = append(data, packageNameByte...)
	data = append(data, versionByte...)
	data = append(data, vers.blobHash...)
	data = append(data, vers.previousVersionRecord...)

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
