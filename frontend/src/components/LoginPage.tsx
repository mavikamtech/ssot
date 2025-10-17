'use client';

export default function LoginPage() {
  return (
    <div className="login-container">
      <div className="login-card">
        <h2>SSOT Application</h2>
        <div style={{ textAlign: 'center', padding: '2rem 0' }}>
          <div style={{ 
            fontSize: '3rem', 
            marginBottom: '1rem',
            color: '#1976d2' 
          }}>
            🔐
          </div>
          <h3 style={{ color: '#333', marginBottom: '1rem' }}>
            OIDC Authentication Required
          </h3>
          <p style={{ color: '#666', lineHeight: '1.6' }}>
            This application requires OIDC authentication through AWS Application Load Balancer.
          </p>
          <p style={{ color: '#666', lineHeight: '1.6' }}>
            Please ensure you are accessing this application through the configured ALB endpoint with OIDC authentication enabled.
          </p>
        </div>
      </div>
    </div>
  );
}