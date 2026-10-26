package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST(
			"/email-verification/verify",
			h.VerifyEmail,
		)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/logout", h.Logout)
		auth.POST(
			"/email-verification/resend",
			h.ResendVerification,
		)
		auth.POST("/password/forgot", h.ForgotPassword)
		auth.POST("/password/reset", h.ResetPassword)
	}
}

// RegisterProfileRoutes memasang endpoint profil pada group yang
// diberikan. Group harus sudah berawalan /auth dan dipanggil dengan
// middleware autentikasi.
func RegisterProfileRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("", h.GetMe)
	rg.PUT("", h.UpdateMe)
}
