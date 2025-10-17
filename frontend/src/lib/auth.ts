export interface User {
  id: string;
  email: string;
  role: string;
  scope?: string;
  clientId: string;
}

export interface Claims {
  iss?: string;
  sub?: string;
  email?: string;
  exp?: number;
  [key: string]: any;
}

function decodeBase64Url(base64Url: string): string {
  // Replace non-url compatible chars with base64 standard chars
  let base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');

  // Pad out with standard base64 required padding characters
  const pad = base64.length % 4;
  if (pad) {
    if (pad === 1) {
      throw new Error('InvalidLengthError: Input base64url string is the wrong length to determine padding');
    }
    base64 += new Array(5 - pad).join('=');
  }

  if (typeof Buffer !== 'undefined') {
    // Node.js environment
    return Buffer.from(base64, 'base64').toString('utf8');
  } else {
    // Browser environment
    return decodeURIComponent(
      atob(base64)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    );
  }
}

/**
 * Validates OIDC authentication from x-amzn-oidc-data header
 * Based on the Go ValidateOIDCAuth function
 */
export function validateOIDCAuth(oidcData: string): User {
  if (!oidcData) {
    throw new Error('No OIDC data found');
  }

  // Split JWT token into parts
  const parts = oidcData.split('.');
  if (parts.length !== 3) {
    throw new Error('Invalid JWT format');
  }

  // Decode the payload (second part) manually
  let payload: string;
  try {
    payload = decodeBase64Url(parts[1]);
  } catch (err) {
    throw new Error(`Failed to decode payload: ${(err as Error).message}`);
  }

  // Parse the payload JSON
  let claims: Claims;
  try {
    claims = JSON.parse(payload);
  } catch (err) {
    throw new Error(`Failed to parse claims: ${(err as Error).message}`);
  }

  // Check issuer to determine if this is a Microsoft token
  const issuer = claims.iss;
  if (!issuer) {
    throw new Error('Issuer not found in token');
  }

  // For Microsoft Entra ID tokens
  if (issuer.includes('login.microsoftonline.com')) {
    return validateMicrosoftToken(claims);
  }

  // For other tokens (like AWS ALB), validate issuer
  if (!issuer.includes('amazonaws.com') && !issuer.includes('microsoftonline.com')) {
    throw new Error(`Invalid issuer: ${issuer}`);
  }

  return createUserFromClaims(claims);
}

/**
 * Validates a Microsoft Entra ID token (without signature verification)
 */
function validateMicrosoftToken(claims: Claims): User {
  // Check if email field exists and is not empty
  if (!claims.email || claims.email === '') {
    throw new Error('Email not found in token');
  }

  // Validate token expiration
  if (claims.exp && Date.now() / 1000 > claims.exp) {
    throw new Error('Token has expired');
  }

  // Create user with the required scope
  return {
    id: `oidc-${claims.sub}`,
    email: claims.email,
    role: 'user',
    clientId: 'oidc-client',
  };
}

/**
 * Creates a user from validated JWT claims
 */
function createUserFromClaims(claims: Claims): User {
  // Check if email field exists and is not empty
  if (!claims.email || claims.email === '') {
    throw new Error('Email not found in OIDC data');
  }

  // Validate token expiration
  if (claims.exp && Date.now() / 1000 > claims.exp) {
    throw new Error('OIDC token has expired');
  }

  // Create user with the required scope
  return {
    id: `oidc-${claims.sub}`,
    email: claims.email,
    role: 'user',
    scope: 'ssot:gql:loancashflow:read',
    clientId: 'oidc-client',
  };
}

/**
 * Gets OIDC data from request headers (server-side)
 * In production, this would be provided by ALB
 */
export function getOIDCDataFromHeaders(request: Request): string | null {
  if (typeof window !== 'undefined') return null;
  
  // In production ALB environment, this header would be automatically provided
  return request.headers.get('x-amzn-oidc-data');
}