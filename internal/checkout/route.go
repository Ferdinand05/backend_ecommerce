package checkout

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	rg.POST("/preview", handler.Preview)
}
