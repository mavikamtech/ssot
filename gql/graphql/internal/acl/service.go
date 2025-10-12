package acl

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ACLService provides access control functionality with caching
type ACLService struct {
	repo  *DynamoRepository
	cache map[string]*CacheEntry
	mutex sync.RWMutex
	ttl   time.Duration
}

// NewACLService creates a new ACL service with the specified TTL
func NewACLService(repo *DynamoRepository, ttl time.Duration) *ACLService {
	service := &ACLService{
		repo:  repo,
		cache: make(map[string]*CacheEntry),
		ttl:   ttl,
	}

	// Start cache cleanup goroutine
	go service.startCacheCleanup()

	return service
}

// GetMergedACL retrieves and merges user and group permissions
func (s *ACLService) GetMergedACL(ctx context.Context, email string) (*MergedACL, error) {
	// Check cache first
	s.mutex.RLock()
	if entry, exists := s.cache[email]; exists && !entry.IsExpired() {
		s.mutex.RUnlock()
		return &MergedACL{
			UserEmail:   email,
			Permissions: entry.ACL.Permissions,
			Groups:      entry.ACL.Groups,
			CachedAt:    time.Now().Add(-s.ttl).Add(time.Until(entry.ExpiresAt)),
		}, nil
	}
	s.mutex.RUnlock()

	// Cache miss or expired, fetch from DynamoDB
	mergedACL, err := s.fetchAndMergeACL(ctx, email)
	if err != nil {
		return nil, err
	}

	// Update cache
	s.mutex.Lock()
	s.cache[email] = NewCacheEntry(&ACLRecord{
		PrincipalID: email,
		Groups:      mergedACL.Groups,
		Permissions: mergedACL.Permissions,
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}, s.ttl)
	s.mutex.Unlock()

	return mergedACL, nil
}

// fetchAndMergeACL fetches user and group data from DynamoDB and merges permissions
func (s *ACLService) fetchAndMergeACL(ctx context.Context, email string) (*MergedACL, error) {
	// Step 1: Get user record
	userRecord, err := s.repo.GetUserRecord(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user record: %w", err)
	}

	// Step 2: Get group records in batch
	groupRecords, err := s.repo.BatchGetGroupRecords(ctx, userRecord.Groups)
	if err != nil {
		return nil, fmt.Errorf("failed to get group records: %w", err)
	}

	// Step 3: Merge permissions (user permissions take precedence)
	mergedPermissions := make(map[string]string)

	// First, add group permissions
	for _, groupRecord := range groupRecords {
		for key, value := range groupRecord.Permissions {
			mergedPermissions[key] = value
		}
	}

	// Then, add user permissions (overrides group permissions)
	for key, value := range userRecord.Permissions {
		mergedPermissions[key] = value
	}

	return &MergedACL{
		UserEmail:   email,
		Permissions: mergedPermissions,
		Groups:      userRecord.Groups,
		CachedAt:    time.Now(),
	}, nil
}

// InvalidateCache removes a user's ACL from cache
func (s *ACLService) InvalidateCache(email string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.cache, email)
}

// InvalidateAllCache clears all cached ACL entries
func (s *ACLService) InvalidateAllCache() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.cache = make(map[string]*CacheEntry)
}

// startCacheCleanup runs a background goroutine to clean expired entries
func (s *ACLService) startCacheCleanup() {
	ticker := time.NewTicker(5 * time.Minute) // Clean every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		s.cleanExpiredEntries()
	}
}

// cleanExpiredEntries removes expired cache entries
func (s *ACLService) cleanExpiredEntries() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for email, entry := range s.cache {
		if now.After(entry.ExpiresAt) {
			delete(s.cache, email)
		}
	}
}

