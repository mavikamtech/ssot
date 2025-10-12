package acl

import (
	"context"
	"fmt"
	"strings"

	"ssot/gql/graphql/internal/auth/middleware"
)

// ACLMiddleware provides permission checking for GraphQL resolvers
type ACLMiddleware struct {
	service *ACLService
}

// NewACLMiddleware creates a new ACL middleware
func NewACLMiddleware(service *ACLService) *ACLMiddleware {
	return &ACLMiddleware{
		service: service,
	}
}

// CheckPermission validates if the current user has the required permission
func (m *ACLMiddleware) CheckPermission(ctx context.Context, table, column, action string) error {
	// Get user from context
	user, err := middleware.GetUserFromContext(ctx)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// Get merged ACL for user
	acl, err := m.service.GetMergedACL(ctx, user.Email)
	if err != nil {
		return fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Check if user has required permission
	if !acl.CanAccess(table, column, action) {
		return fmt.Errorf("access denied: user %s does not have %s permission for %s.%s",
			user.Email, action, table, column)
	}

	return nil
}

// CheckReadPermission is a convenience method for checking read access
func (m *ACLMiddleware) CheckReadPermission(ctx context.Context, table, column string) error {
	return m.CheckPermission(ctx, table, column, "read")
}

// CheckWritePermission is a convenience method for checking write access
func (m *ACLMiddleware) CheckWritePermission(ctx context.Context, table, column string) error {
	return m.CheckPermission(ctx, table, column, "write")
}

// RequirePermission is a decorator function that checks permissions before executing a resolver
func (m *ACLMiddleware) RequirePermission(table, column, action string) func(next func(ctx context.Context) (any, error)) func(ctx context.Context) (any, error) {
	return func(next func(ctx context.Context) (any, error)) func(ctx context.Context) (any, error) {
		return func(ctx context.Context) (any, error) {
			err := m.CheckPermission(ctx, table, column, action)
			if err != nil {
				return nil, err
			}
			return next(ctx)
		}
	}
}

// RequireRead is a convenience decorator for read permissions
func (m *ACLMiddleware) RequireRead(table, column string) func(next func(ctx context.Context) (any, error)) func(ctx context.Context) (any, error) {
	return m.RequirePermission(table, column, "read")
}

// RequireWrite is a convenience decorator for write permissions
func (m *ACLMiddleware) RequireWrite(table, column string) func(next func(ctx context.Context) (any, error)) func(ctx context.Context) (any, error) {
	return m.RequirePermission(table, column, "write")
}

// GetUserPermissions returns all permissions for the current user (for debugging/admin)
func (m *ACLMiddleware) GetUserPermissions(ctx context.Context) (*MergedACL, error) {
	user, err := middleware.GetUserFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}

	return m.service.GetMergedACL(ctx, user.Email)
}

// HasAnyPermission checks if user has any permission for a table
func (m *ACLMiddleware) HasAnyPermission(ctx context.Context, table string) (bool, error) {
	user, err := middleware.GetUserFromContext(ctx)
	if err != nil {
		return false, err
	}

	acl, err := m.service.GetMergedACL(ctx, user.Email)
	if err != nil {
		return false, err
	}

	// Check for any permission pattern matching the table
	for key := range acl.Permissions {
		if len(key) > len(table) && key[:len(table)] == table && key[len(table)] == '#' {
			return true, nil
		}
	}

	// Check for global permissions
	if _, exists := acl.Permissions["*#*"]; exists {
		return true, nil
	}

	return false, nil
}

