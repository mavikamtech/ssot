package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"ssot/gql/graphql/internal/auth"
	"ssot/gql/graphql/internal/auth/providers"
)

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

		// Log all headers for debugging
		log.Println("Request headers:")
		for key, values := range r.Header {
			for _, value := range values {
				log.Printf("  %s: %s", key, value)
			}
		}

		// Check for x-amzn-oidc-data header first
		if user, err := providers.ValidateOIDCAuth(r); err == nil {
			// Convert providers.User to auth.User
			authUser := &auth.User{
				ID:       user.ID,
				Email:    user.Email,
				Role:     user.Role,
				Scope:    user.Scope,
				ClientID: user.ClientID,
			}
			// OIDC authentication successful, add user to context and continue
			ctx := context.WithValue(r.Context(), auth.UserContextKey, authUser)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		} else {
			log.Printf("OIDC validation error: %v\n", err)
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"message":"Authorization header required"}]}`))
			return
		}

		var tokenString string

		// Check if the header starts with "Bearer " and extract token
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// If no Bearer prefix, use the raw value as token
			tokenString = authHeader
		}

		// Try to validate as JWT token first
		jwtClaims, err := auth.ValidateToken(tokenString)
		if err != nil {
			// If JWT validation fails and no Bearer prefix, try as OIDC data
			if !strings.HasPrefix(authHeader, "Bearer ") {
				// Set the authorization header value as x-amzn-oidc-data header
				r.Header.Set("x-amzn-oidc-data", authHeader)
				// Try OIDC validation again
				if user, oidcErr := providers.ValidateOIDCAuth(r); oidcErr == nil {
					// Convert providers.User to auth.User
					authUser := &auth.User{
						ID:       user.ID,
						Email:    user.Email,
						Role:     user.Role,
						Scope:    user.Scope,
						ClientID: user.ClientID,
					}
					// OIDC authentication successful, add user to context and continue
					ctx := context.WithValue(r.Context(), auth.UserContextKey, authUser)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Both JWT and OIDC validation failed
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"message":"Invalid token"}]}`))
			return
		}

		// JWT validation successful, add user to context
		ctx := context.WithValue(r.Context(), auth.UserContextKey, jwtClaims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(ctx context.Context) (*auth.User, error) {
	user, ok := ctx.Value(auth.UserContextKey).(*auth.User)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	return user, nil
}
