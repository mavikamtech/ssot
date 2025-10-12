package services

import (
	"context"
	"fmt"
	"strconv"

	"ssot/gql/graphql/graph/model"
	"ssot/gql/graphql/internal/acl"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// AllLoanCashFlowColumns defines all available columns in the LoanCashFlow table
var AllLoanCashFlowColumns = []string{
	"loancode", "maxHmy", "accrualenddate", "accrualstartdate", "balance",
	"capitalizedFee", "capitalizedInterest", "capitalizedLoanAdministrationFee",
	"capitalizedOtherFees", "commitment", "drawActualPrincipal", "ebalance",
	"glPerioddate", "interest", "leverageActivity", "leverageBalance",
	"leverageInterest", "loandesc", "paymentnumber", "postdate",
	"propertycode", "propertyname", "sbalance", "status",
}

type LoanCashFlowService struct {
	client    *dynamodb.Client
	tableName string
}

func NewLoanCashFlowService(client *dynamodb.Client, tableName string) *LoanCashFlowService {
	return &LoanCashFlowService{
		client:    client,
		tableName: tableName,
	}
}

func (s *LoanCashFlowService) GetByLoanCode(ctx context.Context, loanCode string, columnPermissions *acl.ColumnPermissions) ([]*model.LoanCashFlow, error) {
	// Note: Permission checking is now handled at the resolver level with column-level filtering
	// This service method filters the response based on provided column permissions

	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String("loancode-postdate-maxHmy-index"),
		KeyConditionExpression: aws.String("loancode = :loancodeVal"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":loancodeVal": &types.AttributeValueMemberS{Value: loanCode},
		},
		ScanIndexForward: aws.Bool(true),
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

// GetByLoanCodeWithFieldFilters retrieves loan cash flows and applies field-level filtering
func (s *LoanCashFlowService) GetByLoanCodeWithFieldFilters(ctx context.Context, loanCode string, columnPermissions *acl.ColumnPermissions, fieldFilters map[string]acl.FieldFilter) ([]*model.LoanCashFlow, error) {
	// First get the data with column filtering
	loanCashFlows, err := s.GetByLoanCode(ctx, loanCode, columnPermissions)
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
					return lcf.Loancode
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

func (s *LoanCashFlowService) itemToLoanCashFlowFiltered(item map[string]types.AttributeValue, columnPermissions *acl.ColumnPermissions) (*model.LoanCashFlow, error) {
	loanCashFlow := &model.LoanCashFlow{}

	// Only populate fields that are allowed by column permissions
	if columnPermissions.IsAllowed("loancode") {
		if loanCodeAttr, ok := item["loancode"]; ok {
			if s, ok := loanCodeAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.Loancode = s.Value
			}
		}
	}

	if columnPermissions.IsAllowed("maxHmy") {
		if maxHmyAttr, ok := item["maxHmy"]; ok {
			if s, ok := maxHmyAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.MaxHmy = &s.Value
			}
		}
	}

	if columnPermissions.IsAllowed("accrualenddate") {
		if accrualenddateAttr, ok := item["accrualenddate"]; ok {
			if s, ok := accrualenddateAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.Accrualenddate = &s.Value
			}
		}
	}

	if columnPermissions.IsAllowed("accrualstartdate") {
		if accrualstartdateAttr, ok := item["accrualstartdate"]; ok {
			if s, ok := accrualstartdateAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.Accrualstartdate = &s.Value
			}
		}
	}

	if columnPermissions.IsAllowed("balance") {
		if balanceAttr, ok := item["balance"]; ok {
			if n, ok := balanceAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.Balance = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("capitalizedFee") {
		if capitalizedFeeAttr, ok := item["capitalizedFee"]; ok {
			if n, ok := capitalizedFeeAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.CapitalizedFee = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("capitalizedInterest") {
		if capitalizedInterestAttr, ok := item["capitalizedInterest"]; ok {
			if n, ok := capitalizedInterestAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.CapitalizedInterest = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("capitalizedLoanAdministrationFee") {
		if capitalizedLoanAdministrationFeeAttr, ok := item["capitalizedLoanAdministrationFee"]; ok {
			if n, ok := capitalizedLoanAdministrationFeeAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.CapitalizedLoanAdministrationFee = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("capitalizedOtherFees") {
		if capitalizedOtherFeesAttr, ok := item["capitalizedOtherFees"]; ok {
			if n, ok := capitalizedOtherFeesAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.CapitalizedOtherFees = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("commitment") {
		if commitmentAttr, ok := item["commitment"]; ok {
			if n, ok := commitmentAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.Commitment = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("drawActualPrincipal") {
		if drawActualPrincipalAttr, ok := item["drawActualPrincipal"]; ok {
			if n, ok := drawActualPrincipalAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.DrawActualPrincipal = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("ebalance") {
		if ebalanceAttr, ok := item["ebalance"]; ok {
			if n, ok := ebalanceAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.Ebalance = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("glPerioddate") {
		if glPerioddateAttr, ok := item["glPerioddate"]; ok {
			if s, ok := glPerioddateAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.GlPerioddate = &s.Value
			}
		}
	}

	if columnPermissions.IsAllowed("interest") {
		if interestAttr, ok := item["interest"]; ok {
			if n, ok := interestAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.Interest = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("leverageActivity") {
		if leverageActivityAttr, ok := item["leverageActivity"]; ok {
			if n, ok := leverageActivityAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.LeverageActivity = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("leverageBalance") {
		if leverageBalanceAttr, ok := item["leverageBalance"]; ok {
			if n, ok := leverageBalanceAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.LeverageBalance = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("leverageInterest") {
		if leverageInterestAttr, ok := item["leverageInterest"]; ok {
			if n, ok := leverageInterestAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.LeverageInterest = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("loandesc") {
		if loandescAttr, ok := item["loandesc"]; ok {
			if s, ok := loandescAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.Loandesc = &s.Value
			}
		}
	}

	if columnPermissions.IsAllowed("paymentnumber") {
		if paymentnumberAttr, ok := item["paymentnumber"]; ok {
			if s, ok := paymentnumberAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.Paymentnumber = &s.Value
			}
		}
	}

	if columnPermissions.IsAllowed("postdate") {
		if postdateAttr, ok := item["postdate"]; ok {
			if s, ok := postdateAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.Postdate = &s.Value
			}
		}
	}

	if columnPermissions.IsAllowed("propertycode") {
		if propertycodeAttr, ok := item["propertycode"]; ok {
			if s, ok := propertycodeAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.Propertycode = &s.Value
			}
		}
	}

	if columnPermissions.IsAllowed("propertyname") {
		if propertynameAttr, ok := item["propertyname"]; ok {
			if s, ok := propertynameAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.Propertyname = &s.Value
			}
		}
	}

	if columnPermissions.IsAllowed("sbalance") {
		if sbalanceAttr, ok := item["sbalance"]; ok {
			if n, ok := sbalanceAttr.(*types.AttributeValueMemberN); ok {
				if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
					loanCashFlow.Sbalance = &val
				}
			}
		}
	}

	if columnPermissions.IsAllowed("status") {
		if statusAttr, ok := item["status"]; ok {
			if s, ok := statusAttr.(*types.AttributeValueMemberS); ok {
				loanCashFlow.Status = &s.Value
			}
		}
	}

	return loanCashFlow, nil
}

func (s *LoanCashFlowService) itemToLoanCashFlow(item map[string]types.AttributeValue) (*model.LoanCashFlow, error) {
	loanCashFlow := &model.LoanCashFlow{}

	if loanCodeAttr, ok := item["loancode"]; ok {
		if s, ok := loanCodeAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.Loancode = s.Value
		}
	}

	if maxHmyAttr, ok := item["maxHmy"]; ok {
		if s, ok := maxHmyAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.MaxHmy = &s.Value
		}
	}

	if accrualenddateAttr, ok := item["accrualenddate"]; ok {
		if s, ok := accrualenddateAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.Accrualenddate = &s.Value
		}
	}

	if accrualstartdateAttr, ok := item["accrualstartdate"]; ok {
		if s, ok := accrualstartdateAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.Accrualstartdate = &s.Value
		}
	}

	if compositeSortAttr, ok := item["postdate#maxHmy"]; ok {
		if s, ok := compositeSortAttr.(*types.AttributeValueMemberS); ok {
			// This composite key is used for DynamoDB ordering
			_ = s.Value
		}
	}

	if balanceAttr, ok := item["balance"]; ok {
		if n, ok := balanceAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.Balance = &val
			}
		}
	}

	if capitalizedFeeAttr, ok := item["capitalizedFee"]; ok {
		if n, ok := capitalizedFeeAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.CapitalizedFee = &val
			}
		}
	}

	if capitalizedInterestAttr, ok := item["capitalizedInterest"]; ok {
		if n, ok := capitalizedInterestAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.CapitalizedInterest = &val
			}
		}
	}

	if capitalizedLoanAdministrationFeeAttr, ok := item["capitalizedLoanAdministrationFee"]; ok {
		if n, ok := capitalizedLoanAdministrationFeeAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.CapitalizedLoanAdministrationFee = &val
			}
		}
	}

	if capitalizedOtherFeesAttr, ok := item["capitalizedOtherFees"]; ok {
		if n, ok := capitalizedOtherFeesAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.CapitalizedOtherFees = &val
			}
		}
	}

	if commitmentAttr, ok := item["commitment"]; ok {
		if n, ok := commitmentAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.Commitment = &val
			}
		}
	}

	if drawActualPrincipalAttr, ok := item["drawActualPrincipal"]; ok {
		if n, ok := drawActualPrincipalAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.DrawActualPrincipal = &val
			}
		}
	}

	if ebalanceAttr, ok := item["ebalance"]; ok {
		if n, ok := ebalanceAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.Ebalance = &val
			}
		}
	}

	if glPerioddateAttr, ok := item["glPerioddate"]; ok {
		if s, ok := glPerioddateAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.GlPerioddate = &s.Value
		}
	}

	if interestAttr, ok := item["interest"]; ok {
		if n, ok := interestAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.Interest = &val
			}
		}
	}

	if leverageActivityAttr, ok := item["leverageActivity"]; ok {
		if n, ok := leverageActivityAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.LeverageActivity = &val
			}
		}
	}

	if leverageBalanceAttr, ok := item["leverageBalance"]; ok {
		if n, ok := leverageBalanceAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.LeverageBalance = &val
			}
		}
	}

	if leverageInterestAttr, ok := item["leverageInterest"]; ok {
		if n, ok := leverageInterestAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.LeverageInterest = &val
			}
		}
	}

	if loandescAttr, ok := item["loandesc"]; ok {
		if s, ok := loandescAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.Loandesc = &s.Value
		}
	}

	if paymentnumberAttr, ok := item["paymentnumber"]; ok {
		if s, ok := paymentnumberAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.Paymentnumber = &s.Value
		}
	}

	if postdateAttr, ok := item["postdate"]; ok {
		if s, ok := postdateAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.Postdate = &s.Value
		}
	}

	if propertycodeAttr, ok := item["propertycode"]; ok {
		if s, ok := propertycodeAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.Propertycode = &s.Value
		}
	}

	if propertynameAttr, ok := item["propertyname"]; ok {
		if s, ok := propertynameAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.Propertyname = &s.Value
		}
	}

	if sbalanceAttr, ok := item["sbalance"]; ok {
		if n, ok := sbalanceAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.Sbalance = &val
			}
		}
	}

	if statusAttr, ok := item["status"]; ok {
		if s, ok := statusAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.Status = &s.Value
		}
	}

	return loanCashFlow, nil
}
