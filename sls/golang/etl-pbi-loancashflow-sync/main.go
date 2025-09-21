package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"ssot/pkg/dynamoutils"
	"ssot/pkg/utils"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xuri/excelize/v2"
)

var (
	tableName = os.Getenv("DYNAMO_TABLE_NAME")
)

// Lambda Handler
func Handler(ctx context.Context) error {
	awscfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return fmt.Errorf("load aws config failed: %w", err)
	}
	dynamoClient := dynamodb.NewFromConfig(awscfg)
	s3Client := s3.NewFromConfig(awscfg)

	// Initialize S3 writer for saving processed data
	s3Writer := NewS3Writer(s3Client, "mavik-powerbi-analytics-data", "loan-cashflow")

	bucket := "loancashflow-sync-excel"
	key := "YDC-Response-LoanCashFlow-Camden-Only.xlsx"
	fmt.Printf("Processing file from S3: bucket=%s key=%s\n", bucket, key)

	obj, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		log.Printf("get S3 object failed: %v", err)
		return err
	}
	defer obj.Body.Close()

	f, err := excelize.OpenReader(obj.Body)
	if err != nil {
		log.Printf("open excel failed: %v", err)
		return err
	}

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		log.Printf("no sheets found in excel")
		return fmt.Errorf("no sheets found in excel")
	}
	sheetName := sheets[0]
	log.Printf("Using sheet: %s", sheetName)

	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Printf("read rows failed: %v", err)
		return err
	}

	if len(rows) < 2 {
		log.Printf("not enough rows in sheet, need at least header and one data row")
		return fmt.Errorf("not enough rows in sheet")
	}

	headerRow := rows[0]
	headers := make([]string, len(headerRow))
	for i, header := range headerRow {
		headers[i] = utils.ToCamelCase(header)
	}

	// Find column indices for postdate and maxHmy
	postdateColIdx := -1
	maxHmyColIdx := -1
	loancodeColIdx := -1
	for i, header := range headers {
		if strings.EqualFold(header, "postdate") {
			postdateColIdx = i
		} else if strings.EqualFold(header, "maxHmy") {
			maxHmyColIdx = i
		} else if strings.EqualFold(header, "loancode") {
			loancodeColIdx = i
		}
	}

	// Collect all processed records for S3 upload
	var s3Records []LoanCashFlowRecord

	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]

		if len(row) == 0 {
			continue
		}

		item := map[string]types.AttributeValue{}
		hasData := false

		// Track postdate and maxHmy values
		var postdate, maxHmy, loancode string

		for colIdx, cellValue := range row {
			if colIdx >= len(headers) || headers[colIdx] == "" {
				continue // Skip columns without headers
			}

			attrValue := dynamoutils.ParseValue(cellValue)

			if _, isNull := attrValue.(*types.AttributeValueMemberNULL); isNull {
				continue
			}

			// Save postdate and maxHmy values if found
			switch colIdx {
			case postdateColIdx:
				if sv, ok := attrValue.(*types.AttributeValueMemberS); ok {
					// Parse the date and convert to ISO8601 format
					dateStr := sv.Value
					layout := "1/2/2006 3:04:05 PM" // Go time layout: month/day/year hour:minute:second AM/PM
					t, err := time.Parse(layout, dateStr)
					if err == nil {
						// Successfully parsed, convert to ISO8601 format
						postdate = t.Format("2006-01-02T15:04:05")
					} else {
						// If parsing fails, use the original value
						log.Printf("Warning: could not parse date '%s': %v", dateStr, err)
						postdate = dateStr
					}
				}
			case maxHmyColIdx:
				if sv, ok := attrValue.(*types.AttributeValueMemberN); ok {
					maxHmy = sv.Value
				}
			case loancodeColIdx:
				if sv, ok := attrValue.(*types.AttributeValueMemberS); ok {
					loancode = sv.Value
				}
			default:
				item[headers[colIdx]] = attrValue
			}

			hasData = true
		}

		if !hasData {
			log.Printf("Skipping row %d: no valid data", rowIdx)
			continue
		}

		// Set postdate and maxHmy attributes explicitly
		if postdateColIdx != -1 {
			item["postdate"] = &types.AttributeValueMemberS{Value: postdate}
		}
		if maxHmyColIdx != -1 {
			item["maxHmy"] = &types.AttributeValueMemberS{Value: maxHmy}
		}
		if loancodeColIdx != -1 {
			item["loancode"] = &types.AttributeValueMemberS{Value: loancode}
		}

		// Add the required composite key field
		if postdate != "" && maxHmy != "" {
			compositeKey := fmt.Sprintf("%s#%s", postdate, maxHmy)
			item["postdate#maxHmy"] = &types.AttributeValueMemberS{Value: compositeKey}

			// Add a loancode#shardId field with random shardId from 0-9
			shardId := dynamoutils.GenerateRandomShardId(10) // Random number between 0 and 9
			item["loancode#shardId"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#%d", loancode, shardId)}
		} else {
			log.Printf("Skipping row %d: missing required postdate or maxHmy values", rowIdx)
			continue // Skip rows without the required key fields
		}

		err := dynamoutils.InsertItemWithRetry(ctx, dynamoClient, tableName, item, 3)

		if err != nil {
			log.Printf("DynamoDB insert failed for row %d: %v", rowIdx, err)
		} else {
			log.Printf("✅ Inserted row %d into DynamoDB", rowIdx)
		}

		// Convert DynamoDB item to S3 record format
		itemForS3 := make(map[string]interface{})

		// First, process ALL Excel fields (including nulls) for S3
		for colIdx, header := range headers {
			if header == "" {
				continue // Skip columns without headers
			}

			// Get the cell value for this column in this row
			var cellValue string
			if colIdx < len(row) {
				cellValue = row[colIdx]
			}

			// Parse the value and include nulls for S3
			attrValue := dynamoutils.ParseValue(cellValue)
			switch v := attrValue.(type) {
			case *types.AttributeValueMemberS:
				itemForS3[header] = v.Value
			case *types.AttributeValueMemberN:
				itemForS3[header] = v.Value
			case *types.AttributeValueMemberBOOL:
				itemForS3[header] = v.Value
			case *types.AttributeValueMemberNULL:
				// Include null values for S3
				itemForS3[header] = nil
			default:
				itemForS3[header] = fmt.Sprintf("%v", v)
			}
		}

		// Then add the computed fields from DynamoDB item (no nulls here)
		for key, val := range item {
			// Skip if already added from Excel headers
			if _, exists := itemForS3[key]; exists {
				continue
			}

			// Convert DynamoDB AttributeValue to regular interface{}
			switch v := val.(type) {
			case *types.AttributeValueMemberS:
				itemForS3[key] = v.Value
			case *types.AttributeValueMemberN:
				itemForS3[key] = v.Value
			case *types.AttributeValueMemberBOOL:
				itemForS3[key] = v.Value
			default:
				// For other types, convert to string
				itemForS3[key] = fmt.Sprintf("%v", v)
			}
		}

		// Convert to S3 record and add to collection
		s3Record, err := ConvertDynamoItemToS3Record(itemForS3)
		if err != nil {
			log.Printf("Failed to convert row %d to S3 record: %v", rowIdx, err)
		} else {
			s3Records = append(s3Records, s3Record)
		}
	}

	// Upload all processed records to S3
	if len(s3Records) > 0 {
		log.Printf("Uploading %d records to S3...", len(s3Records))
		err := s3Writer.UploadBatch(ctx, s3Records, key)
		if err != nil {
			log.Printf("Failed to upload records to S3: %v", err)
			return fmt.Errorf("S3 upload failed: %w", err)
		}
		log.Printf("✅ Successfully uploaded %d records to S3", len(s3Records))
	} else {
		log.Printf("No records to upload to S3")
	}

	return nil
}

func main() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
	lambda.Start(Handler)
}
