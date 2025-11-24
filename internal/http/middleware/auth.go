package middleware

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/api/idtoken"
)

// AuthMiddleware для Fiber: поддерживает JWT, Basic, статический Bearer и Google ID token.
func AuthMiddleware() fiber.Handler {
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	fmt.Println("JWT SECRET IN MIDDLEWARE:", jwtSecret)
	expected := strings.TrimSpace(os.Getenv("AUTH_TOKEN"))
	basicUser := strings.TrimSpace(os.Getenv("BASIC_USER"))
	basicPass := os.Getenv("BASIC_PASS")
	googleClientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))

	noAuthConfigured := jwtSecret == "" && expected == "" && basicUser == "" && googleClientID == ""

	return func(c *fiber.Ctx) error {
		if noAuthConfigured {
			return c.Next()
		}

		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Токен отсутствует"})
		}

		// JWT (если настроен)
		if strings.HasPrefix(auth, "Bearer ") && jwtSecret != "" {
			tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err == nil && token.Valid {
				identifier := "jwt_user"
				if v, ok := claims["email"].(string); ok && v != "" {
					identifier = v
				} else if sub, ok := claims["sub"].(string); ok && sub != "" {
					identifier = sub
				}
				c.Locals("user", "local:"+identifier)
				return c.Next()
			}
			// fallthrough к другим методам
		}

		// Basic Auth
		if strings.HasPrefix(auth, "Basic ") && basicUser != "" && basicPass != "" {
			enc := strings.TrimSpace(strings.TrimPrefix(auth, "Basic "))
			decoded, err := base64.StdEncoding.DecodeString(enc)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
			}
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) != 2 {
				return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
			}
			user := parts[0]
			pass := parts[1]
			if user != basicUser || pass != basicPass {
				return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
			}
			c.Locals("user", "basic:"+user)
			return c.Next()
		}

		// Bearer token (статический или Google ID token)
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))

			// Статический токен
			if expected != "" && token == expected {
				c.Locals("user", "token:static")
				return c.Next()
			}

			// Google ID token
			if googleClientID != "" {
				if _, err := idtoken.Validate(context.Background(), token, googleClientID); err == nil {
					// можно дополнительно извлечь email в handlers через idtoken.Validate при необходимости
					c.Locals("user", "google:token")
					return c.Next()
				}
			}
		}

		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}
}

// GetUser — получить пользовательскую локаль из Fiber контекста.
func GetUser(c *fiber.Ctx) string {
	if v := c.Locals("user"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
