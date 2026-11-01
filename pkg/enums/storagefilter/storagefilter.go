package storagefilter

// Mode controls how a user's storage list is interpreted.
type Mode string

const (
	// Blacklist lets the user access every storage except the listed names.
	Blacklist Mode = "blacklist"
	// Whitelist lets the user access only the listed storage names.
	Whitelist Mode = "whitelist"
)

func (m Mode) Valid() bool {
	return m == Blacklist || m == Whitelist
}

// Normalize returns a safe mode, defaulting to Blacklist so new users can use
// every configured storage.
func (m Mode) Normalize() Mode {
	if m == Whitelist {
		return Whitelist
	}
	return Blacklist
}
