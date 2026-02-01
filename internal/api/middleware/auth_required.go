package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	authservice "vngrocery/internal/service/auth"
)

const principalContextKey = "auth.principal"

type AuthRequired struct {
	verifier authservice.Verifier
}

func NewAuthRequired(verifier authservice.Verifier) *AuthRequired {
	return &AuthRequired{verifier: verifier}
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
