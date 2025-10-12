package acl

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoRepository handles DynamoDB operations for ACL records
type DynamoRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoRepository creates a new DynamoDB repository for ACL operations
func NewDynamoRepository(client *dynamodb.Client, tableName string) *DynamoRepository {
	return &DynamoRepository{
		client:    client,
		tableName: tableName,
	}
}

// GetUserRecord fetches a user's ACL record from DynamoDB
func (r *DynamoRepository) GetUserRecord(ctx context.Context, email string) (*ACLRecord, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PrincipalID": &types.AttributeValueMemberS{Value: email},
		},
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get user record for %s: %w", email, err)
	}

	if result.Item == nil {
		// User not found, return empty record
		return &ACLRecord{
			PrincipalID: email,
			Groups:      []string{},
			Permissions: make(map[string]string),
			UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		}, nil
	}

	record := r.unmarshalACLRecord(result.Item)
	return record, nil
}

// BatchGetGroupRecords fetches multiple group records from DynamoDB
func (r *DynamoRepository) BatchGetGroupRecords(ctx context.Context, groupNames []string) ([]*ACLRecord, error) {
	if len(groupNames) == 0 {
		return []*ACLRecord{}, nil
	}

	// Build keys for batch get
	keys := make([]map[string]types.AttributeValue, 0, len(groupNames))
	for _, groupName := range groupNames {
		keys = append(keys, map[string]types.AttributeValue{
			"PrincipalID": &types.AttributeValueMemberS{Value: groupName},
		})
	}

	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			r.tableName: {
				Keys: keys,
			},
		},
	}

	result, err := r.client.BatchGetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to batch get group records: %w", err)
	}

	// Parse results
	var records []*ACLRecord
	items, exists := result.Responses[r.tableName]
	if !exists {
		return records, nil
	}

	for _, item := range items {
		record := r.unmarshalACLRecord(item)
		records = append(records, record)
	}

	return records, nil
}

// PutUserRecord creates or updates a user's ACL record
func (r *DynamoRepository) PutUserRecord(ctx context.Context, record *ACLRecord) error {
	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	item := r.marshalACLRecord(record)

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	_, err := r.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put user record: %w", err)
	}

	return nil
}

// PutGroupRecord creates or updates a group's ACL record
func (r *DynamoRepository) PutGroupRecord(ctx context.Context, record *ACLRecord) error {
	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	item := r.marshalACLRecord(record)

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	_, err := r.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put group record: %w", err)
	}

	return nil
}

// DeleteRecord deletes a user or group record
func (r *DynamoRepository) DeleteRecord(ctx context.Context, principalID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PrincipalID": &types.AttributeValueMemberS{Value: principalID},
		},
	}

	_, err := r.client.DeleteItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete record for %s: %w", principalID, err)
	}

	return nil
}

// ListRecords lists all ACL records (for administrative purposes)
func (r *DynamoRepository) ListRecords(ctx context.Context) ([]*ACLRecord, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	}

	result, err := r.client.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan ACL records: %w", err)
	}

	var records []*ACLRecord
	for _, item := range result.Items {
		record := r.unmarshalACLRecord(item)
		records = append(records, record)
	}

	return records, nil
}

// marshalACLRecord converts ACLRecord to DynamoDB item
func (r *DynamoRepository) marshalACLRecord(record *ACLRecord) map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		"PrincipalID": &types.AttributeValueMemberS{Value: record.PrincipalID},
		"UpdatedAt":   &types.AttributeValueMemberS{Value: record.UpdatedAt},
	}

	// Marshal Groups (list of strings)
	if len(record.Groups) > 0 {
		groupItems := make([]types.AttributeValue, 0, len(record.Groups))
		for _, group := range record.Groups {
			groupItems = append(groupItems, &types.AttributeValueMemberS{Value: group})
		}
		item["Groups"] = &types.AttributeValueMemberL{Value: groupItems}
	} else {
		item["Groups"] = &types.AttributeValueMemberL{Value: []types.AttributeValue{}}
	}

	// Marshal Permissions (map of string to string)
	if len(record.Permissions) > 0 {
		permItems := make(map[string]types.AttributeValue)
		for key, value := range record.Permissions {
			permItems[key] = &types.AttributeValueMemberS{Value: value}
		}
		item["Permissions"] = &types.AttributeValueMemberM{Value: permItems}
	} else {
		item["Permissions"] = &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{}}
	}

	// Marshal FieldFilters (map of string to FieldFilter)
	if len(record.FieldFilters) > 0 {
		fieldFilterItems := make(map[string]types.AttributeValue)
		for key, filter := range record.FieldFilters {
			// Create a map for the FieldFilter struct
			filterMap := map[string]types.AttributeValue{
				"Field":      &types.AttributeValueMemberS{Value: filter.Field},
				"FilterType": &types.AttributeValueMemberS{Value: filter.FilterType},
			}

			// Marshal IncludeList
			if len(filter.IncludeList) > 0 {
				includeItems := make([]types.AttributeValue, 0, len(filter.IncludeList))
				for _, include := range filter.IncludeList {
					includeItems = append(includeItems, &types.AttributeValueMemberS{Value: include})
				}
				filterMap["IncludeList"] = &types.AttributeValueMemberL{Value: includeItems}
			} else {
				filterMap["IncludeList"] = &types.AttributeValueMemberL{Value: []types.AttributeValue{}}
			}

			// Marshal ExcludeList
			if len(filter.ExcludeList) > 0 {
				excludeItems := make([]types.AttributeValue, 0, len(filter.ExcludeList))
				for _, exclude := range filter.ExcludeList {
					excludeItems = append(excludeItems, &types.AttributeValueMemberS{Value: exclude})
				}
				filterMap["ExcludeList"] = &types.AttributeValueMemberL{Value: excludeItems}
			} else {
				filterMap["ExcludeList"] = &types.AttributeValueMemberL{Value: []types.AttributeValue{}}
			}

			fieldFilterItems[key] = &types.AttributeValueMemberM{Value: filterMap}
		}
		item["FieldFilters"] = &types.AttributeValueMemberM{Value: fieldFilterItems}
	} else {
		item["FieldFilters"] = &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{}}
	}

	return item
}

