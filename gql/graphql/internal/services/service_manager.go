package services

import (
	"os"
	"time"

	"ssot/gql/graphql/graph/services"
	"ssot/gql/graphql/internal/acl"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type ServiceManager struct {
	LoanCashFlowService *services.LoanCashFlowService
	ACLService          *acl.ACLService
	ACLMiddleware       *acl.ACLMiddleware
	// Future services can be added here
	// LoanInfoService     *services.LoanInfoService
	// PropertyService     *services.PropertyService
}

type ServiceConfig struct {
	DynamoClient          *dynamodb.Client
	LoanCashFlowTableName string
	ACLTableName          string
	ACLCacheTTL           time.Duration
	// Future table names can be added here
	// LoanInfoTableName          string
	// PropertyTableName          string
}

func NewServiceManager(config ServiceConfig) *ServiceManager {
	// Initialize ACL components
	aclRepo := acl.NewDynamoRepository(config.DynamoClient, config.ACLTableName)
	aclService := acl.NewACLService(aclRepo, config.ACLCacheTTL)
	aclMiddleware := acl.NewACLMiddleware(aclService)

	return &ServiceManager{
		LoanCashFlowService: services.NewLoanCashFlowService(
			config.DynamoClient,
			config.LoanCashFlowTableName,
		),
		ACLService:    aclService,
		ACLMiddleware: aclMiddleware,
		// Future service initializations can be added here
	}
}

func LoadServiceConfigFromEnv(dynamoClient *dynamodb.Client) ServiceConfig {
	return ServiceConfig{
		DynamoClient:          dynamoClient,
		LoanCashFlowTableName: getLoanCashFlowTableName(),
		ACLTableName:          getACLTableName(),
		ACLCacheTTL:           15 * time.Minute, // 15 minute cache TTL
		// Future environment variable mappings can be added here
		// LoanInfoTableName:     getEnvWithDefault("LOAN_INFO_TABLE_NAME", "pbi-loaninfo"),
		// PropertyTableName:     getEnvWithDefault("PROPERTY_TABLE_NAME", "pbi-property"),
	}
}

func getLoanCashFlowTableName() string {
	// TODO: we will have "pbi-loancashflow-staging" for lower envs
	if tableName := os.Getenv("LOAN_CASHFLOW_TABLE_NAME"); tableName != "" {
		return tableName
	}

	env := getEnvWithDefault("ENV", "")
	switch env {
	case "prod":
		return "pbi-loancashflow-prod"
	case "staging":
		return "pbi-loancashflow"
	default:
		return "pbi-loancashflow"
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getACLTableName() string {
	if tableName := os.Getenv("ACL_TABLE_NAME"); tableName != "" {
		return tableName
	}

	env := getEnvWithDefault("ENV", "")
	switch env {
	case "prod":
		return "ssot-gql-acl-prod"
	case "staging":
		return "ssot-gql-acl-staging"
	default:
		return "ssot-gql-acl-staging" // Default for development
	}
}
