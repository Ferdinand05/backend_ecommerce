package role

import "github.com/gin-gonic/gin"

// RegisterRoutes memasang endpoint role pada group yang diberikan.
// Dipasang di router pada group admin (AuthMiddleware +
// RequireRole("admin")). Role biasanya di-seed, bukan di-CRUD via API;
// endpoint ini opsional dan dapat dihapus bila tidak diperlukan.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	roles := rg.Group("/roles")
	{
		roles.GET("", h.FindAll)
		roles.GET("/:id", h.FindByID)
		roles.PUT("/:id", h.Update)
		roles.DELETE("/:id", h.Delete)
	}
}
