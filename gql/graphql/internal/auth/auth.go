package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey = contextKey("user")

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Env    string `json:"env"`
	jwt.RegisteredClaims
}

// GetJWTSecret returns the JWT secret from environment or a default value
func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// In production, you should always set a strong JWT_SECRET environment variable
		secret = "your-super-secret-jwt-key-change-this-in-production"
	}
	return []byte(secret)
}

// GetCurrentEnv returns the current environment from ENV variable
func GetCurrentEnv() string {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development" // default environment
	}
	return env
}

// GenerateToken generates a JWT token for a user
func GenerateToken(user *User) (string, error) {
	currentEnv := GetCurrentEnv()

	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Env:    currentEnv,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(GetJWTSecret())
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return GetJWTSecret(), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Check if token environment matches current environment
		currentEnv := GetCurrentEnv()
		if claims.Env != currentEnv {
			return nil, fmt.Errorf("token environment mismatch: token env '%s' does not match current env '%s'", claims.Env, currentEnv)
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// Middleware creates a middleware that validates JWT tokens
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow OPTIONS requests for CORS
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Allow introspection queries (for GraphQL playground)
		if r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		fmt.Println("x-amzn-oidc-data:", r.Header.Get("x-amzn-oidc-data"))
		fmt.Println("x-amzn-oidc-identity:", r.Header.Get("x-amzn-oidc-identity"))

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"message":"Authorization header required"}]}`))
			return
		}

		// Check if the header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"message":"Invalid authorization header format"}]}`))
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ValidateToken(tokenString)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"message":"Invalid token"}]}`))
			return
		}

		// Create user from claims
		user := &User{
			ID:    claims.UserID,
			Email: claims.Email,
			Role:  claims.Role,
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(ctx context.Context) (*User, error) {
	user, ok := ctx.Value(UserContextKey).(*User)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	return user, nil
}

// RequireRole checks if the user has the required role
func RequireRole(ctx context.Context, requiredRole string) error {
	user, err := GetUserFromContext(ctx)
	if err != nil {
		return err
	}

	if user.Role != requiredRole && user.Role != "admin" {
		return errors.New("insufficient permissions")
	}

	return nil
}
