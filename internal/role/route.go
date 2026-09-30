package role

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	roles := rg.Group("/roles")
	{
		roles.GET("", h.FindAll)
		roles.GET("/:id", h.FindByID)
		roles.PUT("/:id", h.Update)
		roles.DELETE("/:id", h.Delete)
	}
}
