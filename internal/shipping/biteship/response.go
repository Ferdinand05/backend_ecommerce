package biteship

type searchAreasAPIResponse struct {
	Success bool            `json:"success"`
	Areas   []areaAPIResult `json:"areas"`
}

type areaAPIResult struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PostalCode int    `json:"postal_code"`
}
