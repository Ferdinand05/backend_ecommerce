package metrics

import "github.com/gin-gonic/gin"

// RegisterRoutes memasang endpoint ringkasan metrik dashboard pada group admin.
func RegisterRoutes(admin *gin.RouterGroup, h *Handler) {
	admin.GET("/metrics/summary", h.Summary)
}