// unmarshalACLRecord converts DynamoDB item to ACLRecord
func (r *DynamoRepository) unmarshalACLRecord(item map[string]types.AttributeValue) *ACLRecord {
	record := &ACLRecord{
		Groups:       []string{},
		Permissions:  make(map[string]string),
		FieldFilters: make(map[string]FieldFilter),
	}

	// Unmarshal PrincipalID
	if val, ok := item["PrincipalID"]; ok {
		if s, ok := val.(*types.AttributeValueMemberS); ok {
			record.PrincipalID = s.Value
		}
	}

	// Unmarshal UpdatedAt
	if val, ok := item["UpdatedAt"]; ok {
		if s, ok := val.(*types.AttributeValueMemberS); ok {
			record.UpdatedAt = s.Value
		}
	}

	// Unmarshal Groups
	if val, ok := item["Groups"]; ok {
		if l, ok := val.(*types.AttributeValueMemberL); ok {
			for _, groupVal := range l.Value {
				if s, ok := groupVal.(*types.AttributeValueMemberS); ok {
					record.Groups = append(record.Groups, s.Value)
				}
			}
		}
	}

	// Unmarshal Permissions
	if val, ok := item["Permissions"]; ok {
		if m, ok := val.(*types.AttributeValueMemberM); ok {
			for key, permVal := range m.Value {
				if s, ok := permVal.(*types.AttributeValueMemberS); ok {
					record.Permissions[key] = s.Value
				}
			}
		}
	}

	// Unmarshal FieldFilters
	if val, ok := item["FieldFilters"]; ok {
		if m, ok := val.(*types.AttributeValueMemberM); ok {
			for key, filterVal := range m.Value {
				if filterMap, ok := filterVal.(*types.AttributeValueMemberM); ok {
					filter := FieldFilter{}

					// Unmarshal Field
					if fieldVal, exists := filterMap.Value["Field"]; exists {
						if s, ok := fieldVal.(*types.AttributeValueMemberS); ok {
							filter.Field = s.Value
						}
					}

					// Unmarshal FilterType
					if filterTypeVal, exists := filterMap.Value["FilterType"]; exists {
						if s, ok := filterTypeVal.(*types.AttributeValueMemberS); ok {
							filter.FilterType = s.Value
						}
					}

					// Unmarshal IncludeList
					if includeListVal, exists := filterMap.Value["IncludeList"]; exists {
						if l, ok := includeListVal.(*types.AttributeValueMemberL); ok {
							for _, includeVal := range l.Value {
								if s, ok := includeVal.(*types.AttributeValueMemberS); ok {
									filter.IncludeList = append(filter.IncludeList, s.Value)
								}
							}
						}
					}

					// Unmarshal ExcludeList
					if excludeListVal, exists := filterMap.Value["ExcludeList"]; exists {
						if l, ok := excludeListVal.(*types.AttributeValueMemberL); ok {
							for _, excludeVal := range l.Value {
								if s, ok := excludeVal.(*types.AttributeValueMemberS); ok {
									filter.ExcludeList = append(filter.ExcludeList, s.Value)
								}
							}
						}
					}

					record.FieldFilters[key] = filter
				}
			}
		}
	}

	return record
}
