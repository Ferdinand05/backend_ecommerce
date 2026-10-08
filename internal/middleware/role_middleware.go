package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRole menolak request yang role-nya tidak sama dengan parameter.
// Wajib dipasang SETELAH AuthMiddleware karena membaca c.Get("role").
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.GetString("role") != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "forbidden",
			})
			return
		}

		c.Next()
	}
}
