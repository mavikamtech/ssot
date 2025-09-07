package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/pkg/errors"
)

var (
	powerbiApplicationSecret = os.Getenv("POWERBI_APPLICATION_SECRET_NAME")
	powerbiApplicationConfig = os.Getenv("POWERBI_APPLICATION_CONFIG_NAME")
	powerbiEndpoint          = os.Getenv("ENDPOINT")
	secretEndpoint           = os.Getenv("SECRET_ENDPOINT")
	tableName                = os.Getenv("DYNAMO_TABLE_NAME")
)

func handler(ctx context.Context) error {
	awscfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return fmt.Errorf("load aws config failed: %w", err)
	}
	dynamoClient := dynamodb.NewFromConfig(awscfg)

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic recovery:", r)
		}
	}()

	resp, err := getApiResponse(ctx, awscfg)
	if err != nil {
		fmt.Println("error from api request:", err)
		return err
	}

	keyNormalizer := NewNormalizer("Dim_Job[", "]")

	for _, r := range resp.Results {
		for _, t := range r.Tables {
			for _, row := range t.Rows {
				item := map[string]types.AttributeValue{}

				for key, val := range row {
					formattedKey, formattedVal := parseDbField(key, val, keyNormalizer)
					item[formattedKey] = formattedVal
				}

				_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
					TableName: aws.String(tableName),
					Item:      item,
				})
				if err != nil {
					log.Printf("failed to insert %s: %v", item, err)
				}
			}
		}

	}

	return nil
}

func getSecret(ctx context.Context, awsCfg aws.Config, key string, dst interface{}) error {
	secretManagerSession := secretsmanager.NewFromConfig(awsCfg)

	var result *secretsmanager.GetSecretValueOutput
	input := &secretsmanager.GetSecretValueInput{
		SecretId: &key,
	}

	result, err := secretManagerSession.GetSecretValue(ctx, input)
	if err != nil {
		return err
	}
	r := json.NewDecoder(strings.NewReader(*result.SecretString))
	r.UseNumber()
	return r.Decode(&dst)
}

func getToken(ctx context.Context, awsCfg aws.Config, key string) (string, error) {
	dst := powerbiApiSecret{}

	err := getSecret(ctx, awsCfg, key, &dst)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s/oauth2/v2.0/token", secretEndpoint, dst.TenantId)

	payload := strings.NewReader("grant_type=client_credentials&client_id=" + dst.ClientId + "&client_secret=" + dst.ClientSecret + "&scope=https%3A%2F%2Fanalysis.windows.net%2Fpowerbi%2Fapi%2F.default")

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, payload)

	if err != nil {
		return "", errors.Errorf("http request error: %s", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return "", errors.Errorf("secret api error: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("getToken bad request, Status Code: %v", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.Errorf("json request decoding error: %s", err)
	}

	var secretResult secretResult
	err = json.Unmarshal(body, &secretResult)

	return secretResult.AccessToken, err
}

func getApiResponse(ctx context.Context, awsCfg aws.Config) (*apiResponse, error) {
	token, err := getToken(ctx, awsCfg, powerbiApplicationSecret)
	if err != nil {
		return nil, err
	}

	var dst = powerbiApplicationCfg{}

	err = getSecret(ctx, awsCfg, powerbiApplicationConfig, &dst)
	if err != nil {
		return nil, err
	}

	powerBiUrl := fmt.Sprintf("%s/v1.0/myorg/groups/%s/datasets/%s/executeQueries", powerbiEndpoint, dst.WorkspaceID, dst.DatasetID)

	payload := strings.NewReader(`{
  "queries": [
    {
      "query": "EVALUATE VALUES(Dim_Job)"
    }
  ]
}
`)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, powerBiUrl, payload)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getData bad request, Status Code: %v", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	var response apiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &response, nil
}

func main() {
	lambda.Start(handler)
}

func parseDbField(key string, value any, keyNormalizer func(string) string) (formatedKey string, formatedValue *types.AttributeValueMemberS) {
	formatedKey = keyNormalizer(key)

	switch val := value.(type) {
	case string:
		formatedValue = &types.AttributeValueMemberS{Value: strings.Trim(val, " ")}
	case float64, float32, int, bool:
		formatedValue = &types.AttributeValueMemberS{Value: fmt.Sprintf("%v", val)}
	case nil:
		fmt.Println("Got a nil value")
	default:
		fmt.Printf("Got an unknown type %T: %v\n", val, val)
	}
	return
}

func removePrefixAndSuffix(s, prefix, suffix string) string {
	s = strings.TrimPrefix(s, prefix)
	s = strings.TrimSuffix(s, suffix)
	return s
}

func convertKeyToCamel(key string) string {
	key = strings.TrimSpace(key)
	words := strings.Fields(strings.ReplaceAll(key, "_", " "))
	if len(words) == 0 {
		return ""
	}

	result := strings.ToLower(words[0])

	for _, w := range words[1:] {
		if w == "" {
			continue
		}
		runes := []rune(strings.ToLower(w))
		runes[0] = unicode.ToUpper(runes[0])
		result += string(runes)
	}
	return result
}

func NewNormalizer(prefix, suffix string) func(string) string {
	return func(key string) string {
		cleaned := removePrefixAndSuffix(key, prefix, suffix)
		return convertKeyToCamel(cleaned)
	}
}
