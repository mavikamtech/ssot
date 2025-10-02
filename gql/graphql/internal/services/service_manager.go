package services

import (
	"os"

	"ssot/gql/graphql/graph/services"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type ServiceManager struct {
	LoanCashFlowService *services.LoanCashFlowService
	// Future services can be added here
	// LoanInfoService     *services.LoanInfoService
	// PropertyService     *services.PropertyService
}

type ServiceConfig struct {
	DynamoClient          *dynamodb.Client
	LoanCashFlowTableName string
	// Future table names can be added here
	// LoanInfoTableName          string
	// PropertyTableName          string
}

func NewServiceManager(config ServiceConfig) *ServiceManager {
	return &ServiceManager{
		LoanCashFlowService: services.NewLoanCashFlowService(
			config.DynamoClient,
			config.LoanCashFlowTableName,
		),
		// Future service initializations can be added here
	}
}

func LoadServiceConfigFromEnv(dynamoClient *dynamodb.Client) ServiceConfig {
	return ServiceConfig{
		DynamoClient:          dynamoClient,
		LoanCashFlowTableName: getEnvWithDefault("LOAN_CASHFLOW_TABLE_NAME", "pbi-loancashflow"),
		// Future environment variable mappings can be added here
		// LoanInfoTableName:     getEnvWithDefault("LOAN_INFO_TABLE_NAME", "pbi-loaninfo"),
		// PropertyTableName:     getEnvWithDefault("PROPERTY_TABLE_NAME", "pbi-property"),
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