// FilterColumns filters a list of columns based on user's read permissions
func (m *ACLMiddleware) FilterColumns(ctx context.Context, table string, columns []string) ([]string, error) {
	user, err := middleware.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	acl, err := m.service.GetMergedACL(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	// If user has wildcard or global access, return all columns
	if acl.CanAccess(table, "*", "read") {
		return columns, nil
	}

	// Filter columns based on specific permissions
	var allowedColumns []string
	for _, column := range columns {
		if acl.CanAccess(table, column, "read") {
			allowedColumns = append(allowedColumns, column)
		}
	}

	return allowedColumns, nil
}

// CheckPermissionFlexible validates permission using either ACL or scope check
// Returns true if EITHER the ACL check OR the scope check passes
func (m *ACLMiddleware) CheckPermissionFlexible(ctx context.Context, table, column, action, requiredScope string) error {
	// Get user from context
	user, err := middleware.GetUserFromContext(ctx)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// First try ACL check
	acl, err := m.service.GetMergedACL(ctx, user.Email)
	if err == nil && acl.CanAccess(table, column, action) {
		return nil // ACL check passed
	}

	// If ACL check failed or errored, try scope check
	if requiredScope != "" && strings.Contains(user.Scope, requiredScope) {
		return nil // Scope check passed
	}

	// Both checks failed
	return fmt.Errorf("access denied: user %s does not have %s permission for %s.%s (checked both ACL and scope %s)",
		user.Email, action, table, column, requiredScope)
}

// CheckReadPermissionFlexible checks read permission with fallback to scope
func (m *ACLMiddleware) CheckReadPermissionFlexible(ctx context.Context, table, column, requiredScope string) error {
	return m.CheckPermissionFlexible(ctx, table, column, "read", requiredScope)
}

// GetColumnPermissionsFlexible returns column-level permissions for a table with scope fallback
func (m *ACLMiddleware) GetColumnPermissionsFlexible(ctx context.Context, table, requiredScope string, allColumns []string) (*ColumnPermissions, error) {
	// Get user from context
	user, err := middleware.GetUserFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}

	// Try ACL first
	acl, err := m.service.GetMergedACL(ctx, user.Email)
	if err == nil {
		// ACL is available, use it to determine column permissions
		columnAccess := make(map[string]string)

		for _, column := range allColumns {
			if acl.CanAccess(table, column, "read") {
				columnAccess[column] = "allowed"
			} else {
				columnAccess[column] = "blocked"
			}
		}

		return NewColumnPermissions(table, columnAccess, false), nil // ACL used, not scope fallback
	}

	// ACL not available, check scope fallback
	if requiredScope != "" && strings.Contains(user.Scope, requiredScope) {
		// Scope check passed, allow all columns
		columnAccess := make(map[string]string)
		for _, column := range allColumns {
			columnAccess[column] = "allowed"
		}
		return NewColumnPermissions(table, columnAccess, true), nil // Scope fallback used
	}

	// Both ACL and scope failed, block all columns
	columnAccess := make(map[string]string)
	for _, column := range allColumns {
		columnAccess[column] = "blocked"
	}
	return NewColumnPermissions(table, columnAccess, false), fmt.Errorf("access denied: user %s does not have read permission for %s (checked both ACL and scope %s)",
		user.Email, table, requiredScope)
}

// CheckWritePermissionFlexible checks write permission with fallback to scope
func (m *ACLMiddleware) CheckWritePermissionFlexible(ctx context.Context, table, column, requiredScope string) error {
	return m.CheckPermissionFlexible(ctx, table, column, "write", requiredScope)
}

// TryACLFirst attempts ACL check first, returns result and whether ACL was available
func (m *ACLMiddleware) TryACLFirst(ctx context.Context, table, column, action string) (bool, bool, error) {
	// Get user from context
	user, err := middleware.GetUserFromContext(ctx)
	if err != nil {
		return false, false, fmt.Errorf("authentication required: %w", err)
	}

	// Try ACL check
	acl, err := m.service.GetMergedACL(ctx, user.Email)
	if err != nil {
		// ACL not available, caller should use fallback method
		return false, false, nil
	}

	// ACL is available, return the result
	hasPermission := acl.CanAccess(table, column, action)
	return hasPermission, true, nil
}

// GetUserEmail extracts email from the current user context
func GetUserEmail(ctx context.Context) (string, error) {
	user, err := middleware.GetUserFromContext(ctx)
	if err != nil {
		return "", err
	}
	return user.Email, nil
}

// GetFieldFilters retrieves the merged field filters for the current user
func (m *ACLMiddleware) GetFieldFilters(ctx context.Context) (map[string]FieldFilter, error) {
	// Get user from context
	user, err := middleware.GetUserFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}

	// Get merged ACL for user
	acl, err := m.service.GetMergedACL(ctx, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	return acl.FieldFilters, nil
}

// FilterDataByField filters an array of data based on field filter rules
func (m *ACLMiddleware) FilterDataByField(ctx context.Context, data []any, fieldName string, getFieldValue func(any) string) ([]any, error) {
	fieldFilters, err := m.GetFieldFilters(ctx)
	if err != nil {
		return nil, err
	}

	return FilterArrayByField(data, fieldName, fieldFilters, getFieldValue), nil
}

// CheckFieldValueAccess checks if a specific field value is allowed for the current user
func (m *ACLMiddleware) CheckFieldValueAccess(ctx context.Context, fieldName, fieldValue string) (bool, error) {
	fieldFilters, err := m.GetFieldFilters(ctx)
	if err != nil {
		return false, err
	}

	filter, exists := fieldFilters[fieldName]
	if !exists {
		// No filter means all values are allowed
		return true, nil
	}

	return filter.IsValueAllowed(fieldValue), nil
}
