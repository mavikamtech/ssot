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