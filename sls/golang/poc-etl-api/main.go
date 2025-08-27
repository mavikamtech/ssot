package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var tableName = "test-poc-api"

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

	resp, err := getApiResponse(ctx)
	if err != nil {
		fmt.Println("error from api request:", err)
		return err
	}

	for _, p := range resp.Products {
		dimensions := &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"width":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", p.Dimensions.Width)},
			"height": &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", p.Dimensions.Height)},
			"depth":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", p.Dimensions.Depth)},
		}}

		var tags []types.AttributeValue
		for _, t := range p.Tags {
			tags = append(tags, &types.AttributeValueMemberS{Value: t})
		}

		var reviews []types.AttributeValue
		for _, r := range p.Reviews {
			reviews = append(reviews, &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					"rating":        &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", r.Rating)},
					"comment":       &types.AttributeValueMemberS{Value: r.Comment},
					"date":          &types.AttributeValueMemberS{Value: r.Date},
					"reviewerName":  &types.AttributeValueMemberS{Value: r.ReviewerName},
					"reviewerEmail": &types.AttributeValueMemberS{Value: r.ReviewerEmail},
				},
			})
		}

		item := map[string]types.AttributeValue{
			"id":          &types.AttributeValueMemberS{Value: fmt.Sprintf("%d", p.ID)},
			"title":       &types.AttributeValueMemberS{Value: p.Title},
			"description": &types.AttributeValueMemberS{Value: p.Description},
			"category":    &types.AttributeValueMemberS{Value: p.Category},
			"price":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", p.Price)},
			"discount":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", p.DiscountPercentage)},
			"rating":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", p.Rating)},
			"stock":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", p.Stock)},
			"tags":        &types.AttributeValueMemberL{Value: tags},
			"brand":       &types.AttributeValueMemberS{Value: p.Brand},
			"sku":         &types.AttributeValueMemberS{Value: p.Sku},
			"weight":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", p.Weight)},
			"dimensions":  dimensions,
			"warranty":    &types.AttributeValueMemberS{Value: p.WarrantyInformation},
			"shipping":    &types.AttributeValueMemberS{Value: p.ShippingInformation},
			"status":      &types.AttributeValueMemberS{Value: p.AvailabilityStatus},
			"reviews":     &types.AttributeValueMemberL{Value: reviews},
			"images":      &types.AttributeValueMemberL{Value: []types.AttributeValue{&types.AttributeValueMemberS{Value: strings.Join(p.Images, ",")}}},
			"thumbnail":   &types.AttributeValueMemberS{Value: p.Thumbnail},
		}

		_, err := dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      item,
		})
		if err != nil {
			log.Printf("failed to insert %d: %v", p.ID, err)
		}
	}

	return nil
}

func getApiResponse(ctx context.Context) (*apiResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://dummyjson.com/products", nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad request, Status Code: %v", resp.StatusCode)
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
