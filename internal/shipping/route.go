package shipping

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	rg.GET("/areas", handler.SearchAreas)
	rg.POST("/rates", handler.CalculateRates)
}
