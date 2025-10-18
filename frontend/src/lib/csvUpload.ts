import { DynamoDBClient, PutItemCommand } from '@aws-sdk/client-dynamodb';
import { createHash } from 'crypto';
import { v4 as uuidv4 } from 'uuid';

const REGION = process.env.AWS_REGION ?? "us-east-1";
const DYNAMODB_TABLE = process.env.DYNAMODB_TABLE || 'ssot-loan-cashflow';
const S3_BUCKET = process.env.S3_BUCKET || 'mavik-ssot-forecasts';

// Create AWS client
const dynamoDbClient = new DynamoDBClient({ region: REGION });

export interface CSVUploadData {
  loanCode: string;
  monthEnd: string;
  cashflowBasedonmonthend: number;
  createdBy: string;
  versionNote?: string;
}

export function generateUUIDs() {
  return {
    fileId: 'f-' + uuidv4(),
    processId: 'p-' + uuidv4()
  };
}

export function calculateSHA256(content: string): string {
  return 'sha256:' + createHash('sha256').update(content).digest('hex');
}

export function parseCSV(content: string): CSVUploadData[] {
  const lines = content.trim().split('\n');
  if (lines.length < 2) {
    throw new Error('CSV must have header and at least one data row');
  }

  const headers = lines[0].split(',').map(h => h.trim().toLowerCase());
  const requiredHeaders = ['loancode', 'monthend', 'cashflowbasedonmonthend'];
  
  for (const required of requiredHeaders) {
    if (!headers.includes(required)) {
      throw new Error(`Missing required column: ${required}`);
    }
  }

  const data: CSVUploadData[] = [];
  for (let i = 1; i < lines.length; i++) {
    const values = lines[i].split(',').map(v => v.trim());
    if (values.length !== headers.length) {
      throw new Error(`Row ${i + 1} has incorrect number of columns`);
    }

    const row: any = {};
    headers.forEach((header, index) => {
      row[header] = values[index];
    });

    if (!row.loancode || !row.monthend || !row.cashflowbasedonmonthend) {
      throw new Error(`Row ${i + 1} is missing required data`);
    }

    const cashflow = parseFloat(row.cashflowbasedonmonthend);
    if (isNaN(cashflow)) {
      throw new Error(`Row ${i + 1}: cashflowBasedonmonthend must be a valid number`);
    }

    data.push({
      loanCode: row.loancode,
      monthEnd: row.monthend,
      cashflowBasedonmonthend: cashflow,
      createdBy: '', // Will be set from auth
      versionNote: row.versionnote || ''
    });
  }

  return data;
}

export function createS3Key(loanCode: string, monthEnd: string, email: string): string {
  const date = new Date(monthEnd);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').replace('T', '_').slice(0, -5);
  
  return `forecasts/loancashflow/${year}/${month}/${day}/${timestamp}_${email}.csv`;
}

export function parseMonthEndToEpoch(monthEnd: string): number {
  // Parse various date formats to epoch seconds
  const date = new Date(monthEnd);
  if (isNaN(date.getTime())) {
    throw new Error(`Invalid month end date format: ${monthEnd}`);
  }
  return Math.floor(date.getTime() / 1000);
}

export function createDynamoRecord(
  data: CSVUploadData,
  fileId: string,
  processId: string,
  s3Key: string,
  etag: string,
  contentSha256: string,
  createdBy: string
): Record<string, any> {
  const ts = Date.now();
  const monthEndEpoch = parseMonthEndToEpoch(data.monthEnd);
  const ttlEpoch = monthEndEpoch + (6 * 30 * 24 * 60 * 60); // 6 months later

  const pk = `LOAN#${data.loanCode}#ME#${data.monthEnd.replace(/\//g, '')}`;
  const sk = `TS#${ts}#FILE#${fileId}`;

  return {
    PK: { S: pk },
    SK: { S: sk },
    file_id: { S: fileId },
    process_id: { S: processId },
    ts: { N: ts.toString() },
    loan_code: { S: data.loanCode },
    month_end_iso: { S: data.monthEnd },
    month_end_epoch: { N: monthEndEpoch.toString() },
    created_by: { S: createdBy },
    version_note: { S: data.versionNote || 'CSV upload' },
    is_current: { BOOL: true },
    data: {
      M: {
        cashflowBasedonmonthend: { N: data.cashflowBasedonmonthend.toString() }
      }
    },
    comments: { L: [
      {
        M: {
          author: { S: createdBy },
          ts_ms: { N: ts.toString() },
          text: { S: 'CSV file uploaded' }
        }
      }
    ]},
    s3_bucket: { S: S3_BUCKET },
    s3_key: { S: s3Key },
    etag: { S: etag },
    content_sha256: { S: contentSha256 },
    ttl_epoch_sec: { N: ttlEpoch.toString() },
    GSI1PK: { S: `USER#${createdBy}` },
    GSI1SK: { S: sk },
    GSI2PK: { S: `ME#${data.monthEnd.replace(/\//g, '')}` },
    GSI2SK: { S: `LOAN#${data.loanCode}#TS#${ts}` }
  };
}

export async function saveToDynamoDB(record: Record<string, any>): Promise<void> {
  const command = new PutItemCommand({
    TableName: DYNAMODB_TABLE,
    Item: record
  });

  try {
    await dynamoDbClient.send(command);
  } catch (error) {
    throw new Error(`Failed to save to DynamoDB: ${error}`);
  }
}

// For now, we'll save S3 info in DynamoDB but not actually upload to S3
// This can be extended later when S3 configuration is ready
export async function mockS3Upload(s3Key: string, content: string): Promise<string> {
  // Generate a mock etag for DynamoDB record
  const etag = '"' + createHash('md5').update(content).digest('hex') + '"';
  return etag;
}