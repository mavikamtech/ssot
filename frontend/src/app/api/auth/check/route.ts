import { NextRequest, NextResponse } from 'next/server';
import { validateOIDCAuth } from '../../../../lib/auth';

export async function GET(request: NextRequest) {
  try {
    const oidcData = request.headers.get('x-amzn-oidc-data');

    if (!oidcData) {
      return NextResponse.json(
        { 
          authenticated: false, 
          message: 'No OIDC authentication found. This application requires ALB with OIDC authentication.' 
        },
        { status: 401 }
      );
    }

    try {
      const user = validateOIDCAuth(oidcData);
      
      return NextResponse.json({
        authenticated: true,
        user: user,
        token: oidcData
      });
    } catch (validationError) {
      return NextResponse.json(
        { 
          authenticated: false, 
          message: `OIDC token validation failed: ${validationError}` 
        },
        { status: 401 }
      );
    }

  } catch (error) {
    return NextResponse.json(
      { 
        authenticated: false, 
        message: 'Authentication check failed' 
      },
      { status: 500 }
    );
  }
}