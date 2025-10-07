package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey = contextKey("user")

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Scope    string `json:"scope"`
	ClientID string `json:"client_id"`
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Env    string `json:"env"`
	jwt.RegisteredClaims
}

// AWS Cognito configuration
var (
	Region        = "us-east-1"
	UserPoolID    = "ssot-gql-" + GetCurrentEnv()
	RequiredScope = "default-m2m-resource-server-vuiu3j/read"
)

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

// ValidateCognitoToken validates an AWS Cognito JWT token
func ValidateCognitoToken(tokenString string) (*User, error) {
	// Check if Cognito configuration is available
	if UserPoolID == "" {
		return nil, errors.New("cognito configuration not available")
	}

	// Override region from environment if available
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = Region
	}

	// Create JWKS URL
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, UserPoolID)

	// Create JWKS from the resource at the given URL.
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %v", err)
	}
	defer jwks.EndBackground()

	// Parse and validate token
	token, err := jwt.Parse(tokenString, jwks.Keyfunc)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	// Validate issuer
	issuer := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", region, UserPoolID)
	if claims["iss"] != issuer {
		return nil, errors.New("invalid issuer")
	}

	// Check scope
	if scopes, ok := claims["scope"].(string); ok {
		if !strings.Contains(scopes, RequiredScope) {
			return nil, fmt.Errorf("missing required scope %s", RequiredScope)
		}
	} else {
		return nil, errors.New("no scopes in token")
	}

	// Create user from Cognito claims
	user := &User{
		ID:       fmt.Sprintf("cognito-%v", claims["sub"]),
		Email:    fmt.Sprintf("%v", claims["email"]),
		Role:     "user", // Default role for Cognito users
		Scope:    fmt.Sprintf("%v", claims["scope"]),
		ClientID: fmt.Sprintf("%v", claims["client_id"]),
	}

	return user, nil
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

// ValidateToken validates a JWT token and returns the user
// It tries to validate as a local token first, then as a Cognito token
func ValidateToken(tokenString string) (*User, error) {
	// First try to validate as a local token
	localClaims, err := validateLocalToken(tokenString)
	if err == nil {
		// Create user from local claims
		user := &User{
			ID:       localClaims.UserID,
			Email:    localClaims.Email,
			Role:     localClaims.Role,
			Scope:    "ssot:gql:loancashflow:read", // Example scope for local tokens
			ClientID: "use-local-token",            // Local tokens do not have client_id
		}
		return user, nil
	}

	// If local token validation fails, try Cognito token validation
	cognitoUser, cognitoErr := ValidateCognitoToken(tokenString)
	if cognitoErr == nil {
		return cognitoUser, nil
	}

	// Both validations failed
	return nil, fmt.Errorf("token validation failed - local: %v, cognito: %v", err, cognitoErr)
}

// validateLocalToken validates a local JWT token and returns the claims
func validateLocalToken(tokenString string) (*Claims, error) {
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
		user, err := ValidateToken(tokenString)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"message":"Invalid token"}]}`))
			return
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
