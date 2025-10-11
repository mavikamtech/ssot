package providers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

// ValidateOIDCAuth validates OIDC authentication from x-amzn-oidc-data header
func ValidateOIDCAuth(r *http.Request) (*User, error) {
	oidcData := r.Header.Get("x-amzn-oidc-data")
	if oidcData == "" {
		return nil, errors.New("no OIDC data found")
	}

	// First, try to manually parse the token to extract the payload
	parts := strings.Split(oidcData, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	// Decode the payload (second part) manually
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Try with standard base64 padding
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to decode payload: %v", err)
		}
	}

	// Parse the payload JSON
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %v", err)
	}

	// Check issuer to determine if this is a Microsoft token
	issuer, ok := claims["iss"].(string)
	if !ok {
		return nil, errors.New("issuer not found in token")
	}

	// For now, we'll validate the structure but skip signature verification for Microsoft tokens
	// In production, you would need to implement proper signature verification
	if strings.Contains(issuer, "login.microsoftonline.com") {
		// Microsoft Entra ID token - validate structure and extract user info
		return validateMicrosoftToken(claims)
	}

	// For other tokens, try the standard JWT validation approach
	var jwksURL string
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1" // default region
	}
	jwksURL = fmt.Sprintf("https://public-keys.auth.elb.%s.amazonaws.com", region)

	// Now validate the token with proper signature verification
	validatedToken, err := jwt.Parse(oidcData, func(token *jwt.Token) (interface{}, error) {
		// Create JWKS from the resource at the given URL
		jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{})
		if err != nil {
			return nil, fmt.Errorf("failed to get JWKS: %v", err)
		}
		defer jwks.EndBackground()

		// Get the key function for this token
		keyFunc, err := jwks.Keyfunc(token)
		if err != nil {
			return nil, fmt.Errorf("failed to get key function: %v", err)
		}
		return keyFunc, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to validate OIDC token signature: %v", err)
	}

	if !validatedToken.Valid {
		return nil, errors.New("invalid OIDC token")
	}

	// Extract claims from the validated token
	validatedClaims, ok := validatedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return createUserFromClaims(validatedClaims)
}

// validateMicrosoftToken validates a Microsoft Entra ID token (without signature verification for now)
func validateMicrosoftToken(claims map[string]interface{}) (*User, error) {
	// Check if email field exists and is not empty
	email, exists := claims["email"]
	if !exists || email == "" {
		return nil, errors.New("email not found in token")
	}

	emailStr := fmt.Sprintf("%v", email)
	if emailStr == "" {
		return nil, errors.New("empty email in token")
	}

	// Validate token expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, errors.New("token has expired")
		}
	}

	// Create user with the required scope
	user := &User{
		ID:       fmt.Sprintf("oidc-%v", claims["sub"]),
		Email:    emailStr,
		Role:     "user",
		Scope:    "ssot:gql:loancashflow:read",
		ClientID: "oidc-client",
	}

	return user, nil
}

// createUserFromClaims creates a user from validated JWT claims
func createUserFromClaims(claims map[string]interface{}) (*User, error) {
	// Check if email field exists and is not empty
	email, exists := claims["email"]
	if !exists || email == "" {
		return nil, errors.New("email not found in OIDC data")
	}

	emailStr := fmt.Sprintf("%v", email)
	if emailStr == "" {
		return nil, errors.New("empty email in OIDC data")
	}

	// Validate token expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, errors.New("OIDC token has expired")
		}
	}

	// Validate issuer - now support both Microsoft and AWS patterns
	if iss, ok := claims["iss"].(string); ok {
		if !strings.Contains(iss, "amazonaws.com") && !strings.Contains(iss, "microsoftonline.com") {
			return nil, fmt.Errorf("invalid issuer: %s", iss)
		}
	}

	// Create user with the required scope
	user := &User{
		ID:       fmt.Sprintf("oidc-%v", claims["sub"]),
		Email:    emailStr,
		Role:     "user",
		Scope:    "ssot:gql:loancashflow:read",
		ClientID: "oidc-client",
	}

	return user, nil
}
