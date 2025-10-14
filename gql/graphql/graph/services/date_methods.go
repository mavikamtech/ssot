package services

import (
	"context"
	"fmt"
	"time"

	"ssot/gql/graphql/graph/model"
	"ssot/gql/graphql/internal/acl"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// GetByLoanCodesWithEndDate retrieves loan cash flows for multiple loan codes with optional end date filtering
func (s *LoanCashFlowService) GetByLoanCodesWithEndDate(ctx context.Context, loanCodes []*string, endDate *string, columnPermissions *acl.ColumnPermissions) ([]*model.LoanCashFlow, error) {
	// If loanCodes array is empty or nil, get all loans
	if len(loanCodes) == 0 {
		return s.GetAllLoansWithEndDate(ctx, endDate, columnPermissions)
	}

	var allLoanCashFlows []*model.LoanCashFlow

	// Query for each loan code individually
	for _, loanCodePtr := range loanCodes {
		if loanCodePtr == nil {
			continue // Skip nil loan codes
		}

		loanCode := *loanCodePtr
		loanCashFlows, err := s.GetByLoanCodeWithEndDate(ctx, loanCode, endDate, columnPermissions)
		if err != nil {
			return nil, fmt.Errorf("failed to get loan cash flows for loan code %s: %w", loanCode, err)
		}

		allLoanCashFlows = append(allLoanCashFlows, loanCashFlows...)
	}

	return allLoanCashFlows, nil
}

// GetByLoanCodeWithEndDate retrieves loan cash flows for a single loan code with optional end date filtering
func (s *LoanCashFlowService) GetByLoanCodeWithEndDate(ctx context.Context, loanCode string, endDate *string, columnPermissions *acl.ColumnPermissions) ([]*model.LoanCashFlow, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String("loancode-postdate-maxHmy-index"),
		KeyConditionExpression: aws.String("loancode = :loancodeVal"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":loancodeVal": &types.AttributeValueMemberS{Value: loanCode},
		},
		ScanIndexForward: aws.Bool(true),
	}

	// Add date filtering if endDate is provided
	if endDate != nil && *endDate != "" {
		// Parse the date in MM/dd/yyyy format
		parsedDate, err := time.Parse("01/02/2006", *endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid date format, expected MM/dd/yyyy: %w", err)
		}

		// Format to the expected DynamoDB date format (assuming it's stored as YYYY-MM-DD or similar)
		// You may need to adjust this format based on how dates are stored in your DynamoDB
		formattedDate := parsedDate.Format("2006-01-02")

		// Add filter expression for postdate
		input.FilterExpression = aws.String("postdate <= :endDateVal")
		if input.ExpressionAttributeValues == nil {
			input.ExpressionAttributeValues = make(map[string]types.AttributeValue)
		}
		input.ExpressionAttributeValues[":endDateVal"] = &types.AttributeValueMemberS{Value: formattedDate}
	}

	result, err := s.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query DynamoDB: %w", err)
	}

	var loanCashFlows []*model.LoanCashFlow
	for _, item := range result.Items {
		loanCashFlow, err := s.itemToLoanCashFlowFiltered(item, columnPermissions)
		if err != nil {
			return nil, fmt.Errorf("failed to convert DynamoDB item: %w", err)
		}
		loanCashFlows = append(loanCashFlows, loanCashFlow)
	}

	return loanCashFlows, nil
}

// GetAllLoansWithEndDate retrieves all loan cash flows with optional end date filtering
func (s *LoanCashFlowService) GetAllLoansWithEndDate(ctx context.Context, endDate *string, columnPermissions *acl.ColumnPermissions) ([]*model.LoanCashFlow, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	}

	// Add date filtering if endDate is provided
	if endDate != nil && *endDate != "" {
		// Parse the date in MM/dd/yyyy format
		parsedDate, err := time.Parse("01/02/2006", *endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid date format, expected MM/dd/yyyy: %w", err)
		}

		// Format to the expected DynamoDB date format
		formattedDate := parsedDate.Format("2006-01-02")

		// Add filter expression for postdate
		input.FilterExpression = aws.String("postdate <= :endDateVal")
		input.ExpressionAttributeValues = map[string]types.AttributeValue{
			":endDateVal": &types.AttributeValueMemberS{Value: formattedDate},
		}
	}

	result, err := s.client.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan loan cash flows: %w", err)
	}

	var loanCashFlows []*model.LoanCashFlow
	for _, item := range result.Items {
		loanCashFlow, err := s.itemToLoanCashFlowFiltered(item, columnPermissions)
		if err != nil {
			return nil, fmt.Errorf("failed to convert DynamoDB item: %w", err)
		}
		loanCashFlows = append(loanCashFlows, loanCashFlow)
	}

	return loanCashFlows, nil
}

// GetByLoanCodesWithEndDateAndFieldFilters retrieves loan cash flows for multiple loan codes with end date and field filtering
func (s *LoanCashFlowService) GetByLoanCodesWithEndDateAndFieldFilters(ctx context.Context, loanCodes []*string, endDate *string, columnPermissions *acl.ColumnPermissions, fieldFilters map[string]acl.FieldFilter) ([]*model.LoanCashFlow, error) {
	// First get the data with column and date filtering
	loanCashFlows, err := s.GetByLoanCodesWithEndDate(ctx, loanCodes, endDate, columnPermissions)
	if err != nil {
		return nil, err
	}

	// Apply field filters
	if len(fieldFilters) == 0 {
		return loanCashFlows, nil
	}

	// Convert to []any for filtering
	var dataInterface []any
	for _, loanCashFlow := range loanCashFlows {
		dataInterface = append(dataInterface, loanCashFlow)
	}

	// Apply each field filter
	for fieldName := range fieldFilters {
		var getFieldValue func(any) string

		switch fieldName {
		case "loancode":
			getFieldValue = func(item any) string {
				if lcf, ok := item.(*model.LoanCashFlow); ok {
					return lcf.LoanCode
				}
				return ""
			}
		default:
			// Skip unknown fields
			continue
		}

		dataInterface = acl.FilterArrayByField(dataInterface, fieldName, fieldFilters, getFieldValue)
	}

	// Convert back to []*model.LoanCashFlow
	var filteredLoanCashFlows []*model.LoanCashFlow
	for _, item := range dataInterface {
		if lcf, ok := item.(*model.LoanCashFlow); ok {
			filteredLoanCashFlows = append(filteredLoanCashFlows, lcf)
		}
	}

	return filteredLoanCashFlows, nil
}
