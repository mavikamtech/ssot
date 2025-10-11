package handlers

import (
	"encoding/json"
	"net/http"

	"ssot/gql/graphql/internal/auth"
	"ssot/gql/graphql/internal/auth/tokens"

	"golang.org/x/crypto/bcrypt"
)

// Demo handlers - these are for testing purposes only and should not be used in production

// Simple in-memory user store for demo purposes
// In production, you should use a proper database
var users = map[string]string{
	"admin@example.com": "***", // password: "admin123"
	"user@example.com":  "***", // password: "user123"
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash checks if the provided password matches the hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// LoginHandler handles user login - FOR DEMO PURPOSES ONLY
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginReq auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(auth.ErrorResponse{Error: "Invalid JSON"})
		return
	}

	// Check if user exists and password is correct
	hashedPassword, exists := users[loginReq.Email]
	if !exists || !CheckPasswordHash(loginReq.Password, hashedPassword) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(auth.ErrorResponse{Error: "Invalid credentials"})
		return
	}

	// Create user object
	user := &auth.User{
		ID:    "user-" + loginReq.Email, // Simple ID generation
		Email: loginReq.Email,
		Role:  "user", // Default role, you can implement role management
	}

	// Set admin role for admin user
	if loginReq.Email == "admin@example.com" {
		user.Role = "admin"
	}

	// Generate JWT token - convert to tokens.User type
	tokenUser := &tokens.User{
		ID:       user.ID,
		Email:    user.Email,
		Role:     user.Role,
		Scope:    user.Scope,
		ClientID: user.ClientID,
	}
	token, err := tokens.GenerateToken(tokenUser)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(auth.ErrorResponse{Error: "Failed to generate token"})
		return
	}

	// Return token and user info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auth.LoginResponse{
		Token: token,
		User:  user,
	})
}

// CreateUserHandler creates a new user - FOR DEMO PURPOSES ONLY
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type CreateUserRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	var createUserReq CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&createUserReq); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(auth.ErrorResponse{Error: "Invalid JSON"})
		return
	}

	// Check if user already exists
	if _, exists := users[createUserReq.Email]; exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(auth.ErrorResponse{Error: "User already exists"})
		return
	}

	// Hash password
	hashedPassword, err := HashPassword(createUserReq.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(auth.ErrorResponse{Error: "Failed to hash password"})
		return
	}

	// Store user
	users[createUserReq.Email] = hashedPassword

	// Create user object
	user := &auth.User{
		ID:    "user-" + createUserReq.Email,
		Email: createUserReq.Email,
		Role:  createUserReq.Role,
	}

	// Generate JWT token - convert to tokens.User type
	tokenUser := &tokens.User{
		ID:       user.ID,
		Email:    user.Email,
		Role:     user.Role,
		Scope:    user.Scope,
		ClientID: user.ClientID,
	}
	token, err := tokens.GenerateToken(tokenUser)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(auth.ErrorResponse{Error: "Failed to generate token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(auth.LoginResponse{
		Token: token,
		User:  user,
	})
}
