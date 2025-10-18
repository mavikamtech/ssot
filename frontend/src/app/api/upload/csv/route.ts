import { NextRequest, NextResponse } from 'next/server';
import { validateOIDCAuth } from '../../../../lib/auth';
import { 
  parseCSV, 
  generateUUIDs, 
  calculateSHA256, 
  createS3Key, 
  createDynamoRecord, 
  saveToDynamoDB, 
  mockS3Upload,
  CSVUploadData 
} from '../../../../lib/csvUpload';

export async function POST(request: NextRequest) {
  const startTime = Date.now();
  const requestId = crypto.randomUUID();
  
  try {
    // Validate user authentication
    const oidcData = request.headers.get('x-amzn-oidc-data');
    if (!oidcData) {
      return NextResponse.json(
        { error: 'Authentication required' },
        { status: 401 }
      );
    }

    let user;
    try {
      user = validateOIDCAuth(oidcData);
    } catch (validationError) {
      return NextResponse.json(
        { error: 'Invalid authentication' },
        { status: 401 }
      );
    }

    // Parse form data to get the uploaded file
    const formData = await request.formData();
    const file = formData.get('file') as File;
    const loanCode = formData.get('loanCode') as string;
    const monthEnd = formData.get('monthEnd') as string;
    const versionNote = formData.get('versionNote') as string;

    if (!file) {
      return NextResponse.json(
        { error: 'No file uploaded' },
        { status: 400 }
      );
    }

    if (!loanCode || !monthEnd) {
      return NextResponse.json(
        { error: 'Loan code and month end are required' },
        { status: 400 }
      );
    }

    // Validate file type
    if (!file.name.toLowerCase().endsWith('.csv')) {
      return NextResponse.json(
        { error: 'Only CSV files are allowed' },
        { status: 400 }
      );
    }

    // Read file content
    const fileContent = await file.text();
    
    if (!fileContent || fileContent.trim().length === 0) {
      return NextResponse.json(
        { error: 'File is empty' },
        { status: 400 }
      );
    }

    // Parse CSV data
    let csvData: CSVUploadData[];
    try {
      csvData = parseCSV(fileContent);

      // Validate that all loanCode and monthEnd values in CSV match the form parameters
      const mismatchedRows = csvData.filter(
        row =>
          row.loanCode !== loanCode ||
          row.monthEnd !== monthEnd
      );
      if (mismatchedRows.length > 0) {
        return NextResponse.json(
          { 
            error: 'CSV contains loanCode or monthEnd values that do not match the form parameters',
            mismatchedRows: mismatchedRows.map((row, idx) => ({ row: idx + 1, loanCode: row.loanCode, monthEnd: row.monthEnd }))
          },
          { status: 400 }
        );
      }
      
      // Set the loan code, month end, and created by for all rows
      csvData = csvData.map(row => ({
        ...row,
        loanCode: loanCode,
        monthEnd: monthEnd,
        createdBy: user.email,
        versionNote: versionNote || row.versionNote || 'CSV upload'
      }));
    } catch (parseError: any) {
      return NextResponse.json(
        { error: `CSV parsing failed: ${parseError.message}` },
        { status: 400 }
      );
    }

    if (csvData.length === 0) {
      return NextResponse.json(
        { error: 'No valid data rows found in CSV' },
        { status: 400 }
      );
    }

    // Generate IDs and calculate file hash
    const { fileId, processId } = generateUUIDs();
    const contentSha256 = calculateSHA256(fileContent);
    const s3Key = createS3Key(loanCode, monthEnd, user.email);

    // Mock S3 upload (get etag)
    const etag = await mockS3Upload(s3Key, fileContent);

    // Save each row to DynamoDB
    const savedRecords = [];
    const errors = [];

    for (let i = 0; i < csvData.length; i++) {
      try {
        const record = createDynamoRecord(
          csvData[i],
          fileId,
          processId,
          s3Key,
          etag,
          contentSha256,
          user.email
        );
        
        await saveToDynamoDB(record);
        savedRecords.push({
          loanCode: csvData[i].loanCode,
          monthEnd: csvData[i].monthEnd,
          cashflowBasedOnMonthEnd: csvData[i].cashflowBasedOnMonthEnd
        });
      } catch (saveError: any) {
        errors.push({
          row: i + 1,
          data: csvData[i],
          error: saveError.message
        });
      }
    }

    const processingTime = Date.now() - startTime;

    // Return results
    return NextResponse.json({
      success: true,
      message: `Successfully uploaded CSV with ${savedRecords.length} records`,
      data: {
        fileId,
        processId,
        recordsProcessed: csvData.length,
        recordsSaved: savedRecords.length,
        recordsWithErrors: errors.length,
        s3Key,
        contentSha256,
        processingTime: `${processingTime}ms`
      },
      savedRecords,
      errors: errors.length > 0 ? errors : undefined
    });

  } catch (error: any) {
    console.error('CSV upload error:', error);
    
    const processingTime = Date.now() - startTime;
    
    return NextResponse.json(
      { 
        error: 'Internal server error during CSV upload',
        details: error.message,
        processingTime: `${processingTime}ms`,
        requestId
      },
      { status: 500 }
    );
  }
}