package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.svc.Register(ctx, req)
	if err != nil {
		if errors.Is(err, ErrorEmailExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.svc.Login(ctx, req)
	if err != nil {
		if errors.Is(err, ErrorInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) VerifyEmail(c *gin.Context) {

	var req VerifyEmailRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid request",
		})
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	if err := h.svc.VerifyEmail(ctx, req.Token); err != nil {
		switch {
		case errors.Is(err, ErrorInvalidVerificationToken):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid verification token",
			})

		case errors.Is(err, ErrorVerificationTokenExpired):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "verification token expired",
			})

		case errors.Is(err, ErrorEmailAlreadyVerified):
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already verified",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to verify email",
			})
		}

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"message": "email verified successfully",
		},
	)
}
