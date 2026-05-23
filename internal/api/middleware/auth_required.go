package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	authservice "vngrocery/internal/service/auth"
)

const principalContextKey = "auth.principal"
const userContextKey = "auth.user"

type AuthRequired struct {
	verifier authservice.Verifier
	users    repository.UserRepository
}

func NewAuthRequired(verifier authservice.Verifier, users ...repository.UserRepository) *AuthRequired {
	var userRepo repository.UserRepository
	if len(users) > 0 {
		userRepo = users[0]
	}
	return &AuthRequired{verifier: verifier, users: userRepo}
}

func (m *AuthRequired) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := extractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		principal, err := m.verifier.Verify(c.Request.Context(), token)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, authservice.ErrUnauthorized) {
				status = http.StatusUnauthorized
			}

			c.JSON(status, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		if m.users != nil {
			user, err := m.users.GetByID(c.Request.Context(), strings.TrimSpace(principal.UserID))
			if err != nil || user.UserID == "" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": authservice.ErrUnauthorized.Error(),
				})
				c.Abort()
				return
			}
			if !domain.IsActiveUser(user) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "account is not active",
				})
				c.Abort()
				return
			}
			principal.Email = user.Email
			principal.Role = user.Role
			c.Set(userContextKey, user)
		}

		c.Set(principalContextKey, principal)
		c.Next()
	}
}

func GetPrincipal(c *gin.Context) (authservice.Principal, bool) {
	value, ok := c.Get(principalContextKey)
	if !ok {
		return authservice.Principal{}, false
	}

	principal, ok := value.(authservice.Principal)
	return principal, ok
}

func GetUser(c *gin.Context) (domain.User, bool) {
	value, ok := c.Get(userContextKey)
	if !ok {
		return domain.User{}, false
	}

	user, ok := value.(domain.User)
	return user, ok
}

func extractBearerToken(header string) (string, error) {
	if strings.TrimSpace(header) == "" {
		return "", authservice.ErrUnauthorized
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", authservice.ErrUnauthorized
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", authservice.ErrUnauthorized
	}

	return token, nil
}
