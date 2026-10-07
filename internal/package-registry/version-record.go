package package_registry

type VersionRecord struct {
	tag                   string
	DomainName            string
	PackageName           string
	Version               string
	BlobHash              string
	PreviousVersionRecord string
	Sig                   []byte
}
