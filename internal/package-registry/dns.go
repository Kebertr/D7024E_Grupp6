package package_registry

type DNS interface {
	Registery() error
	lookup() (DNSTXT, error)
}

type DNSTXT struct {
	name      string
	publicKey string
}
