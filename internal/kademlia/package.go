package kademlia

import (
	"bytes"
	"errors"
	"strings"

	"google.golang.org/protobuf/proto"
)

type PackageSpec struct {
	Domain  string
	Package string
	Version string
}

func ZeroHash() []byte {
	return make([]byte, IDLength)
}

func NewVersionRecord(
	spec PackageSpec,
	blob []byte,
	previousRecordHash []byte,
) *VersionRecord {
	blobHash := HashBytes(blob)

	return &VersionRecord{
		Domain:             spec.Domain,
		Package:            spec.Package,
		Version:            spec.Version,
		BlobHash:           append([]byte(nil), blobHash[:]...),
		PreviousRecordHash: append([]byte(nil), previousRecordHash...),
	}
}

func NewLatestPointer(
	spec PackageSpec,
	versionRecordHash *KademliaID,
) *LatestPointer {
	return &LatestPointer{
		Domain:            spec.Domain,
		Package:           spec.Package,
		Version:           spec.Version,
		VersionRecordHash: append([]byte(nil), versionRecordHash[:]...),
	}
}

func ParsePackageSpec(value string) (PackageSpec, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return PackageSpec{}, errors.New(
			"expected DOMAIN:PACKAGE:VERSION",
		)
	}

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return PackageSpec{}, errors.New(
				"domain, package, and version cannot be empty",
			)
		}
	}

	return PackageSpec{
		Domain:  parts[0],
		Package: parts[1],
		Version: parts[2],
	}, nil
}

func LatestKey(spec PackageSpec) *KademliaID {
	name := spec.Domain + ":" + spec.Package + ":latest"
	return NewValueID([]byte(name))
}

func HashBytes(data []byte) *KademliaID {
	return NewValueID(data)
}

func MarshalVersionRecord(
	record *VersionRecord,
) ([]byte, error) {
	if record == nil {
		return nil, errors.New("version record is nil")
	}

	wrapper := &RegistryRecord{
		Record: &RegistryRecord_VersionRecord{
			VersionRecord: record,
		},
	}

	return proto.MarshalOptions{
		Deterministic: true,
	}.Marshal(wrapper)
}

func MarshalLatestPointer(
	pointer *LatestPointer,
) ([]byte, error) {
	if pointer == nil {
		return nil, errors.New("latest pointer is nil")
	}

	wrapper := &RegistryRecord{
		Record: &RegistryRecord_LatestPointer{
			LatestPointer: pointer,
		},
	}

	return proto.MarshalOptions{
		Deterministic: true,
	}.Marshal(wrapper)
}

func VersionRecordHash(record *VersionRecord) (*KademliaID, error) {
	data, err := MarshalVersionRecord(record)
	if err != nil {
		return nil, err
	}
	return HashBytes(data), nil
}

func LatestPointerHash(pointer *LatestPointer) (*KademliaID, error) {
	data, err := MarshalLatestPointer(pointer)
	if err != nil {
		return nil, err
	}
	return HashBytes(data), nil
}

func VerifyBlobHash(blob []byte, expected []byte) bool {
	actual := HashBytes(blob)
	return bytes.Equal(actual[:], expected)
}
