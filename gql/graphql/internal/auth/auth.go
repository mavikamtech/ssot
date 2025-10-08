package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey = contextKey("user")

// JWK represents a JSON Web Key
type JWK struct {
	Kty string   `json:"kty"`
	Use string   `json:"use"`
	Kid string   `json:"kid"`
	X5t string   `json:"x5t"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

// JWKSet represents a set of JSON Web Keys
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

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

		oidcClaims, err := parseAlbOIDCHeader(r.Header.Get("x-amzn-oidc-data"))
		if err != nil {
			fmt.Println("parse error:", err)
			return
		}
		fmt.Printf("Authenticated user: %v\n", oidcClaims["email"])
		fmt.Println("claims:", oidcClaims)

		// ctx := context.WithValue(r.Context(), "user", oidcClaims)
		// next.ServeHTTP(w, r.WithContext(ctx))

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
		jwtClaims, err := ValidateToken(tokenString)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"message":"Invalid token"}]}`))
			return
		}

		// Create user from claims
		user := &User{
			ID:    jwtClaims.UserID,
			Email: jwtClaims.Email,
			Role:  jwtClaims.Role,
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

// fetchEntraIDPublicKeys fetches the public keys from Entra ID
func fetchEntraIDPublicKeys() (*JWKSet, error) {
	const jwksURL = "https://login.microsoftonline.com/0beed8a0-2f9c-4bff-a92a-1e445f1c15bd/discovery/v2.0/keys"

	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks JWKSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	return &jwks, nil
}

// jwkToRSAPublicKey converts a JWK to RSA public key
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode the modulus (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode the exponent (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert bytes to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

func parseAlbOIDCHeader(h string) (jwt.MapClaims, error) {
	decoded, err := base64.StdEncoding.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}

	idToken, _ := payload["id_token"].(string)
	if idToken == "" {
		return nil, fmt.Errorf("no id_token in header")
	}

	jwks, err := fetchEntraIDPublicKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Entra ID public keys: %w", err)
	}

	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT header: %w", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("failed to parse JWT header: %w", err)
	}

	kid, ok := header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("no kid in JWT header")
	}

	var jwk *JWK
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			jwk = &key
			break
		}
	}

	if jwk == nil {
		return nil, fmt.Errorf("no matching key found for kid: %s", kid)
	}

	publicKey, err := jwkToRSAPublicKey(*jwk)
	if err != nil {
		return nil, fmt.Errorf("failed to convert JWK to RSA public key: %w", err)
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(idToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse and verify JWT: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	return claims, nil
}
