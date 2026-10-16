package shipping

type Area struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PostalCode int    `json:"postal_code"`
}

type SearchAreasResponse struct {
	Areas []Area `json:"areas"`
}

type CalculateRatesRequest struct {
	DestinationAreaID string             `json:"destination_area_id" binding:"required"`
	Items             []ShippingRateItem `json:"items" binding:"required,min=1,dive"`
}

type ShippingRateItem struct {
	ProductVariantID string `json:"product_variant_id" binding:"required"`
	Quantity         int    `json:"quantity" binding:"required,min=1"`
}

type RatesProviderRequest struct {
	OriginAreaID      string
	DestinationAreaID string
	Couriers          string
	Items             []RatesProviderItem
}

type RatesProviderItem struct {
	Name     string
	Value    int64
	Weight   int
	Quantity int
}

type ShippingRate struct {
	CourierCode           string `json:"courier_code"`
	CourierName           string `json:"courier_name"`
	CourierServiceCode    string `json:"courier_service_code"`
	CourierServiceName    string `json:"courier_service_name"`
	Description           string `json:"description"`
	Price                 int64  `json:"price"`
	ShipmentDurationRange string `json:"shipment_duration_range"`
	ShipmentDurationUnit  string `json:"shipment_duration_unit"`
}

type CalculateRatesResponse struct {
	Rates []ShippingRate `json:"rates"`
}
