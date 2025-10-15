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

        <div style={{ 
          marginTop: '2rem', 
          padding: '1rem', 
          backgroundColor: '#f8f9fa', 
          borderRadius: '4px',
          fontSize: '0.9rem',
          color: '#666'
        }}>
          <p><strong>For Production Deployment:</strong></p>
          <ul style={{ textAlign: 'left', paddingLeft: '1.5rem', margin: '0.5rem 0' }}>
            <li>Configure ALB with OIDC authentication</li>
            <li>Set up Microsoft Entra ID or other OIDC provider</li>
            <li>Access through ALB endpoint (not direct ECS)</li>
          </ul>
        </div>
      </div>
    </div>
  );
}