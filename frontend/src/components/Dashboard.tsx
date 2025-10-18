'use client';

import { useState } from 'react';
import { useLazyQuery, gql } from '@apollo/client';
import { useAuth } from '../contexts/AuthContext';
import { useAccessLogger } from '../hooks/useAccessLogger';

const LOAN_CASHFLOW_QUERY = gql`
  query GetLoanCashFlow($loanCode: [String!]!, $endDate: String!) {
    loanCashFlow {
      byLoanCode(loanCode: $loanCode, endDate: $endDate) {
        loanCode
        maxHmy
        accrualEndDate
        accrualStartDate
        balance
        capitalizedFee
        capitalizedInterest
        capitalizedLoanAdministrationFee
        capitalizedOtherFees
        commitment
        drawActualPrincipal
        eBalance
        glPeriodDate
        interest
        leverageActivity
        leverageBalance
        leverageInterest
        loanDesc
        postDate
        propertyCode
        propertyName
        sBalance
        status
      }
    }
  }
`;

interface LoanCashFlowData {
  loanCode: string;
  maxHmy: number;
  accrualEndDate: string;
  accrualStartDate: string;
  balance: number;
  capitalizedFee: number;
  capitalizedInterest: number;
  capitalizedLoanAdministrationFee: number;
  capitalizedOtherFees: number;
  commitment: number;
  drawActualPrincipal: number;
  eBalance: number;
  glPeriodDate: string;
  interest: number;
  leverageActivity: number;
  leverageBalance: number;
  leverageInterest: number;
  loanDesc: string;
  postDate: string;
  propertyCode: string;
  propertyName: string;
  sBalance: number;
  status: string;
}

// Function to sanitize CSV values to prevent formula injection
const sanitizeCSVValue = (value: string | number | null | undefined): string => {
  if (value === null || value === undefined) return '';
  
  const stringValue = String(value);
  
  // Check if the value starts with potentially dangerous characters
  if (stringValue.match(/^[=+\-@]/)) {
    // Prefix with single quote to prevent formula interpretation
    return `"'${stringValue.replace(/"/g, '""')}"`;
  }
  
  // For string values, wrap in quotes and escape existing quotes
  if (typeof value === 'string') {
    return `"${stringValue.replace(/"/g, '""')}"`;
  }
  
  // For numeric values, return as-is
  return stringValue;
};

