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

	return item
}

// unmarshalACLRecord converts DynamoDB item to ACLRecord
func (r *DynamoRepository) unmarshalACLRecord(item map[string]types.AttributeValue) *ACLRecord {
	record := &ACLRecord{
		Groups:      []string{},
		Permissions: make(map[string]string),
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

	return record
}
