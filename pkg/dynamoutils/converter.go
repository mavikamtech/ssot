package dynamoutils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ParseValue determines the appropriate DynamoDB attribute type for a value
func ParseValue(val string) types.AttributeValue {
	// Clean up the value
	trimmedVal := strings.TrimSpace(val)
	if trimmedVal == "" {
		return &types.AttributeValueMemberNULL{Value: true}
	}

	// Remove commas for number parsing
	numVal := strings.ReplaceAll(trimmedVal, ",", "")

	// Try to parse as integer first
	if intVal, err := strconv.ParseInt(numVal, 10, 64); err == nil {
		return &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", intVal)}
	}

	// If not integer, try to parse as float
	if floatVal, err := strconv.ParseFloat(numVal, 64); err == nil {
		return &types.AttributeValueMemberN{Value: fmt.Sprintf("%g", floatVal)}
	}

	// Try to parse as boolean
	lowerVal := strings.ToLower(trimmedVal)
	switch lowerVal {
	case "true", "yes":
		return &types.AttributeValueMemberBOOL{Value: true}
	case "false", "no":
		return &types.AttributeValueMemberBOOL{Value: false}
	}

	// Otherwise, treat as string
	return &types.AttributeValueMemberS{Value: trimmedVal}
}
