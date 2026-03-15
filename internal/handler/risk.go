package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/silentpass/silentpass/internal/model"
	"github.com/silentpass/silentpass/internal/service/risk"
)

type RiskHandler struct {
	svc *risk.Service
}

func NewRiskHandler(svc *risk.Service) *RiskHandler {
	return &RiskHandler{svc: svc}
}

// SIMSwap handles POST /v1/risk/sim-swap
func (h *RiskHandler) SIMSwap(c *gin.Context) {
	var req model.SIMSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	resp, err := h.svc.CheckSIMSwap(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SIM swap check failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Verdict handles POST /v1/risk/verdict
func (h *RiskHandler) Verdict(c *gin.Context) {
	var req model.VerdictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	resp, err := h.svc.EvaluateVerdict(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verdict evaluation failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}
