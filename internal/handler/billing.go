package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type BillingEntry struct {
	Product    string  `json:"product"`
	Country    string  `json:"country"`
	Calls      int     `json:"calls"`
	Successful int     `json:"successful"`
	UnitPrice  float64 `json:"unit_price"`
	Total      float64 `json:"total"`
}

type BillingHandler struct{}

func NewBillingHandler() *BillingHandler {
	return &BillingHandler{}
}

// Summary handles GET /v1/billing/summary
func (h *BillingHandler) Summary(c *gin.Context) {
	// In production, this queries pg_billing via PGBillingRepo.SummaryByTenant
	entries := []BillingEntry{
		{Product: "Silent Verification", Country: "ID", Calls: 45200, Successful: 38420, UnitPrice: 0.03, Total: 1152.60},
		{Product: "Silent Verification", Country: "TH", Calls: 32100, Successful: 28569, UnitPrice: 0.035, Total: 999.92},
		{Product: "SMS OTP", Country: "ID", Calls: 6780, Successful: 6510, UnitPrice: 0.045, Total: 292.95},
		{Product: "SMS OTP", Country: "TH", Calls: 3531, Successful: 3390, UnitPrice: 0.05, Total: 169.50},
		{Product: "SIM Swap Check", Country: "ID", Calls: 12400, Successful: 12400, UnitPrice: 0.01, Total: 124.00},
		{Product: "SIM Swap Check", Country: "TH", Calls: 8900, Successful: 8900, UnitPrice: 0.012, Total: 106.80},
		{Product: "WhatsApp OTP", Country: "ID", Calls: 1200, Successful: 1140, UnitPrice: 0.06, Total: 68.40},
	}

	var totalCost float64
	for _, e := range entries {
		totalCost += e.Total
	}

	c.JSON(http.StatusOK, gin.H{
		"entries":      entries,
		"total_cost":   totalCost,
		"period":       "2026-03",
		"currency":     "USD",
	})
}
