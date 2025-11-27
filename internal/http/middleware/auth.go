package middleware

import (
	"context"
	"encoding/base64"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/api/idtoken"
)

type ctxKey string

const ctxUserKey ctxKey = "user"

// AuthMiddleware — middleware для net/http.
// Ожидает заголовок Authorization: Bearer <token>.
// Токен сравнивается с AUTH_TOKEN из окружения.
// Если AUTH_TOKEN пустой — авторизация отключена и запросы пропускаются.
func AuthMiddleware(jwtSecret string) fiber.Handler {
	expected := strings.TrimSpace(os.Getenv("AUTH_TOKEN"))
	basicUser := strings.TrimSpace(os.Getenv("BASIC_USER"))
	basicPass := os.Getenv("BASIC_PASS")
	googleClientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))

	// jwtSecret is passed as argument, no need to read from env

	noAuthConfigured := expected == "" && basicUser == "" && googleClientID == "" && jwtSecret == ""

	return func(c *fiber.Ctx) error {
		// Если токен не настроен — пропустить (dev mode)
		if noAuthConfigured {
			return c.Next()
		}

		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		// JWT (если настроен)
		if strings.HasPrefix(auth, "Bearer ") && jwtSecret != "" {
			tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			if err == nil && token != nil && token.Valid {
				identifier := "jwt_user"
				if v, ok := claims["email"].(string); ok && v != "" {
					identifier = v
				}

				// Try to get user_id from sub
				if sub, ok := claims["sub"]; ok {
					var uid uint
					switch v := sub.(type) {
					case float64:
						uid = uint(v)
					case int64:
						uid = uint(v)
					case int:
						uid = uint(v)
					}
					if uid > 0 {
						c.Locals("user_id", uid)
					}
				}

				if role, ok := claims["role"].(string); ok {
					c.Locals("user_role", role)
				}

				if name, ok := claims["name"].(string); ok {
					c.Locals("user_name", name)
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

// GetUserID — получить ID пользователя из Fiber контекста (если есть).
func GetUserID(c *fiber.Ctx) uint {
	if v := c.Locals("user_id"); v != nil {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// GetUserRole — получить роль пользователя из Fiber контекста.
func GetUserRole(c *fiber.Ctx) string {
	if v := c.Locals("user_role"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetUserName — получить имя пользователя из Fiber контекста.
func GetUserName(c *fiber.Ctx) string {
	if v := c.Locals("user_name"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
