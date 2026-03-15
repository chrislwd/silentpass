package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/silentpass/silentpass/internal/model"
	"github.com/silentpass/silentpass/internal/service/verification"
)

type VerificationHandler struct {
	svc *verification.Service
}

func NewVerificationHandler(svc *verification.Service) *VerificationHandler {
	return &VerificationHandler{svc: svc}
}

// CreateSession handles POST /v1/verification/session
func (h *VerificationHandler) CreateSession(c *gin.Context) {
	var req model.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	resp, err := h.svc.CreateSession(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// SilentVerify handles POST /v1/verification/silent
func (h *VerificationHandler) SilentVerify(c *gin.Context) {
	var req model.SilentVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	resp, err := h.svc.SilentVerify(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SendOTP handles POST /v1/verification/otp/send
func (h *VerificationHandler) SendOTP(c *gin.Context) {
	var req model.OTPSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	resp, err := h.svc.SendOTP(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send OTP"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CheckOTP handles POST /v1/verification/otp/check
func (h *VerificationHandler) CheckOTP(c *gin.Context) {
	var req model.OTPCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	resp, err := h.svc.CheckOTP(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OTP check failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}
