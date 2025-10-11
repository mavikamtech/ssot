package tokens

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents an authenticated user (duplicated to avoid import cycle)
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Scope    string `json:"scope"`
	ClientID string `json:"client_id"`
}

// Claims represents JWT claims for local tokens (duplicated to avoid import cycle)
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Env    string `json:"env"`
	jwt.RegisteredClaims
}

// GetCurrentEnv returns the current environment from ENV variable
func GetCurrentEnv() string {
	env := os.Getenv("ENV")
	if env == "" {
		env = "staging" // default environment
	}
	return env
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

// ValidateLocalToken validates a local JWT token and returns the claims
func ValidateLocalToken(tokenString string) (*Claims, error) {
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
