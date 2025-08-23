package kemono

type UserProfile struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Service  string  `json:"service"`
	PublicID *string `json:"public_id,omitempty"`
}
