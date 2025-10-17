/**
 * Server startup logger for Next.js application
 * This module logs detailed startup information when the server initializes
 */

class StartupLogger {
  private static instance: StartupLogger;
  private hasLogged = false;

  private constructor() {}

  public static getInstance(): StartupLogger {
    if (!StartupLogger.instance) {
      StartupLogger.instance = new StartupLogger();
    }
    return StartupLogger.instance;
  }

  public logStartup(): void {
    // Only log once and only on server-side
    if (this.hasLogged || typeof window !== 'undefined') {
      return;
    }

    this.hasLogged = true;

    const startupTime = new Date().toISOString();
    
    console.log('====================================');
    console.log('🚀 SSOT Frontend Server Starting');
    console.log('====================================');
    
    console.log('📅 Startup Time:', startupTime);
    console.log('🌍 Environment:', process.env.NODE_ENV || 'development');
    console.log('🔧 Node.js Version:', process.version);
    console.log('💻 Platform:', `${process.platform} (${process.arch})`);
    console.log('🆔 Process ID:', process.pid);
    console.log('⏱️ Process Uptime:', `${process.uptime().toFixed(2)}s`);
    
    // Memory usage
    const memUsage = process.memoryUsage();
    console.log('💾 Memory Usage:', {
      rss: `${(memUsage.rss / 1024 / 1024).toFixed(2)} MB`,
      heapTotal: `${(memUsage.heapTotal / 1024 / 1024).toFixed(2)} MB`,
      heapUsed: `${(memUsage.heapUsed / 1024 / 1024).toFixed(2)} MB`,
      external: `${(memUsage.external / 1024 / 1024).toFixed(2)} MB`,
    });

    // Environment variables
    console.log('🔧 Configuration:');
    console.log('  - GraphQL Endpoint:', process.env.NEXT_PUBLIC_GQL_ENDPOINT || 'NOT SET');
    console.log('  - Azure AD Tenant ID:', process.env.NEXT_PUBLIC_AZURE_AD_TENANT_ID || 'NOT SET');
    console.log('  - Post Logout URI:', process.env.NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI || 'NOT SET');
    console.log('  - Port:', process.env.PORT || '3000');
    console.log('  - Hostname:', process.env.HOSTNAME || 'localhost');

    // System information
    if (process.env.NODE_ENV === 'production') {
      console.log('🐳 Container Info:');
      console.log('  - Working Directory:', process.cwd());
      console.log('  - User ID:', process.getuid?.() || 'N/A');
      console.log('  - Group ID:', process.getgid?.() || 'N/A');
    }

    // Feature flags and capabilities
    console.log('🎛️ Features:');
    console.log('  - Standalone Output: enabled');
    console.log('  - Trailing Slash: enabled');
    console.log('  - Logging Level: verbose');
    console.log('  - Console Removal: disabled');
    
    console.log('====================================');
    console.log('✅ Server initialization complete');
    console.log('====================================');

    // Log periodic health checks
    this.startHealthCheckLogging();
  }

  private startHealthCheckLogging(): void {
    // Log system health every 5 minutes in production
    if (process.env.NODE_ENV === 'production') {
      setInterval(() => {
        const memUsage = process.memoryUsage();
        console.log('💚 Health Check:', {
          timestamp: new Date().toISOString(),
          uptime: `${(process.uptime() / 60).toFixed(2)} minutes`,
          memoryUsed: `${(memUsage.heapUsed / 1024 / 1024).toFixed(2)} MB`,
          memoryTotal: `${(memUsage.heapTotal / 1024 / 1024).toFixed(2)} MB`,
          pid: process.pid
        });
      }, 5 * 60 * 1000); // 5 minutes
    }
  }

  public logRequest(url: string, method: string, userAgent?: string): void {
    if (typeof window !== 'undefined') return;
    
    console.log('🌐 Incoming Request:', {
      timestamp: new Date().toISOString(),
      method,
      url,
      userAgent: userAgent?.substring(0, 100) || 'unknown',
      pid: process.pid
    });
  }

  public logError(error: Error, context?: string): void {
    console.error('❌ Server Error:', {
      timestamp: new Date().toISOString(),
      context: context || 'unknown',
      message: error.message,
      stack: error.stack,
      pid: process.pid
    });
  }
}

// Initialize and export singleton instance
const startupLogger = StartupLogger.getInstance();

// Auto-log startup when module is imported
startupLogger.logStartup();

export default startupLogger;