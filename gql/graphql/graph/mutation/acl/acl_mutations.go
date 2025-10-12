package acl

import (
	"context"
	"fmt"
	"ssot/gql/graphql/graph/model"
	"ssot/gql/graphql/internal/acl"
	"ssot/gql/graphql/internal/services"
	"strings"
)

// ACLMutationResolver handles ACL-related mutations
type ACLMutationResolver struct {
	ServiceManager *services.ServiceManager
}

// NewACLMutationResolver creates a new ACL mutation resolver
func NewACLMutationResolver(serviceManager *services.ServiceManager) *ACLMutationResolver {
	return &ACLMutationResolver{
		ServiceManager: serviceManager,
	}
}

// AddUserACL creates a new user ACL record
func (r *ACLMutationResolver) AddUserACL(ctx context.Context, input model.AddUserACLInput) (*model.ACLMutationResult, error) {
	// Check admin access
	if err := r.ServiceManager.ACLMiddleware.RequireAdminAccess(ctx); err != nil {
		return &model.ACLMutationResult{
			Success: false,
			Message: fmt.Sprintf("Access denied: %v", err),
		}, nil
	}

	// Prevent adding admin group - only admin users can modify ACL, but cannot assign admin role
	for _, group := range input.Groups {
		if group == "admin" {
			return &model.ACLMutationResult{
				Success: false,
				Message: "Cannot assign admin group through ACL configuration",
			}, nil
		}
	}

	// Convert permissions from GraphQL to ACL format
	permissions := convertPermissionsToACL(input.Permissions)
	fieldFilters := convertFieldFiltersToACL(input.FieldFilters)

	// Create the user ACL
	err := r.ServiceManager.ACLService.CreateUserWithFieldFilters(
		ctx, input.Email, input.Groups, permissions, fieldFilters)
	if err != nil {
		return &model.ACLMutationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create user ACL: %v", err),
		}, nil
	}

	// Fetch the created record for response
	record, err := r.ServiceManager.ACLService.GetMergedACL(ctx, input.Email)
	if err != nil {
		// Return success but without the record
		return &model.ACLMutationResult{
			Success: true,
			Message: "User ACL created successfully",
		}, nil
	}

	return &model.ACLMutationResult{
		Success: true,
		Message: "User ACL created successfully",
		Record:  convertACLToGraphQL(record, input.Email),
	}, nil
}

// UpdateUserACL updates an existing user ACL record
func (r *ACLMutationResolver) UpdateUserACL(ctx context.Context, input model.UpdateUserACLInput) (*model.ACLMutationResult, error) {
	// Check admin access
	if err := r.ServiceManager.ACLMiddleware.RequireAdminAccess(ctx); err != nil {
		return &model.ACLMutationResult{
			Success: false,
			Message: fmt.Sprintf("Access denied: %v", err),
		}, nil
	}

	// Prevent adding admin group
	if len(input.Groups) > 0 {
		for _, group := range input.Groups {
			if group == "admin" {
				return &model.ACLMutationResult{
					Success: false,
					Message: "Cannot assign admin group through ACL configuration",
				}, nil
			}
		}
	}

	// Update groups if provided
	if len(input.Groups) > 0 {
		err := r.ServiceManager.ACLService.UpdateUserGroups(ctx, input.Email, input.Groups)
		if err != nil {
			return &model.ACLMutationResult{
				Success: false,
				Message: fmt.Sprintf("Failed to update user groups: %v", err),
			}, nil
		}
	}

	// Update permissions if provided
	if len(input.Permissions) > 0 {
		permissions := convertPermissionsToACL(input.Permissions)
		err := r.ServiceManager.ACLService.UpdateUserPermissions(ctx, input.Email, permissions)
		if err != nil {
			return &model.ACLMutationResult{
				Success: false,
				Message: fmt.Sprintf("Failed to update user permissions: %v", err),
			}, nil
		}
	}

	// Fetch the updated record for response
	record, err := r.ServiceManager.ACLService.GetMergedACL(ctx, input.Email)
	if err != nil {
		// Return success but without the record
		return &model.ACLMutationResult{
			Success: true,
			Message: "User ACL updated successfully",
		}, nil
	}

	return &model.ACLMutationResult{
		Success: true,
		Message: "User ACL updated successfully",
		Record:  convertACLToGraphQL(record, input.Email),
	}, nil
}

