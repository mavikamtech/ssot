package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xuri/excelize/v2"
)

var tableName = "test-poc"

// Lambda Handler
func Handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	awscfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return fmt.Errorf("load aws config failed: %w", err)
	}
	dynamoClient := dynamodb.NewFromConfig(awscfg)
	s3Client := s3.NewFromConfig(awscfg)

	for _, record := range sqsEvent.Records {
		fmt.Printf("SQS Message: %s\n", record.Body)

		var snsMsg events.SNSEntity
		if err := json.Unmarshal([]byte(record.Body), &snsMsg); err != nil {
			log.Printf("unmarshal SNS failed: %v", err)
			continue
		}

		var s3Event events.S3Event
		if err := json.Unmarshal([]byte(snsMsg.Message), &s3Event); err != nil {
			log.Printf("unmarshal S3 event failed: %v", err)
			continue
		}

		for _, rec := range s3Event.Records {
			bucket := rec.S3.Bucket.Name
			key := rec.S3.Object.Key
			fmt.Printf("Processing file from S3: bucket=%s key=%s\n", bucket, key)

			obj, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: &bucket,
				Key:    &key,
			})
			if err != nil {
				log.Printf("get S3 object failed: %v", err)
				continue
			}
			defer obj.Body.Close()

			f, err := excelize.OpenReader(obj.Body)
			if err != nil {
				log.Printf("open excel failed: %v", err)
				continue
			}

			sheets := f.GetSheetList()
			if len(sheets) == 0 {
				log.Printf("no sheets found in excel")
				continue
			}
			sheetName := sheets[0]
			log.Printf("Using sheet: %s", sheetName)

			rows, err := f.GetRows(sheetName)
			if err != nil {
				log.Printf("read rows failed: %v", err)
				continue
			}

			for i, row := range rows {

				if i < 7 {
					continue
				}

				if len(row) < 6 {
					continue
				}

				investmentName := strings.TrimSpace(row[1])
				if investmentName == "" || investmentName == "Investment Name" {
					continue
				}

				periodStr := strings.ReplaceAll(row[2], ",", "")
				periodVal, _ := strconv.ParseFloat(periodStr, 64)

				yearStr := strings.ReplaceAll(row[4], ",", "")
				yearVal, _ := strconv.ParseFloat(yearStr, 64)

				item := map[string]types.AttributeValue{
					"investmentName": &types.AttributeValueMemberS{Value: investmentName},
					"periodToDate":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", periodVal)},
					"yearToDate":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", yearVal)},
				}

				_, err := dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
					TableName: &tableName,
					Item:      item,
				})
				if err != nil {
					log.Printf("dynamo put failed for %s: %v", investmentName, err)
				} else {
					log.Printf("✅ Inserted %s into DynamoDB", investmentName)
				}
			}
		}
	}
	return nil
}

func main() {
	lambda.Start(Handler)
}
