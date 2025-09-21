package main

import (
	"fmt"
	"time"
)

type LoanCashFlowRecord struct {
	ID       string    `json:"id"`       // Unique identifier: loancode#postdate#maxHmy
	LoanCode string    `json:"loanCode"` // Loan identifier
	PostDate time.Time `json:"postDate"` // Transaction post date
	MaxHmy   int       `json:"maxHmy"`

	// Metadata for ETL tracking and versioning
	ETL struct {
		ProcessedAt time.Time `json:"processedAt"` // When this record was processed
		Version     string    `json:"version"`     // ETL version for schema evolution
		ShardID     int       `json:"shardId"`     // For DynamoDB compatibility
	} `json:"etl"`

	Data map[string]interface{} `json:"data"`
}

type S3StorageStructure struct {
	BucketName  string   `json:"bucketName"`
	BasePath    string   `json:"basePath"`
	PartitionBy []string `json:"partitionBy"` // ["year", "month", "date"]

	FileStrategy struct {
		MaxRecordsPerFile int    `json:"maxRecordsPerFile"`
		CompressionType   string `json:"compressionType"`
		FileFormat        string `json:"fileFormat"`
	} `json:"fileStrategy"`
}

type BatchUploadPayload struct {
	Metadata struct {
		BatchID     string    `json:"batchId"`
		ProcessedAt time.Time `json:"processedAt"`
		SourceFile  string    `json:"sourceFile"`
		RecordCount int       `json:"recordCount"`
		ETLVersion  string    `json:"etlVersion"`
		Checksum    string    `json:"checksum"`
	} `json:"metadata"`

	Records []LoanCashFlowRecord `json:"records"`
}

func GenerateS3Key(record LoanCashFlowRecord, basePath string) string {
	now := time.Now().UTC()
	year := now.Year()
	month := int(now.Month())
	day := now.Day()

	// Example: loan-cashflow/year=2024/month=09/date=20/2024-09-20_143022_filename.json
	return fmt.Sprintf("%s/year=%d/month=%02d/date=%d/%s_%s.json",
		basePath,
		year,
		month,
		day,
		now.Format("2006-01-02_150405"),
		"processed-data",
	)
}

func GenerateRecordID(loanCode string, postDate time.Time, maxHmy int) string {
	return fmt.Sprintf("%s#%s#%d",
		loanCode,
		postDate.Format("2006-01-02T15:04:05"),
		maxHmy,
	)
}
