import { NextRequest, NextResponse } from 'next/server';

/**
 * POST /api/auth/logout
 * Handles logout by clearing ALB authentication cookies and redirecting to ALB logout endpoint
 */
export async function POST(request: NextRequest) {
  const requestId = crypto.randomUUID();
  const timestamp = new Date().toISOString();
  
  console.log(`[LOGOUT] ${timestamp} - Request ID: ${requestId} - Logout flow initiated`, {
    method: 'POST',
    userAgent: request.headers.get('user-agent'),
    ip: request.ip || request.headers.get('x-forwarded-for') || 'unknown',
    referer: request.headers.get('referer')
  });

  try {
    // Get Azure AD configuration from environment variables
    const tenant = process.env.NEXT_PUBLIC_AZURE_AD_TENANT_ID || 'common';
    const postLogoutRedirectUri = process.env.NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI;
    
    console.log(`[LOGOUT] ${requestId} - Configuration loaded`, {
      tenant: tenant,
      hasPostLogoutRedirect: !!postLogoutRedirectUri,
      postLogoutRedirectUri: postLogoutRedirectUri || 'not_configured'
    });
    
    // Build Azure AD logout URL using URLSearchParams for proper encoding
    const baseLogoutUrl = `https://login.microsoftonline.com/${tenant}/oauth2/v2.0/logout`;
    let logoutUrl: string;
    
    if (postLogoutRedirectUri) {
      const url = new URL(baseLogoutUrl);
      url.searchParams.set('post_logout_redirect_uri', postLogoutRedirectUri);
      logoutUrl = url.toString();
    } else {
      // Fallback to basic Azure AD logout without redirect
      logoutUrl = baseLogoutUrl;
    }
    
    console.log(`[LOGOUT] ${requestId} - Azure AD logout URL constructed`, {
      logoutUrl: logoutUrl,
      baseUrl: baseLogoutUrl
    });
    
    // Create response that will redirect to Azure AD logout endpoint
    const response = NextResponse.redirect(logoutUrl);
    
    // The ALB uses sharded cookies (up to 4 shards for 16K total size)
    // We need to clear all potential authentication cookie shards
    const cookieNames = [
      'AWSELBAuthSessionCookie-0',
      'AWSELBAuthSessionCookie-1', 
      'AWSELBAuthSessionCookie-2',
      'AWSELBAuthSessionCookie-3',
    ];

    console.log(`[LOGOUT] ${requestId} - Starting ALB cookie invalidation`, {
      cookieNames: cookieNames,
      cookieCount: cookieNames.length
    });

    // Set expiration time to -1 (past date) for all authentication cookies
    // This follows AWS ALB logout requirements - must set both expires and maxAge to -1
    cookieNames.forEach(cookieName => {
      response.cookies.set(cookieName, '', {
        path: '/',
        expires: new Date(0), // Set expiry to epoch (past date) to delete cookie
        maxAge: -1, // Explicitly set maxAge to -1 for ALB cookie invalidation
        httpOnly: true,
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'lax',
      });
    });

    console.log(`[LOGOUT] ${requestId} - ALB cookies invalidated successfully`, {
      clearedCookies: cookieNames,
      isProduction: process.env.NODE_ENV === 'production',
      secureFlag: process.env.NODE_ENV === 'production'
    });

    console.log(`[LOGOUT] ${requestId} - Logout flow completed successfully`, {
      redirectTo: logoutUrl,
      timestamp: new Date().toISOString(),
      totalCookiesCleared: cookieNames.length
    });

    return response;
  } catch (error) {
    console.error(`[LOGOUT] ${requestId} - Logout error occurred`, {
      error: error instanceof Error ? error.message : 'Unknown error',
      stack: error instanceof Error ? error.stack : undefined,
      timestamp: new Date().toISOString(),
      requestId: requestId
    });
    
    return NextResponse.json(
      { 
        error: 'Logout failed',
        requestId: requestId,
        timestamp: new Date().toISOString()
      },
      { status: 500 }
    );
  }
}

/**
 * GET /api/auth/logout  
 * Alternative endpoint for GET requests (redirects to POST)
 */
export async function GET(request: NextRequest) {
  const requestId = crypto.randomUUID();
  console.log(`[LOGOUT] ${new Date().toISOString()} - Request ID: ${requestId} - GET logout request received, delegating to POST handler`, {
    method: 'GET',
    userAgent: request.headers.get('user-agent'),
    ip: request.ip || request.headers.get('x-forwarded-for') || 'unknown'
  });
  
  // For GET requests, we'll also handle logout directly
  return POST(request);
}
