package main

import (
	"fmt"
	"time"
)

// LoanCashFlowRecord represents a single loan cash flow record optimized for S3 storage and API querying
type LoanCashFlowRecord struct {
	// Primary identifiers for indexing and searching
	ID       string    `json:"id"`       // Unique identifier: loancode#postdate#maxHmy
	LoanCode string    `json:"loanCode"` // Loan identifier
	PostDate time.Time `json:"postDate"` // Transaction post date (ISO 8601)
	MaxHmy   int       `json:"maxHmy"`   // Maximum HMY value

	// Metadata for ETL tracking and versioning
	ETL struct {
		ProcessedAt time.Time `json:"processedAt"` // When this record was processed
		Version     string    `json:"version"`     // ETL version for schema evolution
		ShardID     int       `json:"shardId"`     // For DynamoDB compatibility
	} `json:"etl"`

	// All data fields from source (Excel, API, etc.) - except key fields
	Data map[string]interface{} `json:"data"`
}

// S3StorageStructure defines how data should be organized in S3 for optimal querying
type S3StorageStructure struct {
	// Hierarchical partitioning scheme based on processing time
	// s3://bucket/loan-cashflow/year=2024/month=09/date=20/processed-file.json

	BucketName string `json:"bucketName"`
	BasePath   string `json:"basePath"` // e.g., "loan-cashflow"

	// Partitioning strategy based on processing time (not data time)
	PartitionBy []string `json:"partitionBy"` // ["year", "month", "date"]

	// File organization - one Excel file = one S3 file
	FileStrategy struct {
		MaxRecordsPerFile int    `json:"maxRecordsPerFile"` // e.g., all records from one Excel file
		CompressionType   string `json:"compressionType"`   // "gzip" recommended
		FileFormat        string `json:"fileFormat"`        // "json" or "jsonl" (JSON Lines)
	} `json:"fileStrategy"`
}

// BatchUploadPayload represents a batch of records to upload to S3
type BatchUploadPayload struct {
	Metadata struct {
		BatchID     string    `json:"batchId"`
		ProcessedAt time.Time `json:"processedAt"`
		SourceFile  string    `json:"sourceFile"`
		RecordCount int       `json:"recordCount"`
		ETLVersion  string    `json:"etlVersion"`
		Checksum    string    `json:"checksum"` // For data integrity verification
	} `json:"metadata"`

	Records []LoanCashFlowRecord `json:"records"`
}

// S3 Key Generation Functions
func GenerateS3Key(record LoanCashFlowRecord, basePath string) string {
	// Use current processing time for partitioning (not the data's postdate)
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
		"processed-data", // Generic filename since we process entire files
	)
}

func GenerateRecordID(loanCode string, postDate time.Time, maxHmy int) string {
	return fmt.Sprintf("%s#%s#%d",
		loanCode,
		postDate.Format("2006-01-02T15:04:05"),
		maxHmy,
	)
}
