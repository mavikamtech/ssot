package acl

import (
	"context"
	"fmt"
	"ssot/gql/graphql/graph/model"
	"ssot/gql/graphql/internal/services"
)

// ACLQueryResolver handles ACL-related queries
type ACLQueryResolver struct {
	ServiceManager *services.ServiceManager
}

// NewACLQueryResolver creates a new ACL query resolver
func NewACLQueryResolver(serviceManager *services.ServiceManager) *ACLQueryResolver {
	return &ACLQueryResolver{
		ServiceManager: serviceManager,
	}
}

// SsotReportsAdministratorConfiguration handles the query for ACL administration
func (r *ACLQueryResolver) SsotReportsAdministratorConfiguration(ctx context.Context) (*model.SsotReportsAdministratorConfiguration, error) {
	// Check admin access
	if err := r.ServiceManager.ACLMiddleware.RequireAdminAccess(ctx); err != nil {
		return nil, fmt.Errorf("access denied: %v", err)
	}

	// Get all ACL records
	records, err := r.ServiceManager.ACLService.ListAllRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list ACL records: %v", err)
	}

	// Convert to GraphQL format
	var gqlRecords []*model.ACLRecord
	for _, record := range records {
		gqlRecord := convertACLRecordToGraphQL(record)
		gqlRecords = append(gqlRecords, gqlRecord)
	}

	return &model.SsotReportsAdministratorConfiguration{
		ListACLRecords: gqlRecords,
	}, nil
}
