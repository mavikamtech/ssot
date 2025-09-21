package dynamoutils

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// InsertItemWithRetry inserts an item into DynamoDB with retry logic
func InsertItemWithRetry(ctx context.Context, client *dynamodb.Client, tableName string, item map[string]types.AttributeValue, maxRetries int) error {
	var err error
	for i := 0; i <= maxRetries; i++ {
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: &tableName,
			Item:      item,
		})

		if err == nil {
			return nil
		}

		// If we've tried the maximum number of times, return the error
		if i == maxRetries {
			return fmt.Errorf("failed to insert item after %d retries: %w", maxRetries, err)
		}

		log.Printf("Retry %d: DynamoDB insert failed: %v", i+1, err)
	}

	return err
}

// GenerateRandomShardId generates a random shard ID between 0 and maxShards-1
func GenerateRandomShardId(maxShards int) int {
	return rand.Intn(maxShards)
}
