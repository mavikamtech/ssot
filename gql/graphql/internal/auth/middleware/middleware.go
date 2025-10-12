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

		// Check if the header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"message":"Invalid authorization header format"}]}`))
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		jwtClaims, err := auth.ValidateToken(tokenString)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"message":"Invalid token"}]}`))
			return
		}

		// Add user to context
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
