package metrics

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func TestSummaryResponseJSONShape(t *testing.T) {
	data, err := json.Marshal(SummaryResponse{
		TotalRevenue:  decimal.NewFromInt(150000),
		ActiveOrders:  3,
		NewUsers:      2,
		LowStockCount: 1,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"total_revenue", "active_orders", "new_users", "low_stock_count"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing key %q in %s", key, data)
		}
	}

	if _, ok := got["total_revenue"].(float64); !ok {
		t.Fatalf("total_revenue = %T, want number (body: %s)", got["total_revenue"], data)
	}
}
