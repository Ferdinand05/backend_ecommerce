package metrics

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

// SummaryResponse ringkasan metrik dashboard untuk admin.
type SummaryResponse struct {
	TotalRevenue  decimal.Decimal `json:"total_revenue"`
	ActiveOrders  int64           `json:"active_orders"`
	NewUsers      int64           `json:"new_users"`
	LowStockCount int64           `json:"low_stock_count"`
}

// MarshalJSON menulis total_revenue sebagai angka (bukan string) agar frontend
// bisa langsung memakainya di chart.
func (s SummaryResponse) MarshalJSON() ([]byte, error) {

	revenue := s.TotalRevenue
	if revenue.IsZero() {
		revenue = decimal.Zero
	}

	return json.Marshal(struct {
		TotalRevenue  json.RawMessage `json:"total_revenue"`
		ActiveOrders  int64           `json:"active_orders"`
		NewUsers      int64           `json:"new_users"`
		LowStockCount int64           `json:"low_stock_count"`
	}{
		TotalRevenue:  json.RawMessage(revenue.String()),
		ActiveOrders:  s.ActiveOrders,
		NewUsers:      s.NewUsers,
		LowStockCount: s.LowStockCount,
	})
}
