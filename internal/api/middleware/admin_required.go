package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type AdminRequired struct {
	users repository.UserRepository
}

func NewAdminRequired(users repository.UserRepository) *AdminRequired {
	return &AdminRequired{users: users}
}

func (m *AdminRequired) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := GetPrincipal(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "authenticated principal not found",
			})
			c.Abort()
			return
		}

		if m.users == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "user repository not configured",
			})
			c.Abort()
			return
		}

		user, ok := GetUser(c)
		if !ok {
			var err error
			user, err = m.users.GetByID(c.Request.Context(), strings.TrimSpace(principal.UserID))
			if err != nil || user.UserID == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "access denied",
				})
				c.Abort()
				return
			}
		}

		if !strings.EqualFold(strings.TrimSpace(user.Role), domain.RoleAdmin) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "admin access required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