// CreateUser creates a new user ACL record
func (s *ACLService) CreateUser(ctx context.Context, email string, groups []string, permissions map[string]string) error {
	record := &ACLRecord{
		PrincipalID: email,
		Groups:      groups,
		Permissions: permissions,
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	err := s.repo.PutUserRecord(ctx, record)
	if err != nil {
		return err
	}

	// Invalidate cache for this user
	s.InvalidateCache(email)
	return nil
}

// CreateGroup creates a new group ACL record
func (s *ACLService) CreateGroup(ctx context.Context, groupName string, permissions map[string]string) error {
	if !isGroupName(groupName) {
		groupName = "group:" + groupName
	}

	record := &ACLRecord{
		PrincipalID: groupName,
		Groups:      []string{}, // Groups don't have group memberships
		Permissions: permissions,
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	err := s.repo.PutGroupRecord(ctx, record)
	if err != nil {
		return err
	}

	// Invalidate all cache since group permissions affect multiple users
	s.InvalidateAllCache()
	return nil
}

// UpdateUserGroups updates a user's group memberships
func (s *ACLService) UpdateUserGroups(ctx context.Context, email string, groups []string) error {
	// Get current user record
	userRecord, err := s.repo.GetUserRecord(ctx, email)
	if err != nil {
		return err
	}

	// Update groups
	userRecord.Groups = groups
	err = s.repo.PutUserRecord(ctx, userRecord)
	if err != nil {
		return err
	}

	// Invalidate cache for this user
	s.InvalidateCache(email)
	return nil
}

// UpdateUserPermissions updates a user's direct permissions
func (s *ACLService) UpdateUserPermissions(ctx context.Context, email string, permissions map[string]string) error {
	// Get current user record
	userRecord, err := s.repo.GetUserRecord(ctx, email)
	if err != nil {
		return err
	}

	// Update permissions
	userRecord.Permissions = permissions
	err = s.repo.PutUserRecord(ctx, userRecord)
	if err != nil {
		return err
	}

	// Invalidate cache for this user
	s.InvalidateCache(email)
	return nil
}

// UpdateGroupPermissions updates a group's permissions
func (s *ACLService) UpdateGroupPermissions(ctx context.Context, groupName string, permissions map[string]string) error {
	if !isGroupName(groupName) {
		groupName = "group:" + groupName
	}

	// Get current group record
	groupRecord, err := s.repo.GetUserRecord(ctx, groupName) // Using GetUserRecord since it's the same operation
	if err != nil {
		return err
	}

	// Update permissions
	groupRecord.Permissions = permissions
	err = s.repo.PutGroupRecord(ctx, groupRecord)
	if err != nil {
		return err
	}

	// Invalidate all cache since group permissions affect multiple users
	s.InvalidateAllCache()
	return nil
}

// DeleteUser removes a user ACL record
func (s *ACLService) DeleteUser(ctx context.Context, email string) error {
	err := s.repo.DeleteRecord(ctx, email)
	if err != nil {
		return err
	}

	s.InvalidateCache(email)
	return nil
}

// DeleteGroup removes a group ACL record
func (s *ACLService) DeleteGroup(ctx context.Context, groupName string) error {
	if !isGroupName(groupName) {
		groupName = "group:" + groupName
	}

	err := s.repo.DeleteRecord(ctx, groupName)
	if err != nil {
		return err
	}

	// Invalidate all cache since group deletion affects multiple users
	s.InvalidateAllCache()
	return nil
}

// ListAllRecords returns all ACL records (for administrative purposes)
func (s *ACLService) ListAllRecords(ctx context.Context) ([]*ACLRecord, error) {
	return s.repo.ListRecords(ctx)
}

// GetCacheStats returns cache statistics
func (s *ACLService) GetCacheStats() (int, int) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	total := len(s.cache)
	expired := 0
	now := time.Now()

	for _, entry := range s.cache {
		if now.After(entry.ExpiresAt) {
			expired++
		}
	}

	return total, expired
}

// isGroupName checks if a string is already a group name (starts with "group:")
func isGroupName(name string) bool {
	return len(name) > 6 && name[:6] == "group:"
}
