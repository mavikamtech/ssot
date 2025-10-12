package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Resp struct {
	Message string `json:"message"`
}

type Post struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type SecretData struct {
	URL          string `json:"url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scope        string `json:"scope"`
}

func getSecret(ctx context.Context, secretName string) (*SecretData, error) {
	// load AWS config (region is us-east-1 for your secret)
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %v", err)
	}

	sm := secretsmanager.NewFromConfig(cfg)

	out, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %v", err)
	}

	var data SecretData
	if err := json.Unmarshal([]byte(aws.ToString(out.SecretString)), &data); err != nil {
		return nil, fmt.Errorf("failed to parse secret JSON: %v", err)
	}
	return &data, nil
}

func updateSecretTokensOnly(ctx context.Context, secretName string, token Post) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %v", err)
	}
	sm := secretsmanager.NewFromConfig(cfg)

	out, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return fmt.Errorf("failed to get secret: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(aws.ToString(out.SecretString)), &m); err != nil {
		return fmt.Errorf("failed to parse secret JSON: %v", err)
	}

	m["access_token"] = token.AccessToken
	m["expires_in"] = token.ExpiresIn
	m["token_type"] = token.TokenType

	updatedJSON, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal updated JSON: %v", err)
	}

	_, err = sm.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(secretName),
		SecretString: aws.String(string(updatedJSON)),
	})
	if err != nil {
		return fmt.Errorf("failed to update secret: %v", err)
	}

	fmt.Println("✅ Secret token fields updated successfully:", secretName)
	return nil
}

func handler(ctx context.Context, req events.LambdaFunctionURLRequest) (Resp, error) {

	raw := os.Getenv("SECRET_NAMES")
	if raw == "" {
		return Resp{}, fmt.Errorf("environment variable SECRET_NAMES is not set")
	}

	parts := strings.Split(raw, ",")
	secretNames := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			secretNames = append(secretNames, p)
		}
	}
	if len(secretNames) == 0 {
		return Resp{}, fmt.Errorf("SECRET_NAMES is empty after parsing")
	}
	fmt.Println("🔧 Secrets to process:", secretNames)

	for _, secretName := range secretNames {
		fmt.Println("=== Processing secret:", secretName, "===")

		secret, err := getSecret(ctx, secretName)
		if err != nil {
			fmt.Printf("❌ Error reading secret %s: %v\n", secretName, err)
			continue
		}

		maskedClientSecret := "****"
		if n := len(secret.ClientSecret); n > 8 {
			maskedClientSecret = secret.ClientSecret[:4] + "****" + secret.ClientSecret[n-2:]
		} else if n > 2 {
			maskedClientSecret = secret.ClientSecret[:2] + "****"
		}

		fmt.Println("URL:", secret.URL)
		fmt.Println("ClientID:", secret.ClientID)
		fmt.Println("Scope:", secret.Scope)
		fmt.Println("ClientSecret(masked):", maskedClientSecret)

		posturl := secret.URL
		body := []byte(
			"grant_type=client_credentials" +
				"&client_id=" + secret.ClientID +
				"&client_secret=" + secret.ClientSecret +
				"&scope=" + secret.Scope)

		r, err := http.NewRequest("POST", posturl, bytes.NewBuffer(body))
		if err != nil {
			fmt.Printf("❌ Error building request for %s: %v\n", secretName, err)
			continue
		}

		r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}
		res, err := client.Do(r)
		if err != nil {
			fmt.Printf("❌ HTTP request failed for %s: %v\n", secretName, err)
			continue
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			fmt.Printf("❌ HTTP request for %s returned status %s\n", secretName, res.Status)
			continue
		}

		post := &Post{}
		if err := json.NewDecoder(res.Body).Decode(post); err != nil {
			fmt.Printf("❌ Failed to decode response for %s: %v\n", secretName, err)
			continue
		}
		maskedToken := "****"
		if n := len(post.AccessToken); n > 10 {
			maskedToken = post.AccessToken[:10] + "...(len=" + fmt.Sprint(n) + ")"
		}

		fmt.Println("AccessToken(masked):", maskedToken)
		fmt.Println("ExpiresIn:", post.ExpiresIn)
		fmt.Println("TokenType:", post.TokenType)

		if err := updateSecretTokensOnly(ctx, secretName, *post); err != nil {
			fmt.Printf("❌ Failed to update secret %s: %v\n", secretName, err)
			continue
		}

		fmt.Println("=== Done:", secretName, "===")
	}

	return Resp{Message: "Hello, the function executed successfully!"}, nil
}

func main() {
	lambda.Start(handler)
}
