package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

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
		headers[i] = toCamelCase(header)
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

			attrValue := parseValue(cellValue)

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
			shardId := rand.Intn(10) // Random number between 0 and 9
			item["loancode#shardId"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#%d", loancode, shardId)}
		} else {
			log.Printf("Skipping row %d: missing required postdate or maxHmy values", rowIdx)
			continue // Skip rows without the required key fields
		}

		_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: &tableName,
			Item:      item,
		})

		if err != nil {
			log.Printf("DynamoDB insert failed for row %d: %v", rowIdx, err)
		} else {
			log.Printf("✅ Inserted row %d into DynamoDB", rowIdx)
		}
	}

	return nil
}

func main() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
	lambda.Start(Handler)
}

// Converts header names to camelCase for DynamoDB attribute names
func toCamelCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	words := strings.Fields(strings.ReplaceAll(s, "_", " "))
	if len(words) == 0 {
		return ""
	}

	result := strings.ToLower(words[0])
	for _, w := range words[1:] {
		if w == "" {
			continue
		}
		runes := []rune(strings.ToLower(w))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		result += string(runes)
	}

	return result
}

// Determines the appropriate DynamoDB attribute type for a value
func parseValue(val string) types.AttributeValue {
	// Clean up the value
	trimmedVal := strings.TrimSpace(val)
	if trimmedVal == "" {
		return &types.AttributeValueMemberNULL{Value: true}
	}

	// Remove commas for number parsing
	numVal := strings.ReplaceAll(trimmedVal, ",", "")

	// Try to parse as integer first
	if intVal, err := strconv.ParseInt(numVal, 10, 64); err == nil {
		return &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", intVal)}
	}

	// If not integer, try to parse as float
	if floatVal, err := strconv.ParseFloat(numVal, 64); err == nil {
		return &types.AttributeValueMemberN{Value: fmt.Sprintf("%g", floatVal)}
	}

	// Try to parse as boolean
	lowerVal := strings.ToLower(trimmedVal)
	switch lowerVal {
	case "true", "yes":
		return &types.AttributeValueMemberBOOL{Value: true}
	case "false", "no":
		return &types.AttributeValueMemberBOOL{Value: false}
	}

	// Otherwise, treat as string
	return &types.AttributeValueMemberS{Value: trimmedVal}
}
