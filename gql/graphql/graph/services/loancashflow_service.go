package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"ssot/gql/graphql/graph/model"
	"ssot/gql/graphql/internal/auth"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var loancashflowReadScope = "ssot:gql:loancashflow:read"

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

func (s *LoanCashFlowService) GetByLoanCode(ctx context.Context, loanCode string) ([]*model.LoanCashFlow, error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}
	if !strings.Contains(user.Scope, loancashflowReadScope) {
		return nil, fmt.Errorf("insufficient scope: missing required scope %s", loancashflowReadScope)
	}

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
		loanCashFlow, err := s.itemToLoanCashFlow(item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert DynamoDB item: %w", err)
		}
		loanCashFlows = append(loanCashFlows, loanCashFlow)
	}

	return loanCashFlows, nil
}

func (s *LoanCashFlowService) itemToLoanCashFlow(item map[string]types.AttributeValue) (*model.LoanCashFlow, error) {
	loanCashFlow := &model.LoanCashFlow{}

	if loanCodeAttr, ok := item["loanCode"]; ok {
		if s, ok := loanCodeAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.LoanCode = s.Value
		}
	}

	if maxHmyAttr, ok := item["maxHmy"]; ok {
		if s, ok := maxHmyAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.MaxHmy = &s.Value
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

	if commitmentAttr, ok := item["commitment"]; ok {
		if n, ok := commitmentAttr.(*types.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				loanCashFlow.Commitment = &val
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

	if periodendAttr, ok := item["periodend"]; ok {
		if s, ok := periodendAttr.(*types.AttributeValueMemberS); ok {
			loanCashFlow.Periodend = &s.Value
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
