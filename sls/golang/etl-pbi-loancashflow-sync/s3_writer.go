package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Writer handles writing loan cash flow data to S3 with optimal structure
type S3Writer struct {
	s3Client   *s3.Client
	bucketName string
	basePath   string
}

// generateUUID creates a simple UUID-like string
func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// NewS3Writer creates a new S3Writer instance
func NewS3Writer(s3Client *s3.Client, bucketName, basePath string) *S3Writer {
	return &S3Writer{
		s3Client:   s3Client,
		bucketName: bucketName,
		basePath:   basePath,
	}
}

// UploadBatch uploads a batch of records to S3 with proper partitioning
func (w *S3Writer) UploadBatch(ctx context.Context, records []LoanCashFlowRecord, sourceFile string) error {
	if len(records) == 0 {
		return fmt.Errorf("no records to upload")
	}

	// Group records by partition for efficient storage
	partitionGroups := make(map[string][]LoanCashFlowRecord)

	for _, record := range records {
		// Create partition key from postDate
		year := record.PostDate.Year()
		month := int(record.PostDate.Month())
		quarter := ((month - 1) / 3) + 1

		// Create partition key
		partitionKey := fmt.Sprintf("year=%d/month=%02d/quarter=Q%d/loancode=%s",
			year,
			month,
			quarter,
			record.LoanCode,
		)

		partitionGroups[partitionKey] = append(partitionGroups[partitionKey], record)
	}

	// Upload each partition group
	for partitionKey, partitionRecords := range partitionGroups {
		err := w.uploadPartition(ctx, partitionKey, partitionRecords, sourceFile)
		if err != nil {
			return fmt.Errorf("failed to upload partition %s: %w", partitionKey, err)
		}
	}

	return nil
}

// uploadPartition uploads records for a specific partition
func (w *S3Writer) uploadPartition(ctx context.Context, partitionKey string, records []LoanCashFlowRecord, sourceFile string) error {
	// Create batch payload
	batch := BatchUploadPayload{
		Records: records,
	}

	// Set metadata
	batch.Metadata.BatchID = generateUUID()
	batch.Metadata.ProcessedAt = time.Now().UTC()
	batch.Metadata.SourceFile = sourceFile
	batch.Metadata.RecordCount = len(records)
	batch.Metadata.ETLVersion = "1.0.0"

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Calculate checksum
	hash := md5.Sum(jsonData)
	batch.Metadata.Checksum = fmt.Sprintf("%x", hash)

	// Re-serialize with checksum
	jsonData, err = json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON with checksum: %w", err)
	}

	// Compress data
	compressedData, err := w.compressData(jsonData)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	// Generate S3 key
	timestamp := time.Now().UTC().Format("2006-01-02_150405")
	s3Key := fmt.Sprintf("%s/%s/%s_batch_%s.json.gz",
		w.basePath,
		partitionKey,
		timestamp,
		batch.Metadata.BatchID[:8], // First 8 chars of UUID
	)

	// Upload to S3
	_, err = w.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:          aws.String(w.bucketName),
		Key:             aws.String(s3Key),
		Body:            bytes.NewReader(compressedData),
		ContentType:     aws.String("application/json"),
		ContentEncoding: aws.String("gzip"),
		Metadata: map[string]string{
			"source-file":   sourceFile,
			"record-count":  fmt.Sprintf("%d", len(records)),
			"etl-version":   "1.0.0",
			"processed-at":  batch.Metadata.ProcessedAt.Format(time.RFC3339),
			"partition-key": partitionKey,
			"checksum":      batch.Metadata.Checksum,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	log.Printf("✅ Uploaded %d records to S3: %s", len(records), s3Key)
	return nil
}

// compressData compresses JSON data using gzip
func (w *S3Writer) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)

	_, err := gzWriter.Write(data)
	if err != nil {
		return nil, err
	}

	err = gzWriter.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ConvertDynamoItemToS3Record converts a DynamoDB item to an S3 record
