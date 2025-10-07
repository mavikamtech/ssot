package constants

var UserPoolMap = map[string]string{
	"ssot-gql-staging":    "us-east-1_yjLrQIIRX",
	"ssot-gql-production": "us-east-1_HI7lAfHb0",
}

func GetUserPoolID(envBasedName string) string {
	if poolID, exists := UserPoolMap[envBasedName]; exists {
		return poolID
	}
	return ""
}
