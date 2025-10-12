package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"ssot/gql/graphql/graph/mutation/acl"
	"ssot/gql/graphql/internal/services"
)

type Resolver struct {
	ServiceManager *services.ServiceManager
	ACLMutations   *acl.ACLMutationResolver
	ACLQueries     *acl.ACLQueryResolver
}

// NewResolver creates a new resolver with all dependencies
func NewResolver(serviceManager *services.ServiceManager) *Resolver {
	return &Resolver{
		ServiceManager: serviceManager,
		ACLMutations:   acl.NewACLMutationResolver(serviceManager),
		ACLQueries:     acl.NewACLQueryResolver(serviceManager),
	}
}
