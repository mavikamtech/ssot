import { NextRequest, NextResponse } from 'next/server';

/**
 * POST /api/auth/logout
 * Handles logout by clearing ALB authentication cookies and redirecting to ALB logout endpoint
 */
export async function POST(request: NextRequest) {
  try {
    // Create response that will redirect to ALB logout endpoint
    const response = NextResponse.redirect(new URL('/oauth2/logout', request.url));
    
    // The ALB uses sharded cookies (up to 4 shards for 16K total size)
    // We need to clear all potential authentication cookie shards
    const cookieNames = [
      'AWSELBAuthSessionCookie-0',
      'AWSELBAuthSessionCookie-1', 
      'AWSELBAuthSessionCookie-2',
      'AWSELBAuthSessionCookie-3',
    ];

    // Set expiration time to -1 (past date) for all authentication cookies
    // This follows AWS ALB logout requirements
    cookieNames.forEach(cookieName => {
      response.cookies.set(cookieName, '', {
        path: '/',
        expires: new Date(-1), // Set expiry to past date to delete cookie
        httpOnly: true,
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'lax',
      });
    });

    return response;
  } catch (error) {
    console.error('Logout error:', error);
    return NextResponse.json(
      { error: 'Logout failed' },
      { status: 500 }
    );
  }
}

/**
 * GET /api/auth/logout  
 * Alternative endpoint for GET requests (redirects to POST)
 */
export async function GET(request: NextRequest) {
  // For GET requests, we'll also handle logout directly
  return POST(request);
}