// AddGroupACL creates a new group ACL record
func (r *ACLMutationResolver) AddGroupACL(ctx context.Context, input model.AddGroupACLInput) (*model.ACLMutationResult, error) {
	// Check admin access
	if err := r.ServiceManager.ACLMiddleware.RequireAdminAccess(ctx); err != nil {
		return &model.ACLMutationResult{
			Success: false,
			Message: fmt.Sprintf("Access denied: %v", err),
		}, nil
	}

	// Prevent creating admin group
	if input.GroupName == "admin" {
		return &model.ACLMutationResult{
			Success: false,
			Message: "Cannot modify admin group through ACL configuration",
		}, nil
	}

	// Convert permissions from GraphQL to ACL format
	permissions := convertPermissionsToACL(input.Permissions)
	fieldFilters := convertFieldFiltersToACL(input.FieldFilters)

	// Create the group ACL
	err := r.ServiceManager.ACLService.CreateGroupWithFieldFilters(
		ctx, input.GroupName, permissions, fieldFilters)
	if err != nil {
		return &model.ACLMutationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create group ACL: %v", err),
		}, nil
	}

	return &model.ACLMutationResult{
		Success: true,
		Message: fmt.Sprintf("Group ACL '%s' created successfully", input.GroupName),
	}, nil
}

// UpdateGroupACL updates an existing group ACL record
func (r *ACLMutationResolver) UpdateGroupACL(ctx context.Context, input model.UpdateGroupACLInput) (*model.ACLMutationResult, error) {
	// Check admin access
	if err := r.ServiceManager.ACLMiddleware.RequireAdminAccess(ctx); err != nil {
		return &model.ACLMutationResult{
			Success: false,
			Message: fmt.Sprintf("Access denied: %v", err),
		}, nil
	}

	// Prevent modifying admin group
	if input.GroupName == "admin" {
		return &model.ACLMutationResult{
			Success: false,
			Message: "Cannot modify admin group through ACL configuration",
		}, nil
	}

	// Update permissions if provided
	if len(input.Permissions) > 0 {
		permissions := convertPermissionsToACL(input.Permissions)
		err := r.ServiceManager.ACLService.UpdateGroupPermissions(ctx, input.GroupName, permissions)
		if err != nil {
			return &model.ACLMutationResult{
				Success: false,
				Message: fmt.Sprintf("Failed to update group permissions: %v", err),
			}, nil
		}
	}

	return &model.ACLMutationResult{
		Success: true,
		Message: fmt.Sprintf("Group ACL '%s' updated successfully", input.GroupName),
	}, nil
}

// DeleteUserACL deletes a user ACL record
func (r *ACLMutationResolver) DeleteUserACL(ctx context.Context, email string) (*model.ACLMutationResult, error) {
	// Check admin access
	if err := r.ServiceManager.ACLMiddleware.RequireAdminAccess(ctx); err != nil {
		return &model.ACLMutationResult{
			Success: false,
			Message: fmt.Sprintf("Access denied: %v", err),
		}, nil
	}

	// Delete the user ACL
	err := r.ServiceManager.ACLService.DeleteUser(ctx, email)
	if err != nil {
		return &model.ACLMutationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to delete user ACL: %v", err),
		}, nil
	}

	return &model.ACLMutationResult{
		Success: true,
		Message: fmt.Sprintf("User ACL for '%s' deleted successfully", email),
	}, nil
}

