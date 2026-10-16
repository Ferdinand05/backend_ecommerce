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

// rates

type ratesAPIRequest struct {
	OriginAreaID      string         `json:"origin_area_id"`
	DestinationAreaID string         `json:"destination_area_id"`
	Couriers          string         `json:"couriers"`
	Items             []ratesAPIItem `json:"items"`
}

type ratesAPIItem struct {
	Name     string `json:"name"`
	Value    int64  `json:"value"`
	Weight   int    `json:"weight"`
	Quantity int    `json:"quantity"`
}

type ratesAPIResponse struct {
	Success bool              `json:"success"`
	Pricing []ratesAPIPricing `json:"pricing"`
}

type ratesAPIPricing struct {
	CourierCode           string `json:"courier_code"`
	CourierName           string `json:"courier_name"`
	CourierServiceCode    string `json:"courier_service_code"`
	CourierServiceName    string `json:"courier_service_name"`
	Description           string `json:"description"`
	Price                 int64  `json:"price"`
	ShipmentDurationRange string `json:"shipment_duration_range"`
	ShipmentDurationUnit  string `json:"shipment_duration_unit"`
}
