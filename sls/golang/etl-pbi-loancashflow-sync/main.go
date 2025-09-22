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

// renameCamdenToProject renames Camden references to Project-A in cell values
func renameCamdenToProject(cellValue string) string {
	// Replace all instances of Camden with Project-A
	cellValue = strings.ReplaceAll(cellValue, "Camden", "Project-A")
	return cellValue
}

func Handler(ctx context.Context) error {
	awscfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return fmt.Errorf("load aws config failed: %w", err)
	}
	dynamoClient := dynamodb.NewFromConfig(awscfg)
	s3Client := s3.NewFromConfig(awscfg)

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

	var s3Records []LoanCashFlowRecord

	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]

		if len(row) == 0 {
			continue
		}

		item := map[string]types.AttributeValue{}
		hasData := false

		var postdate, maxHmy, loancode string

		for colIdx, cellValue := range row {
			if colIdx >= len(headers) || headers[colIdx] == "" {
				continue
			}

			// Apply Camden to Project-A renaming transformation
			cellValue = renameCamdenToProject(cellValue)

			attrValue := dynamoutils.ParseValue(cellValue)

			if _, isNull := attrValue.(*types.AttributeValueMemberNULL); isNull {
				continue
			}

			switch colIdx {
			case postdateColIdx:
				if sv, ok := attrValue.(*types.AttributeValueMemberS); ok {
					dateStr := sv.Value
					layout := "1/2/2006 3:04:05 PM"
					t, err := time.Parse(layout, dateStr)
					if err == nil {
						postdate = t.Format("2006-01-02T15:04:05")
					} else {
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

		if postdateColIdx != -1 {
			item["postdate"] = &types.AttributeValueMemberS{Value: postdate}
		}
		if maxHmyColIdx != -1 {
			item["maxHmy"] = &types.AttributeValueMemberS{Value: maxHmy}
		}
		if loancodeColIdx != -1 {
			item["loancode"] = &types.AttributeValueMemberS{Value: loancode}
		}

		if postdate != "" && maxHmy != "" {
			compositeKey := fmt.Sprintf("%s#%s", postdate, maxHmy)
			item["postdate#maxHmy"] = &types.AttributeValueMemberS{Value: compositeKey}

			shardId := dynamoutils.GenerateRandomShardId(10)
			item["loancode#shardId"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#%d", loancode, shardId)}
		} else {
			log.Printf("Skipping row %d: missing required postdate or maxHmy values", rowIdx)
			continue
		}

		err := dynamoutils.InsertItemWithRetry(ctx, dynamoClient, tableName, item, 3)

		if err != nil {
			log.Printf("DynamoDB insert failed for row %d: %v", rowIdx, err)
		} else {
			log.Printf("✅ Inserted row %d into DynamoDB", rowIdx)
		}

		itemForS3 := make(map[string]interface{})

		for colIdx, header := range headers {
			if header == "" {
				continue
			}

			var cellValue string
			if colIdx < len(row) {
				cellValue = row[colIdx]
			}

			// Apply Camden to Project-A renaming transformation
			cellValue = renameCamdenToProject(cellValue)

			attrValue := dynamoutils.ParseValue(cellValue)
			itemForS3[header] = convertDynamoAttributeToInterface(attrValue)
		}

		for key, val := range item {
			if _, exists := itemForS3[key]; exists {
				continue
			}

			itemForS3[key] = convertDynamoAttributeToInterface(val)
		}

		s3Record, err := ConvertDynamoItemToS3Record(itemForS3)
		if err != nil {
			log.Printf("Failed to convert row %d to S3 record: %v", rowIdx, err)
		} else {
			s3Records = append(s3Records, s3Record)
		}
	}

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
	rand.Seed(time.Now().UnixNano())
	lambda.Start(Handler)
}