export default function Dashboard() {
  const { user, logout } = useAuth();
  const { logAccess } = useAccessLogger();
  const [endDate, setEndDate] = useState("");
  
  // CSV Upload states
  const [csvFile, setCsvFile] = useState<File | null>(null);
  const [uploadLoanCode, setUploadLoanCode] = useState("");
  const [uploadMonthEnd, setUploadMonthEnd] = useState("");
  const [uploadVersionNote, setUploadVersionNote] = useState("");
  const [isUploading, setIsUploading] = useState(false);
  interface UploadResultData {
    fileId: string;
    processId: string;
    recordsProcessed: number;
    recordsSaved: number;
    recordsWithErrors: number;
    s3Key: string;
    contentSha256: string;
    processingTime: string;
  }
  interface UploadResult {
    message: string;
    data?: UploadResultData;
  }
  const [uploadResult, setUploadResult] = useState<UploadResult | null>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);
  
  const [fetchData, { data, loading, error, called }] = useLazyQuery(LOAN_CASHFLOW_QUERY, {
    notifyOnNetworkStatusChange: true,
  });

  const handleQueryClick = async () => {
    // Log access event
    await logAccess({
      action: 'GET_LOANS',
      route: '/api/graphql',
      method: 'POST',
      description: `Fetching loan cash flow data${endDate ? ` for end date: ${endDate}` : ' (all dates)'}`
    });

    fetchData({
      variables: {
        loanCode: [],
        endDate: endDate,
      },
    });
  };

  const handleCSVUpload = async () => {
    if (!csvFile || !uploadLoanCode || !uploadMonthEnd) {
      setUploadError('Please fill in all required fields and select a CSV file');
      return;
    }

    setIsUploading(true);
    setUploadError(null);
    setUploadResult(null);

    try {
      // Log access event
      await logAccess({
        action: 'UPLOAD_CSV',
        route: '/api/upload/csv',
        method: 'POST',
        description: `Uploading CSV for loan ${uploadLoanCode} - ${uploadMonthEnd}`
      });

      const formData = new FormData();
      formData.append('file', csvFile);
      formData.append('loanCode', uploadLoanCode);
      formData.append('monthEnd', uploadMonthEnd);
      formData.append('versionNote', uploadVersionNote);

      const response = await fetch('/api/upload/csv', {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Upload failed');
      }

      const result = await response.json();
      setUploadResult(result);
      
      // Clear form on success
      setCsvFile(null);
      setUploadLoanCode('');
      setUploadMonthEnd('');
      setUploadVersionNote('');
      
      // Refresh data if we have the same loan loaded
      if (data && called) {
        fetchData({
          variables: {
            loanCode: [],
            endDate: endDate,
          },
        });
      }
      
    } catch (error: any) {
      console.error('CSV upload error:', error);
      setUploadError(error.message);
    } finally {
      setIsUploading(false);
    }
  };

  const downloadCSV = async () => {
    if (!data?.loanCashFlow?.byLoanCode || data.loanCashFlow.byLoanCode.length === 0) {
      alert('No data available to download');
      return;
    }

    // Log access event
    await logAccess({
      action: 'DOWNLOAD_CSV',
      route: '/loans/csv',
      method: 'GET',
      description: `Downloading loan cash flow CSV with ${data.loanCashFlow.byLoanCode.length} records${endDate ? ` for end date: ${endDate}` : ' (all dates)'}`
    });

    // Define CSV headers in the desired order
    const headers = [
      'Property Code',
      'Property Name',
      'Loan Code',
      'Loan Description',
      'Commitment',
      'Status',
      'Accrual Start Date',
      'Accrual End Date',
      'Post Date',
      'GL Period Date',
      'S Balance',
      'Interest',
      'Balance',
      'E Balance',
      'Draw Actual Principal',
      'Capitalized Interest',
      'Capitalized Other Fees',
      'Capitalized Loan Admin Fee',
      'Capitalized Fee',
      'Leverage Balance',
      'Leverage Activity',
      'Leverage Interest',
      'Max HMY'
    ];

    // Sort data the same way as displayed in the table
    const sortedData = data.loanCashFlow.byLoanCode
      .slice()
      .sort((a: LoanCashFlowData, b: LoanCashFlowData) => {
        if (a.loanCode !== b.loanCode) {
          return a.loanCode.localeCompare(b.loanCode);
        }
        if (a.postDate !== b.postDate) {
          return a.postDate.localeCompare(b.postDate);
        }
        return (a.maxHmy || 0) - (b.maxHmy || 0);
      });

    // Convert data to CSV format
    const csvContent = [
      headers.join(','),
      ...sortedData.map((item: LoanCashFlowData) => [
        sanitizeCSVValue(item.propertyCode),
        sanitizeCSVValue(item.propertyName),
        sanitizeCSVValue(item.loanCode),
        sanitizeCSVValue(item.loanDesc),
        item.commitment || '',
        sanitizeCSVValue(item.status),
        sanitizeCSVValue(item.accrualStartDate),
        sanitizeCSVValue(item.accrualEndDate),
        sanitizeCSVValue(item.postDate),
        sanitizeCSVValue(item.glPeriodDate),
        item.sBalance || '',
        item.interest || '',
        item.balance || '',
        item.eBalance || '',
        item.drawActualPrincipal || '',
        item.capitalizedInterest || '',
        item.capitalizedOtherFees || '',
        item.capitalizedLoanAdministrationFee || '',
        item.capitalizedFee || '',
        item.leverageBalance || '',
        item.leverageActivity || '',
        item.leverageInterest || '',
        item.maxHmy || ''
      ].join(','))
    ].join('\n');

    // Create and download the file
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', `loan_cashflow_data_${new Date().toISOString().replace(/[:.]/g, '-').replace('T', '_').slice(0, -5)}_UTC.csv`);
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div>
      <header className="header">
        <h1>SSOT Dashboard</h1>
        <div className="user-info">
          <span>Welcome, {user?.email}</span>
          <button onClick={logout} className="btn btn-secondary">
            Logout
          </button>
        </div>
      </header>

      <div className="container">
        <div style={{ marginBottom: '2rem' }}>
          <h2>Loan Cash Flow Data</h2>
          <p>Click the button below to fetch loan cash flow data from the GraphQL API.</p>
          
          <div style={{ marginBottom: '1rem' }}>
            <label htmlFor="endDate" style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 'bold' }}>
              End Date (optional):
            </label>
            <input
              id="endDate"
              type="text"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              placeholder="e.g., 12/31/2024 or leave empty"
              style={{
                padding: '0.5rem',
                border: '1px solid #ccc',
                borderRadius: '4px',
                width: '200px',
                marginRight: '1rem'
              }}
            />
            <small style={{ color: '#666' }}>
              Leave empty for all dates, or enter in format MM/DD/YYYY
            </small>
          </div>
          
          <button 
            onClick={handleQueryClick}
            className="btn"
            disabled={loading}
          >
            {loading ? 'Loading...' : called ? 'Refresh Data' : 'Fetch Loan Cash Flow Data'}
          </button>
          
          {data && data.loanCashFlow && data.loanCashFlow.byLoanCode && data.loanCashFlow.byLoanCode.length > 0 && (
            <button 
              onClick={downloadCSV}
              className="btn btn-secondary"
              style={{ marginLeft: '1rem' }}
            >
              Download CSV
            </button>
          )}
        </div>

        {error && (
          <div className="error">
            <strong>Error:</strong> {error.message}
            <details style={{ marginTop: '0.5rem' }}>
              <summary style={{ cursor: 'pointer' }}>View Error Details</summary>
              <pre style={{ fontSize: '0.8rem', marginTop: '0.5rem', overflow: 'auto' }}>
                {JSON.stringify({
                  message: error.message,
                  networkError: error.networkError ? {
                    message: error.networkError.message,
                    statusCode: (error.networkError as any)?.statusCode,
                    result: (error.networkError as any)?.result
                  } : null,
                  graphQLErrors: error.graphQLErrors
                }, null, 2)}
              </pre>
            </details>
          </div>
        )}

        {/* CSV Upload Section - Only show when data is successfully loaded */}
        {data && data.loanCashFlow && data.loanCashFlow.byLoanCode && data.loanCashFlow.byLoanCode.length > 0 && (
          <div style={{ 
            marginBottom: '2rem', 
            padding: '1.5rem', 
            background: '#f8f9fa', 
            borderRadius: '8px',
            border: '1px solid #e9ecef'
          }}>
            <h3 style={{ marginTop: 0, marginBottom: '1rem', color: '#495057' }}>
              Upload CSV Data
            </h3>
            <p style={{ marginBottom: '1rem', fontSize: '0.9rem', color: '#6c757d' }}>
              Upload a CSV file with loan cash flow data. Required columns: loancode, monthend, cashflowBasedonmonthend
            </p>
            <details style={{ marginBottom: '1rem' }}>
              <summary style={{ cursor: 'pointer', fontSize: '0.9rem', color: '#007bff' }}>
                View CSV Format Example
              </summary>
              <pre style={{ 
                marginTop: '0.5rem', 
                fontSize: '0.8rem', 
                background: '#fff', 
                padding: '0.5rem', 
                border: '1px solid #dee2e6',
                borderRadius: '4px',
                color: '#495057'
              }}>
{`loancode,monthend,cashflowBasedonmonthend,versionnote
VS1-0001,2024-12-31,150000.50,Q4 forecast
VS1-0002,2024-12-31,275000.00,Updated projection`}
              </pre>
            </details>
            
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem', marginBottom: '1rem' }}>
              <div>
                <label style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 'bold', fontSize: '0.9rem' }}>
                  Loan Code *
                </label>
                <input
                  type="text"
                  value={uploadLoanCode}
                  onChange={(e) => setUploadLoanCode(e.target.value)}
                  placeholder="e.g., VS1-0001"
                  style={{
                    padding: '0.5rem',
                    border: '1px solid #ced4da',
                    borderRadius: '4px',
                    width: '100%',
                    fontSize: '0.9rem'
                  }}
                />
              </div>
              
              <div>
                <label style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 'bold', fontSize: '0.9rem' }}>
                  Month End *
                </label>
                <input
                  type="text"
                  value={uploadMonthEnd}
                  onChange={(e) => setUploadMonthEnd(e.target.value)}
                  placeholder="e.g., 2024-12-31"
                  style={{
                    padding: '0.5rem',
                    border: '1px solid #ced4da',
                    borderRadius: '4px',
                    width: '100%',
                    fontSize: '0.9rem'
                  }}
                />
              </div>
            </div>
            
            <div style={{ marginBottom: '1rem' }}>
              <label style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 'bold', fontSize: '0.9rem' }}>
                Version Note (optional)
              </label>
              <input
                type="text"
                value={uploadVersionNote}
                onChange={(e) => setUploadVersionNote(e.target.value)}
                placeholder="e.g., Q4 forecast update"
                style={{
                  padding: '0.5rem',
                  border: '1px solid #ced4da',
                  borderRadius: '4px',
                  width: '100%',
                  fontSize: '0.9rem'
                }}
              />
            </div>
            
            <div style={{ marginBottom: '1rem' }}>
              <label style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 'bold', fontSize: '0.9rem' }}>
                CSV File *
              </label>
              <input
                type="file"
                accept=".csv"
                onChange={(e) => setCsvFile(e.target.files?.[0] || null)}
                style={{
                  padding: '0.5rem',
                  border: '1px solid #ced4da',
                  borderRadius: '4px',
                  width: '100%',
                  fontSize: '0.9rem'
                }}
              />
              {csvFile && (
                <small style={{ color: '#28a745', fontSize: '0.8rem' }}>
                  Selected: {csvFile.name} ({(csvFile.size / 1024).toFixed(1)} KB)
                </small>
              )}
            </div>
            
            <button
              onClick={handleCSVUpload}
              disabled={isUploading || !csvFile || !uploadLoanCode || !uploadMonthEnd}
              className="btn"
              style={{
                marginRight: '1rem',
                opacity: (isUploading || !csvFile || !uploadLoanCode || !uploadMonthEnd) ? 0.6 : 1
              }}
            >
              {isUploading ? 'Uploading...' : 'Upload CSV'}
            </button>
            
            {uploadResult && (
              <div style={{ 
                marginTop: '1rem', 
                padding: '1rem', 
                background: '#d4edda', 
                border: '1px solid #c3e6cb',
                borderRadius: '4px',
                fontSize: '0.9rem'
              }}>
                <strong style={{ color: '#155724' }}>✓ Upload Successful!</strong>
                <div style={{ marginTop: '0.5rem', color: '#155724' }}>
                  {uploadResult.message}
                </div>
                {uploadResult.data && (
                  <details style={{ marginTop: '0.5rem' }}>
                    <summary style={{ cursor: 'pointer', color: '#155724' }}>View Details</summary>
                    <pre style={{ marginTop: '0.5rem', fontSize: '0.8rem', color: '#155724' }}>
                      {JSON.stringify(uploadResult.data, null, 2)}
                    </pre>
                  </details>
                )}
              </div>
            )}
            
            {uploadError && (
              <div style={{ 
                marginTop: '1rem', 
                padding: '1rem', 
                background: '#f8d7da', 
                border: '1px solid #f5c6cb',
                borderRadius: '4px',
                fontSize: '0.9rem',
                color: '#721c24'
              }}>
                <strong>✗ Upload Failed:</strong> {uploadError}
              </div>
            )}
          </div>
        )}

        {data && data.loanCashFlow && data.loanCashFlow.byLoanCode && (
          <div>
            <h3>Results ({data.loanCashFlow.byLoanCode.length} records)</h3>
            
            {data.loanCashFlow.byLoanCode.length === 0 ? (
              <p>No loan cash flow data found.</p>
            ) : (
              <div style={{ overflowX: 'auto' }}>
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>Property Code</th>
                      <th>Property Name</th>
                      <th>Loan Code</th>
                      <th>Loan Description</th>
                      <th>Commitment</th>
                      <th>Status</th>
                      <th>Accrual Start Date</th>
                      <th>Accrual End Date</th>
                      <th>Post Date</th>
                      <th>GL Period Date</th>
                      <th>S Balance</th>
                      <th>Interest</th>
                      <th>Balance</th>
                      <th>E Balance</th>
                      <th>Draw Actual Principal</th>
                      <th>Capitalized Interest</th>
                      <th>Capitalized Other Fees</th>
                      <th>Capitalized Loan Admin Fee</th>
                      <th>Capitalized Fee</th>
                      <th>Leverage Balance</th>
                      <th>Leverage Activity</th>
                      <th>Leverage Interest</th>
                      <th>Max HMY</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.loanCashFlow.byLoanCode
                      .slice() // Create a copy to avoid mutating the original array
                      .sort((a: LoanCashFlowData, b: LoanCashFlowData) => {
                        // Sort by loanCode first
                        if (a.loanCode !== b.loanCode) {
                          return a.loanCode.localeCompare(b.loanCode);
                        }
                        // Then by postDate
                        if (a.postDate !== b.postDate) {
                          return a.postDate.localeCompare(b.postDate);
                        }
                        // Finally by maxHmy
                        return (a.maxHmy || 0) - (b.maxHmy || 0);
                      })
                      .map((item: LoanCashFlowData, index: number) => (
                      <tr key={`${item.loanCode}-${item.postDate}-${item.maxHmy}`}>
                        <td>{item.propertyCode}</td>
                        <td>{item.propertyName}</td>
                        <td>{item.loanCode}</td>
                        <td>{item.loanDesc}</td>
                        <td>${item.commitment?.toLocaleString() || 'N/A'}</td>
                        <td>{item.status}</td>
                        <td>{item.accrualStartDate}</td>
                        <td>{item.accrualEndDate}</td>
                        <td>{item.postDate}</td>
                        <td>{item.glPeriodDate}</td>
                        <td>${item.sBalance?.toLocaleString() || 'N/A'}</td>
                        <td>${item.interest?.toLocaleString() || 'N/A'}</td>
                        <td>${item.balance?.toLocaleString() || 'N/A'}</td>
                        <td>${item.eBalance?.toLocaleString() || 'N/A'}</td>
                        <td>${item.drawActualPrincipal?.toLocaleString() || 'N/A'}</td>
                        <td>${item.capitalizedInterest?.toLocaleString() || 'N/A'}</td>
                        <td>${item.capitalizedOtherFees?.toLocaleString() || 'N/A'}</td>
                        <td>${item.capitalizedLoanAdministrationFee?.toLocaleString() || 'N/A'}</td>
                        <td>${item.capitalizedFee?.toLocaleString() || 'N/A'}</td>
                        <td>${item.leverageBalance?.toLocaleString() || 'N/A'}</td>
                        <td>{item.leverageActivity?.toLocaleString() || 'N/A'}</td>
                        <td>${item.leverageInterest?.toLocaleString() || 'N/A'}</td>
                        <td>{item.maxHmy || 'N/A'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            <details style={{ marginTop: '2rem', padding: '1rem', background: '#f8f9fa', borderRadius: '4px' }}>
              <summary style={{ cursor: 'pointer', fontWeight: 'bold' }}>
                View All Fields (Raw Data)
              </summary>
              <pre style={{ marginTop: '1rem', fontSize: '0.9rem', overflow: 'auto' }}>
                {JSON.stringify(data.loanCashFlow.byLoanCode, null, 2)}
              </pre>
            </details>
          </div>
        )}
      </div>
    </div>
  );
}