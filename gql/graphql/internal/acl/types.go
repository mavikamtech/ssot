package acl

import (
	"time"
)

// ACLRecord represents a user or group entry in DynamoDB
// Each entry contains the principal ID (email for users, group:name for groups)
// and their associated permissions
type ACLRecord struct {
	PrincipalID string            `dynamodbav:"PrincipalID"` // "paul@mavik.com" or "group:admin"
	Groups      []string          `dynamodbav:"Groups"`      // User's group memberships (empty for group entries)
	Permissions map[string]string `dynamodbav:"Permissions"` // Permission mappings
	UpdatedAt   string            `dynamodbav:"UpdatedAt"`   // Last update timestamp
}

// PermissionAction defines the allowed actions
type PermissionAction string

const (
	ActionRead      PermissionAction = "read"
	ActionWrite     PermissionAction = "write"
	ActionReadWrite PermissionAction = "readwrite"
	ActionBlocking  PermissionAction = "blocking" // Explicitly blocks access
)

// Permission represents a specific permission level
type Permission struct {
	Table   string           // Table name (e.g., "LoanCache")
	Columns []string         // Column names (e.g., ["Balance", "InterestRate"] or ["*"] for all)
	Action  PermissionAction // Action allowed
}

// CacheEntry represents a cached ACL record with TTL
type CacheEntry struct {
	ACL       *ACLRecord
	ExpiresAt time.Time
}

// IsExpired checks if the cache entry has expired
func (c *CacheEntry) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// NewCacheEntry creates a new cache entry with TTL
func NewCacheEntry(acl *ACLRecord, ttl time.Duration) *CacheEntry {
	return &CacheEntry{
		ACL:       acl,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// MergedACL represents the final ACL after merging user and group permissions
type MergedACL struct {
	UserEmail   string            // Original user email
	Permissions map[string]string // All merged permissions
	Groups      []string          // User's groups
	CachedAt    time.Time         // When this was cached
}

// ColumnPermissions represents column-level access control
type ColumnPermissions struct {
	Table             string            // Table name
	ColumnAccess      map[string]string // Column name -> "allowed"/"blocked"
	UsedScopeFallback bool              // Whether scope fallback was used instead of ACL
	AllowedColumns    []string          // List of allowed column names (for convenience)
	BlockedColumns    []string          // List of blocked column names (for convenience)
}

// NewColumnPermissions creates a ColumnPermissions from a map
func NewColumnPermissions(table string, columnAccess map[string]string, usedScopeFallback bool) *ColumnPermissions {
	cp := &ColumnPermissions{
		Table:             table,
		ColumnAccess:      columnAccess,
		UsedScopeFallback: usedScopeFallback,
		AllowedColumns:    []string{},
		BlockedColumns:    []string{},
	}

	// Populate convenience lists
	for column, access := range columnAccess {
		if access == "allowed" {
			cp.AllowedColumns = append(cp.AllowedColumns, column)
		} else {
			cp.BlockedColumns = append(cp.BlockedColumns, column)
		}
	}

	return cp
}

// IsAllowed checks if a specific column is allowed
func (cp *ColumnPermissions) IsAllowed(column string) bool {
	return cp.ColumnAccess[column] == "allowed"
}

// IsBlocked checks if a specific column is blocked
func (cp *ColumnPermissions) IsBlocked(column string) bool {
	return cp.ColumnAccess[column] == "blocked"
}

// CanAccess checks if the merged ACL allows a specific action
func (m *MergedACL) CanAccess(table, column, action string) bool {
	// Check for explicit blocking first (highest priority)
	exactKey := table + "#" + column
	if perm, exists := m.Permissions[exactKey]; exists && perm == "blocking" {
		return false
	}

	wildcardKey := table + "#*"
	if perm, exists := m.Permissions[wildcardKey]; exists && perm == "blocking" {
		return false
	}

	if perm, exists := m.Permissions["*#*"]; exists && perm == "blocking" {
		return false
	}

	// Now check for positive permissions
	// Try exact match first: "LoanCache#Balance"
	if perm, exists := m.Permissions[exactKey]; exists {
		return hasPermission(perm, action)
	}

	// Try wildcard match: "LoanCache#*"
	if perm, exists := m.Permissions[wildcardKey]; exists {
		return hasPermission(perm, action)
	}

	// Try global permission: "*#*"
	if perm, exists := m.Permissions["*#*"]; exists {
		return hasPermission(perm, action)
	}

	// No permission found
	return false
}

// hasPermission checks if a permission string allows the requested action
func hasPermission(permission, action string) bool {
	switch action {
	case "read":
		return permission == "read" || permission == "readwrite"
	case "write":
		return permission == "readwrite"
	case "blocking":
		return false // blocking always denies access
	default:
		return false
	}
}
