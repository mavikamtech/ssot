'use client';

import { useState, useEffect } from 'react';
import { useQuery, gql } from '@apollo/client';
import { useAuth } from '../contexts/AuthContext';

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

export default function Dashboard() {
  const { user, logout } = useAuth();
  const [queryTriggered, setQueryTriggered] = useState(false);
  const [endDate, setEndDate] = useState("");
  
  const { data, loading, error, refetch } = useQuery(LOAN_CASHFLOW_QUERY, {
    variables: {
      loanCode: [],
      endDate: ""
    },
    skip: !queryTriggered,
    notifyOnNetworkStatusChange: true
  });

  const handleQueryClick = async () => {
    setQueryTriggered(true);
    
    if (queryTriggered) {
      await refetch({
        loanCode: [],
        endDate: endDate
      });
    }
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
            {loading ? 'Loading...' : queryTriggered ? 'Refresh Data' : 'Fetch Loan Cash Flow Data'}
          </button>
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
                      <th>Loan Code</th>
                      <th>Max HMY</th>
                      <th>Post Date</th>
                      <th>Property Name</th>
                      <th>Property Code</th>
                      <th>Status</th>
                      <th>Balance</th>
                      <th>Interest</th>
                      <th>Commitment</th>
                      <th>Accrual Start Date</th>
                      <th>Accrual End Date</th>
                      <th>Capitalized Fee</th>
                      <th>Capitalized Interest</th>
                      <th>Capitalized Loan Admin Fee</th>
                      <th>Capitalized Other Fees</th>
                      <th>Draw Actual Principal</th>
                      <th>E Balance</th>
                      <th>GL Period Date</th>
                      <th>Leverage Activity</th>
                      <th>Leverage Balance</th>
                      <th>Leverage Interest</th>
                      <th>Loan Description</th>
                      <th>S Balance</th>
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
                      <tr key={index}>
                        <td>{item.loanCode}</td>
                        <td>{item.maxHmy || 'N/A'}</td>
                        <td>{item.postDate}</td>
                        <td>{item.propertyName}</td>
                        <td>{item.propertyCode}</td>
                        <td>{item.status}</td>
                        <td>${item.balance?.toLocaleString() || 'N/A'}</td>
                        <td>${item.interest?.toLocaleString() || 'N/A'}</td>
                        <td>${item.commitment?.toLocaleString() || 'N/A'}</td>
                        <td>{item.accrualStartDate}</td>
                        <td>{item.accrualEndDate}</td>
                        <td>${item.capitalizedFee?.toLocaleString() || 'N/A'}</td>
                        <td>${item.capitalizedInterest?.toLocaleString() || 'N/A'}</td>
                        <td>${item.capitalizedLoanAdministrationFee?.toLocaleString() || 'N/A'}</td>
                        <td>${item.capitalizedOtherFees?.toLocaleString() || 'N/A'}</td>
                        <td>${item.drawActualPrincipal?.toLocaleString() || 'N/A'}</td>
                        <td>${item.eBalance?.toLocaleString() || 'N/A'}</td>
                        <td>{item.glPeriodDate}</td>
                        <td>{item.leverageActivity?.toLocaleString() || 'N/A'}</td>
                        <td>${item.leverageBalance?.toLocaleString() || 'N/A'}</td>
                        <td>${item.leverageInterest?.toLocaleString() || 'N/A'}</td>
                        <td>{item.loanDesc}</td>
                        <td>${item.sBalance?.toLocaleString() || 'N/A'}</td>
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