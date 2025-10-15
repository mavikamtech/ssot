import { NextResponse } from 'next/server';

export async function GET() {
  try {
    // Basic health check - verify the service is running
    const healthStatus = {
      status: "ok",
      timestamp: new Date().toISOString(),
      service: "ssot-frontend",
      version: "0.1.0", // Hard-coded version to avoid module resolution issues
      environment: process.env.NODE_ENV || "development",
      uptime: process.uptime()
    };

    return NextResponse.json(healthStatus, {
      status: 200,
      headers: {
        'Content-Type': 'application/json',
        'Cache-Control': 'no-cache'
      }
    });
  } catch (error) {
    console.error('Health check failed:', error);
    
    return NextResponse.json(
      { 
        status: "error",
        timestamp: new Date().toISOString(),
        service: "ssot-frontend",
        error: "Health check failed"
      },
      { 
        status: 503,
        headers: {
          'Content-Type': 'application/json'
        }
      }
    );
  }
}

// Support HEAD requests for load balancer health checks
export async function HEAD() {
  return new NextResponse(null, { status: 200 });
}