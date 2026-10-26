package auth

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/user"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// currentUserID membaca user id dari context yang diisi oleh
// AuthMiddleware. Nilainya disimpan sebagai string uuid.
func currentUserID(c *gin.Context) (uuid.UUID, bool) {

	raw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return uuid.Nil, false
	}

	value, ok := raw.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return uuid.Nil, false
	}

	userID, err := uuid.Parse(value)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return uuid.Nil, false
	}

	return userID, true
}

// GetMe mengembalikan profil user yang sedang login.
func (h *Handler) GetMe(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	profile, err := h.svc.GetMe(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrorUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateMe memperbarui profil user yang sedang login.
func (h *Handler) UpdateMe(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	profile, err := h.svc.UpdateMe(ctx, userID, req)
	if err != nil {
		if errors.Is(err, user.ErrorUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}
