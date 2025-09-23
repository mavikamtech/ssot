package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"ssot/api/graphql/internal/services"
)

type Resolver struct {
	ServiceManager *services.ServiceManager
}
