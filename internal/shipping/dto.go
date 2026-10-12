package shipping

type Area struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PostalCode int    `json:"postal_code"`
}

type SearchAreasResponse struct {
	Areas []Area `json:"areas"`
}
