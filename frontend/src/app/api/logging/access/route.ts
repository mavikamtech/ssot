import { NextRequest, NextResponse } from 'next/server';
import { logAccessIfNeeded, createAccessLogEvent, ensureLogsInfra } from '../../../../lib/cloudwatchLogger';
import { validateOIDCAuth } from '../../../../lib/auth';

// Ensure logging infrastructure exists at application startup
let infrastructureEnsured = false;

async function ensureInfrastructure() {
  if (!infrastructureEnsured) {
    try {
      await ensureLogsInfra();
      infrastructureEnsured = true;
    } catch (error) {
      console.error('Failed to ensure CloudWatch logs infrastructure:', error);
    }
  }
}

export async function POST(request: NextRequest) {
  const startTime = Date.now();
  const requestId = crypto.randomUUID();
  
  try {
    // Ensure logging infrastructure exists
    await ensureInfrastructure();
    
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

    // Parse request body
    const body = await request.json();
    const { action, route, method = 'POST', description } = body;

    // Validate required fields
    if (!action || !route) {
      return NextResponse.json(
        { error: 'Missing required fields: action, route' },
        { status: 400 }
      );
    }

    // Get client information
    const ip = request.ip || 
               request.headers.get('x-forwarded-for') || 
               request.headers.get('x-real-ip') || 
               'unknown';
    const userAgent = request.headers.get('user-agent') || 'unknown';

    // Create access log event
    const logEvent = createAccessLogEvent({
      request_id: requestId,
      email: user.email,
      ip,
      route,
      method,
      status: 200,
      duration_ms: Date.now() - startTime,
      action,
      description,
      user_agent: userAgent
    });

    // Log to CloudWatch (only for target operations)
    await logAccessIfNeeded(logEvent);

    return NextResponse.json({
      success: true,
      logged: logEvent.action === 'GET_LOANS' || 
               logEvent.action === 'DOWNLOAD_CSV' || 
               /\/csv(\?|$)/i.test(logEvent.route),
      request_id: requestId
    });

  } catch (error) {
    console.error(`[ACCESS_LOG] ${requestId} - Error:`, error);
    
    return NextResponse.json(
      { 
        error: 'Failed to log access event',
        request_id: requestId
      },
      { status: 500 }
    );
  }
}