package auth

import (
	"fmt"

	"ssot/gql/graphql/internal/auth/providers"
	"ssot/gql/graphql/internal/auth/tokens"
)

// ValidateToken validates a JWT token and returns the user
// It tries to validate as a local token first, then as a Cognito token
func ValidateToken(tokenString string) (*User, error) {
	// First try to validate as a local token
	localClaims, err := tokens.ValidateLocalToken(tokenString)
	if err == nil {
		// Create user from local claims
		user := &User{
			ID:       localClaims.UserID,
			Role:     localClaims.Role,
			Scope:    "ssot:gql:loancashflow:read", // Example scope for local tokens
			ClientID: "use-local-token",            // Local tokens do not have client_id
		}
		return user, nil
	}

	// If local token validation fails, try Cognito token validation
	cognitoUser, cognitoErr := providers.ValidateCognitoToken(tokenString)
	if cognitoErr == nil {
		// Convert providers.User to auth.User
		user := &User{
			ID:       cognitoUser.ID,
			Email:    cognitoUser.Email,
			Role:     cognitoUser.Role,
			Scope:    cognitoUser.Scope,
			ClientID: cognitoUser.ClientID,
		}
		return user, nil
	}

	// Both validations failed
	return nil, fmt.Errorf("token validation failed - local: %v, cognito: %v", err, cognitoErr)
}