// DeleteGroupACL deletes a group ACL record
func (r *ACLMutationResolver) DeleteGroupACL(ctx context.Context, groupName string) (*model.ACLMutationResult, error) {
	// Check admin access
	if err := r.ServiceManager.ACLMiddleware.RequireAdminAccess(ctx); err != nil {
		return &model.ACLMutationResult{
			Success: false,
			Message: fmt.Sprintf("Access denied: %v", err),
		}, nil
	}

	// Prevent deleting admin group
	if groupName == "admin" {
		return &model.ACLMutationResult{
			Success: false,
			Message: "Cannot delete admin group through ACL configuration",
		}, nil
	}

	// Delete the group ACL
	err := r.ServiceManager.ACLService.DeleteGroup(ctx, groupName)
	if err != nil {
		return &model.ACLMutationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to delete group ACL: %v", err),
		}, nil
	}

	return &model.ACLMutationResult{
		Success: true,
		Message: fmt.Sprintf("Group ACL '%s' deleted successfully", groupName),
	}, nil
}

// Helper functions for converting between GraphQL models and ACL types

// convertPermissionsToACL converts GraphQL PermissionInput to ACL format
func convertPermissionsToACL(permissions []*model.PermissionInput) map[string]string {
	result := make(map[string]string)
	for _, perm := range permissions {
		key := fmt.Sprintf("%s#*", perm.Table) // Table-level permission format
		result[key] = perm.Action
	}
	return result
}

// convertFieldFiltersToACL converts GraphQL FieldFilterInput to ACL format
func convertFieldFiltersToACL(filters []*model.FieldFilterInput) map[string]acl.FieldFilter {
	result := make(map[string]acl.FieldFilter)
	for _, filter := range filters {
		result[filter.Field] = acl.FieldFilter{
			Field:       filter.Field,
			IncludeList: filter.IncludeList,
			ExcludeList: filter.ExcludeList,
			FilterType:  filter.FilterType,
		}
	}
	return result
}

// convertACLToGraphQL converts ACL MergedACL to GraphQL ACLRecord
func convertACLToGraphQL(acl *acl.MergedACL, principalID string) *model.ACLRecord {
	// Convert permissions map to GraphQL Permission slice
	var permissions []*model.Permission
	for key, action := range acl.Permissions {
		// Parse table from key format "table#*"
		parts := strings.Split(key, "#")
		if len(parts) >= 1 {
			permissions = append(permissions, &model.Permission{
				Table:  parts[0],
				Action: action,
			})
		}
	}

	// Convert field filters map to GraphQL FieldFilter slice
	var fieldFilters []*model.FieldFilter
	for _, filter := range acl.FieldFilters {
		fieldFilters = append(fieldFilters, &model.FieldFilter{
			Field:       filter.Field,
			IncludeList: filter.IncludeList,
			ExcludeList: filter.ExcludeList,
			FilterType:  filter.FilterType,
		})
	}

	return &model.ACLRecord{
		PrincipalID:  principalID,
		Groups:       acl.Groups,
		Permissions:  permissions,
		FieldFilters: fieldFilters,
		UpdatedAt:    acl.CachedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// convertACLRecordToGraphQL converts ACL ACLRecord to GraphQL ACLRecord
func convertACLRecordToGraphQL(record *acl.ACLRecord) *model.ACLRecord {
	// Convert permissions map to GraphQL Permission slice
	var permissions []*model.Permission
	for key, action := range record.Permissions {
		// Parse table from key format "table#*"
		parts := strings.Split(key, "#")
		if len(parts) >= 1 {
			permissions = append(permissions, &model.Permission{
				Table:  parts[0],
				Action: action,
			})
		}
	}

	// Convert field filters map to GraphQL FieldFilter slice
	var fieldFilters []*model.FieldFilter
	for _, filter := range record.FieldFilters {
		fieldFilters = append(fieldFilters, &model.FieldFilter{
			Field:       filter.Field,
			IncludeList: filter.IncludeList,
			ExcludeList: filter.ExcludeList,
			FilterType:  filter.FilterType,
		})
	}

	return &model.ACLRecord{
		PrincipalID:  record.PrincipalID,
		Groups:       record.Groups,
		Permissions:  permissions,
		FieldFilters: fieldFilters,
		UpdatedAt:    record.UpdatedAt,
	}
}
