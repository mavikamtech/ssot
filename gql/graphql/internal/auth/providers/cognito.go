package providers

import (
	"errors"
	"fmt"
	"os"

	"ssot/gql/graphql/internal/constants"

	"github.com/MicahParks/keyfunc/v2"
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

// GetCurrentEnv returns the current environment from ENV variable
func GetCurrentEnv() string {
	env := os.Getenv("ENV")
	if env == "" {
		env = "staging" // default environment
	}
	return env
}

// AWS Cognito configuration
var (
	Region     = "us-east-1"
	UserPoolID = constants.GetUserPoolID("ssot-gql-" + GetCurrentEnv())
)

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
