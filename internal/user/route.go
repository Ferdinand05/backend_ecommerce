package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	users := rg.Group("/users")
	{
		users.GET("", h.FindAll)
		users.GET("/:id", h.FindByID)
		users.POST("", h.Create)
	}
}