func ConvertDynamoItemToS3Record(item map[string]interface{}) (LoanCashFlowRecord, error) {
	record := LoanCashFlowRecord{}

	// Extract core fields
	if loanCode, ok := item["loancode"].(string); ok {
		record.LoanCode = loanCode
	}

	if postDateStr, ok := item["postdate"].(string); ok {
		postDate, err := time.Parse("2006-01-02T15:04:05", postDateStr)
		if err != nil {
			return record, fmt.Errorf("failed to parse postdate: %w", err)
		}
		record.PostDate = postDate
	}

	if maxHmyStr, ok := item["maxHmy"].(string); ok {
		// Convert string to int
		if maxHmy, err := parseFloat64(maxHmyStr); err == nil {
			record.MaxHmy = int(maxHmy)
		}
	}

	// Generate ID
	record.ID = GenerateRecordID(record.LoanCode, record.PostDate, record.MaxHmy)

	// Set ETL metadata
	record.ETL.ProcessedAt = time.Now().UTC()
	record.ETL.Version = "1.0.0"

	// Extract data - ALL other fields go directly into data map
	record.Data = make(map[string]interface{})

	// Process all fields except core identifiers
	for key, value := range item {
		// Skip core identifier fields and composite keys
		if key == "loancode" || key == "postdate" || key == "maxHmy" ||
			strings.Contains(key, "#") { // Skip composite keys
			continue
		}

		// All other fields go directly into data map
		record.Data[key] = value
	}

	// Set search fields - simplified (can be added back later if needed)
	// record.SearchFields.LoanCodeLower = strings.ToLower(record.LoanCode)

	// No need to populate partition fields - they're calculated on demand

	return record, nil
}

// Helper function to parse various numeric types to float64
func parseFloat64ToString(value interface{}) (float64, error) {
	switch v := value.(type) {
	case string:
		return parseFloat64(v)
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func parseFloat64(s string) (float64, error) {
	// Remove commas and parse
	cleanStr := strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	return strconv.ParseFloat(cleanStr, 64)
}

// QueryHelper provides utilities for querying the S3 data structure
type QueryHelper struct {
	s3Client   *s3.Client
	bucketName string
	basePath   string
}

// NewQueryHelper creates a new QueryHelper instance
func NewQueryHelper(s3Client *s3.Client, bucketName, basePath string) *QueryHelper {
	return &QueryHelper{
		s3Client:   s3Client,
		bucketName: bucketName,
		basePath:   basePath,
	}
}

// ListPartitions returns available partitions for querying
func (q *QueryHelper) ListPartitions(ctx context.Context, filters map[string]string) ([]string, error) {
	var prefixes []string

	// Build prefix based on filters
	prefix := q.basePath + "/"

	if year, ok := filters["year"]; ok {
		prefix += fmt.Sprintf("year=%s/", year)
		if month, ok := filters["month"]; ok {
			prefix += fmt.Sprintf("month=%s/", month)
			if quarter, ok := filters["quarter"]; ok {
				prefix += fmt.Sprintf("quarter=%s/", quarter)
				if loanCode, ok := filters["loancode"]; ok {
					prefix += fmt.Sprintf("loancode=%s/", loanCode)
				}
			}
		}
	}

	// List objects with the prefix
	resp, err := q.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(q.bucketName),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	for _, commonPrefix := range resp.CommonPrefixes {
		prefixes = append(prefixes, *commonPrefix.Prefix)
	}

	return prefixes, nil
}

// Example usage for your ETL integration:
/*
func integrateS3Upload(ctx context.Context, s3Client *s3.Client, records []LoanCashFlowRecord) error {
	s3Writer := NewS3Writer(s3Client, "your-s3-bucket", "loan-cashflow")

	return s3Writer.UploadBatch(ctx, records, "YDC-Response-LoanCashFlow-Camden-Only.xlsx")
}
*/
