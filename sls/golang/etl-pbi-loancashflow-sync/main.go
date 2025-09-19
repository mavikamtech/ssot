package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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

	bucket := "rec.S3.Bucket.Name"
	key := "rec.S3.Object.Key"
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

	// Process headers - convert them to camelCase
	headerRow := rows[0]
	headers := make([]string, len(headerRow))
	for i, header := range headerRow {
		headers[i] = toCamelCase(header)
	}

	// Process data rows (skip the header row)
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]

		// Skip empty rows
		if len(row) == 0 {
			continue
		}

		// Build DynamoDB item
		item := map[string]types.AttributeValue{}
		hasData := false

		// Add row number as id if needed
		item["id"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("row-%d", rowIdx)}

		// Process each cell in the row
		for colIdx, cellValue := range row {
			if colIdx >= len(headers) || headers[colIdx] == "" {
				continue // Skip columns without headers
			}

			// Parse the value
			attrValue := parseValue(cellValue)

			// Skip null values
			if _, isNull := attrValue.(*types.AttributeValueMemberNULL); isNull {
				continue
			}

			// Add to item
			item[headers[colIdx]] = attrValue
			hasData = true
		}

		// Skip rows with no valid data
		if !hasData {
			log.Printf("Skipping row %d: no valid data", rowIdx)
			continue
		}

		// Insert into DynamoDB
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

	// Try to parse as float
	if floatVal, err := strconv.ParseFloat(numVal, 64); err == nil {
		return &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", floatVal)}
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
