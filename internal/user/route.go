package user

import "github.com/gin-gonic/gin"

// RegisterRoutes memasang seluruh endpoint user pada group yang
// diberikan. Dipasang di router pada group admin (AuthMiddleware +
// RequireRole("admin")); pendaftaran akun baru publik lewat
// /auth/register, bukan endpoint ini.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	users := rg.Group("/users")
	{
		users.GET("", h.FindAll)
		users.GET("/:id", h.FindByID)
		users.POST("", h.Create)
		users.PATCH("/:id/status", h.UpdateStatus)
		users.PATCH("/:id/role", h.UpdateRole)
		users.PATCH("/:id/profile", h.UpdateProfile)
	}
}
