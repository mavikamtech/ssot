package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Writer struct {
	s3Client   *s3.Client
	bucketName string
	basePath   string
}

func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func NewS3Writer(s3Client *s3.Client, bucketName, basePath string) *S3Writer {
	return &S3Writer{
		s3Client:   s3Client,
		bucketName: bucketName,
		basePath:   basePath,
	}
}

func (w *S3Writer) UploadBatch(ctx context.Context, records []LoanCashFlowRecord, sourceFile string) error {
	if len(records) == 0 {
		return fmt.Errorf("no records to upload")
	}

	now := time.Now().UTC()
	year := now.Year()
	month := int(now.Month())
	day := now.Day()

	dateKey := fmt.Sprintf("year=%d/month=%02d/date=%d",
		year,
		month,
		day,
	)

	err := w.uploadDatePartition(ctx, dateKey, records, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to upload file %s: %w", sourceFile, err)
	}

	return nil
}

func (w *S3Writer) uploadDatePartition(ctx context.Context, dateKey string, records []LoanCashFlowRecord, sourceFile string) error {
	batch := BatchUploadPayload{
		Records: records,
	}

	batch.Metadata.BatchID = generateUUID()
	batch.Metadata.ProcessedAt = time.Now().UTC()
	batch.Metadata.SourceFile = sourceFile
	batch.Metadata.RecordCount = len(records)
	batch.Metadata.ETLVersion = "1.0.0"

	jsonData, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	compressedData, err := w.compressData(jsonData)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	timestamp := time.Now().UTC().Format("2006-01-02_150405")

	sourceFileName := sourceFile
	if strings.HasSuffix(sourceFile, ".xlsx") {
		sourceFileName = strings.TrimSuffix(sourceFile, ".xlsx")
	}

	s3Key := fmt.Sprintf("%s/%s/%s_%s.json.gz",
		w.basePath,
		dateKey,
		timestamp,
		sourceFileName,
	)

	_, err = w.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:             aws.String(w.bucketName),
		Key:                aws.String(s3Key),
		Body:               bytes.NewReader(compressedData),
		ContentType:        aws.String("application/gzip"),
		ContentEncoding:    aws.String("gzip"),
		ContentDisposition: aws.String(fmt.Sprintf("attachment; filename=\"%s_%s.json.gz\"", timestamp, sourceFileName)),
		Metadata: map[string]string{
			"source-file":  sourceFile,
			"record-count": fmt.Sprintf("%d", len(records)),
			"etl-version":  "1.0.0",
			"processed-at": batch.Metadata.ProcessedAt.Format(time.RFC3339),
			"date-key":     dateKey,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	log.Printf("✅ Uploaded %d records to S3: %s", len(records), s3Key)
	return nil
}

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

func ConvertDynamoItemToS3Record(item map[string]interface{}) (LoanCashFlowRecord, error) {
	record := LoanCashFlowRecord{}

	if loanCode, ok := item["loancode"].(string); ok {
		record.LoanCode = loanCode
	}

	if postDateStr, ok := item["postdate"].(string); ok {
		postDate, err := time.Parse("1/2/2006 3:04:05 PM", postDateStr)
		if err != nil {
			return record, fmt.Errorf("failed to parse postdate: %w", err)
		}
		record.PostDate = postDate
	}

	if maxHmyStr, ok := item["maxHmy"].(string); ok {
		if maxHmy, err := parseFloat64(maxHmyStr); err == nil {
			record.MaxHmy = int(maxHmy)
		}
	}

	record.ID = GenerateRecordID(record.LoanCode, record.PostDate, record.MaxHmy)

	record.ETL.ProcessedAt = time.Now().UTC()
	record.ETL.Version = "1.0.0"

	record.Data = make(map[string]interface{})

	for key, value := range item {
		record.Data[key] = value
	}
	return record, nil
}

func parseFloat64(s string) (float64, error) {
	cleanStr := strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	return strconv.ParseFloat(cleanStr, 64)
}

func convertDynamoAttributeToInterface(val interface{}) interface{} {
	switch v := val.(type) {
	case *types.AttributeValueMemberS:
		return v.Value
	case *types.AttributeValueMemberN:
		return v.Value
	case *types.AttributeValueMemberBOOL:
		return v.Value
	case *types.AttributeValueMemberNULL:
		return nil
	default:
		return fmt.Sprintf("%v", v)
	}
}

type QueryHelper struct {
	s3Client   *s3.Client
	bucketName string
	basePath   string
}

func NewQueryHelper(s3Client *s3.Client, bucketName, basePath string) *QueryHelper {
	return &QueryHelper{
		s3Client:   s3Client,
		bucketName: bucketName,
		basePath:   basePath,
	}
}

func (q *QueryHelper) ListPartitions(ctx context.Context, filters map[string]string) ([]string, error) {
	var prefixes []string

	prefix := q.basePath + "/"

	if year, ok := filters["year"]; ok {
		prefix += fmt.Sprintf("year=%s/", year)
		if month, ok := filters["month"]; ok {
			prefix += fmt.Sprintf("month=%s/", month)
			if date, ok := filters["date"]; ok {
				prefix += fmt.Sprintf("date=%s/", date)
			}
		}
	}

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
